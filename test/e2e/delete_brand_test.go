package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	capAPI "CB_auto/internal/client/cap"
	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/factory"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/internal/repository/brand"
	"CB_auto/internal/transport/kafka"
	"CB_auto/pkg/utils"

	"github.com/google/uuid"
	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type DeleteBrandSuite struct {
	suite.Suite
	config     *config.Config
	database   *repository.Connector
	capService capAPI.CapAPI
	kafka      *kafka.Kafka
	brandRepo  *brand.Repository
}

const (
	BrandStatusDeleted = 2
)

func (s *DeleteBrandSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла.", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация http-клиента и CAP API сервиса.", func(sCtx provider.StepCtx) {
		s.capService = factory.InitClient[capAPI.CapAPI](t, s.config, clientTypes.Cap)
	})

	t.WithNewStep("Соединение с базой данных.", func(sCtx provider.StepCtx) {
		connector := repository.OpenConnector(t, &s.config.MySQL, repository.Core)
		s.database = &connector
		s.brandRepo = brand.NewRepository(s.database.DB(), &s.config.MySQL)
	})

	t.WithNewStep("Инициализация Kafka consumer.", func(sCtx provider.StepCtx) {
		s.kafka = kafka.NewConsumer(t, s.config, kafka.BrandTopic)
	})
}

func (s *DeleteBrandSuite) TestDeleteBrand(t provider.T) {
	t.Epic("Brands")
	t.Feature("Удаление бренда")
	t.Title("Проверка удаления бренда")
	t.Tags("CAP", "Brands", "Platform")

	var testData struct {
		createRequest          *clientTypes.Request[models.CreateCapBrandRequestBody]
		createCapBrandResponse *models.CreateCapBrandResponseBody
	}

	t.WithNewStep("Создание тестового бренда", func(sCtx provider.StepCtx) {
		testData.createRequest = &clientTypes.Request[models.CreateCapBrandRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken()),
				"Platform-NodeID": s.config.Node.ProjectID,
			},
			Body: &models.CreateCapBrandRequestBody{
				Sort:  1,
				Alias: utils.GenerateAlias(),
				Names: map[string]string{
					"en": utils.GenerateAlias(),
					"ru": utils.GenerateAlias(),
				},
				Description: "Test brand description",
			},
		}
		createResp := s.capService.CreateCapBrand(testData.createRequest)
		testData.createCapBrandResponse = &createResp.Body
		sCtx.Assert().NotEmpty(testData.createCapBrandResponse.ID, "ID созданного бренда не пустой")
	})

	t.WithNewStep("Проверка создания бренда в БД", func(sCtx provider.StepCtx) {
		err := repository.ExecuteWithRetry(context.Background(), &s.config.MySQL, func(ctx context.Context) error {
			brandData := s.brandRepo.GetBrand(t, map[string]interface{}{
				"uuid": testData.createCapBrandResponse.ID,
			})

			if brandData == nil {
				return fmt.Errorf("бренд не найден в БД")
			}

			var dbNames map[string]string
			if err := json.Unmarshal(brandData.LocalizedNames, &dbNames); err != nil {
				return fmt.Errorf("ошибка при парсинге LocalizedNames: %v", err)
			}

			sCtx.Assert().Equal(testData.createRequest.Body.Names, dbNames, "Names бренда в БД совпадают с Names в запросе")
			sCtx.Assert().Equal(testData.createRequest.Body.Alias, brandData.Alias, "Alias бренда в БД совпадает с Alias в запросе")
			sCtx.Assert().Equal(testData.createRequest.Body.Sort, brandData.Sort, "Sort бренда в БД совпадает с Sort в запросе")
			sCtx.Assert().Equal(testData.createRequest.Body.Description, brandData.Description, "Description бренда в БД совпадает с Description в запросе")
			sCtx.Assert().Equal(uuid.MustParse(s.config.Node.ProjectID), brandData.NodeUUID, "NodeUUID бренда в БД совпадает с NodeUUID в запросе")
			sCtx.Assert().Equal(models.StatusDisabled, brandData.Status, "Status бренда в БД совпадает с Status в запросе")
			sCtx.Assert().NotZero(brandData.CreatedAt, "Время создания бренда в БД не равно нулю")

			sCtx.WithAttachments(allure.NewAttachment("Бренд из БД", allure.JSON, utils.CreatePrettyJSON(brandData)))
			return nil
		})
		sCtx.Assert().NoError(err, "Ошибка при проверке бренда в БД")
	})

	t.WithNewStep("Удаление бренда", func(sCtx provider.StepCtx) {
		s.kafka.StartReading(t)

		deleteReq := &clientTypes.Request[struct{}]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", s.capService.GetToken()),
			},
			PathParams: map[string]string{
				"id": testData.createCapBrandResponse.ID,
			},
		}
		deleteResp := s.capService.DeleteCapBrand(deleteReq)
		sCtx.Assert().Equal(http.StatusNoContent, deleteResp.StatusCode, "Статус код ответа равен 204")

		err := s.kafka.WaitForMessage(t, func(msg []byte) error {
			var event models.BrandEvent
			if err := json.Unmarshal(msg, &event); err != nil {
				return err
			}
			if event.Brand.UUID != testData.createCapBrandResponse.ID {
				return fmt.Errorf("неверный ID бренда в сообщении")
			}
			return nil
		})
		sCtx.Assert().NoError(err, "Получено сообщение о удалении бренда")
	})

	t.WithNewStep("Проверка удаления бренда в БД", func(sCtx provider.StepCtx) {
		err := repository.ExecuteWithRetry(context.Background(), &s.config.MySQL, func(ctx context.Context) error {
			brandData := s.brandRepo.GetBrand(t, map[string]interface{}{
				"uuid": testData.createCapBrandResponse.ID,
			})

			if brandData == nil {
				return fmt.Errorf("бренд не найден в БД")
			}

			sCtx.Assert().Equal(BrandStatusDeleted, brandData.Status, "Статус бренда - удален")
			sCtx.WithAttachments(allure.NewAttachment("Удаленный бренд из БД", allure.JSON, utils.CreatePrettyJSON(brandData)))
			return nil
		})
		sCtx.Assert().NoError(err, "Ошибка при проверке удаления бренда в БД")
	})
}

func (s *DeleteBrandSuite) TestDeleteAlreadyDeletedBrand(t provider.T) {
	t.Epic("Brands")
	t.Feature("Удаление бренда")
	t.Title("Проверка повторного удаления бренда")
	t.Tags("CAP", "Brands", "Platform", "Negative")

}

func (s *DeleteBrandSuite) AfterAll(t provider.T) {
	if s.kafka != nil {
		s.kafka.Close(t)
	}
	if s.database != nil {
		if err := s.database.Close(); err != nil {
			t.Errorf("Ошибка при закрытии соединения с базой данных: %v", err)
		}
	}
}

func TestDeleteBrandSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(DeleteBrandSuite))
}
