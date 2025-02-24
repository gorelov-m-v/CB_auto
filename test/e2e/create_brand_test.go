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

type CreateBrandSuite struct {
	suite.Suite
	config     *config.Config
	database   *repository.Connector
	capService capAPI.CapAPI
	kafka      *kafka.Kafka
}

func (s *CreateBrandSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла.", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация http-клиента и CAP API сервиса.", func(sCtx provider.StepCtx) {
		s.capService = factory.InitClient[capAPI.CapAPI](t, s.config, clientTypes.Cap)
	})

	t.WithNewStep("Соединение с базой данных.", func(sCtx provider.StepCtx) {
		connector := repository.OpenConnector(t, &s.config.MySQL, repository.Core)
		s.database = &connector
	})

	t.WithNewStep("Инициализация Kafka consumer.", func(sCtx provider.StepCtx) {
		s.kafka = kafka.NewConsumer(t, s.config, kafka.BrandTopic)
		s.kafka.StartReading(t)
	})
}

func (s *CreateBrandSuite) TestGetBrandByFilters(t provider.T) {
	t.Epic("Brands.")
	t.Feature("Получение бренда по фильтрам.")
	t.Tags("CAP", "Brands", "Platform")
	t.Title("Проверка получения бренда из БД по универсальным фильтрам.")

	brandRepo := brand.NewRepository(s.database.DB(), &s.config.MySQL)

	var testData struct {
		createRequest          *clientTypes.Request[models.CreateCapBrandRequestBody]
		createCapBrandResponse *models.CreateCapBrandResponseBody
	}

	t.WithNewStep("Создание бренда в CAP.", func(sCtx provider.StepCtx) {
		brandName := fmt.Sprintf("test-brand-%s", utils.GenerateAlias())
		names := map[string]string{
			"en": brandName,
		}

		testData.createRequest = &clientTypes.Request[models.CreateCapBrandRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken()),
				"Platform-Locale": "en",
				"Platform-NodeId": s.config.Node.ProjectID,
			},
			Body: &models.CreateCapBrandRequestBody{
				Sort:        1,
				Alias:       brandName,
				Names:       names,
				Description: fmt.Sprintf("Test brand description %s", utils.GenerateAlias()),
			},
		}

		createResp := s.capService.CreateCapBrand(testData.createRequest)
		sCtx.Assert().Equal(http.StatusOK, createResp.StatusCode,
			"Статус ответа при создании бренда должен быть 200 OK")
		testData.createCapBrandResponse = &createResp.Body

		sCtx.WithAttachments(allure.NewAttachment("CreateBrand Request", allure.JSON, utils.CreateHttpAttachRequest(testData.createRequest)))
		sCtx.WithAttachments(allure.NewAttachment("CreateBrand Response", allure.JSON, utils.CreateHttpAttachResponse(createResp)))
	})

	t.WithNewStep("Ожидание создания бренда в БД", func(sCtx provider.StepCtx) {
		err := repository.ExecuteWithRetry(context.Background(), &s.config.MySQL, func(ctx context.Context) error {
			filters := map[string]interface{}{
				"uuid": testData.createCapBrandResponse.ID,
			}
			brandData := brandRepo.GetBrand(t, filters)

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
			sCtx.Assert().Zero(brandData.UpdatedAt, "Время обновления бренда в БД равно нулю")

			sCtx.WithAttachments(allure.NewAttachment("Бренд из БД", allure.JSON, utils.CreatePrettyJSON(brandData)))
			return nil
		})

		sCtx.Assert().NoError(err, "Ошибка при проверке бренда в БД")
	})

	t.WithNewStep("Удаление бренда.", func(sCtx provider.StepCtx) {
		deleteReq := &clientTypes.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken()),
				"Platform-Nodeid": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": testData.createCapBrandResponse.ID,
			},
		}

		deleteResp := s.capService.DeleteCapBrand(deleteReq)

		sCtx.Assert().Equal(http.StatusNoContent, deleteResp.StatusCode, "Статус ответа при удалении бренда в CAP соответствует ожидаемому")

		sCtx.WithAttachments(allure.NewAttachment("DeleteBrand Request", allure.JSON, utils.CreateHttpAttachRequest(deleteReq)))
	})
}

func (s *CreateBrandSuite) AfterAll(t provider.T) {
	t.WithNewStep("Закрытие соединения с Kafka.", func(sCtx provider.StepCtx) {
		if s.kafka != nil {
			s.kafka.Close(t)
		}
	})

	t.WithNewStep("Закрытие соединения с базой данных.", func(sCtx provider.StepCtx) {
		if s.database != nil {
			if err := s.database.Close(); err != nil {
				t.Fatalf("Ошибка при закрытии соединения с базой данных: %v", err)
			}
		}
	})
}

func TestCreateBrandSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(CreateBrandSuite))
}
