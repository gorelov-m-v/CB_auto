package test

import (
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
}

type BrandDeletedMessage struct {
	Message struct {
		EventType string `json:"eventType"`
	} `json:"message"`
	Brand struct {
		UUID      string `json:"uuid"`
		DeletedAt int64  `json:"deleted_at"`
	} `json:"brand"`
}

const (
	BrandStatusDeleted = 2
)

func (s *DeleteBrandSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла.", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация http-клиента и CAP API сервиса.", func(sCtx provider.StepCtx) {
		s.capService = factory.InitClient[capAPI.CapAPI](sCtx, s.config, clientTypes.Cap)
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

func (s *DeleteBrandSuite) TestDeleteBrand(t provider.T) {
	t.Epic("Brands.")
	t.Feature("Удаление бренда.")
	t.Tags("CAP", "Brands", "Platform")
	t.Title("Проверка удаления бренда.")

	brandRepo := brand.NewRepository(s.database.DB(), &s.config.MySQL)

	var testData struct {
		createCapBrandRequest *clientTypes.Request[models.CreateCapBrandRequestBody]
		createBrandResponse   *clientTypes.Response[models.CreateCapBrandResponseBody]
		deletedMessage        *BrandDeletedMessage
	}

	t.WithNewStep("Создание бренда.", func(sCtx provider.StepCtx) {
		testData.createCapBrandRequest = &clientTypes.Request[models.CreateCapBrandRequestBody]{
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
		testData.createBrandResponse = s.capService.CreateCapBrand(sCtx, testData.createCapBrandRequest)

		sCtx.Assert().NotEmpty(testData.createBrandResponse.Body.ID, "ID созданного бренда не пустой")
	})

	t.WithNewAsyncStep("Проверка наличия бренда в БД.", func(sCtx provider.StepCtx) {
		brandFromDB := brandRepo.GetBrand(sCtx, map[string]interface{}{
			"uuid": testData.createBrandResponse.Body.ID,
		})

		sCtx.Assert().NotNil(brandFromDB, "Бренд найден в БД")
		sCtx.Assert().Equal(testData.createCapBrandRequest.Body.Alias, brandFromDB.Alias, "Алиас бренда совпадает")
	})

	t.WithNewStep("Удаление бренда.", func(sCtx provider.StepCtx) {
		deleteReq := &clientTypes.Request[struct{}]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", s.capService.GetToken()),
			},
			PathParams: map[string]string{
				"id": testData.createBrandResponse.Body.ID,
			},
		}
		deleteResp := s.capService.DeleteCapBrand(sCtx, deleteReq)

		sCtx.Assert().Equal(http.StatusNoContent, deleteResp.StatusCode, "Статус код ответа равен 204")

		sCtx.WithAttachments(allure.NewAttachment("Delete Request", allure.JSON, utils.CreateHttpAttachRequest(deleteReq)))
		sCtx.WithAttachments(allure.NewAttachment("Delete Response", allure.JSON, utils.CreateHttpAttachResponse(deleteResp)))
	})

	t.WithNewAsyncStep("Проверка удаления бренда в БД.", func(sCtx provider.StepCtx) {
		brandFromDB := brandRepo.GetBrand(sCtx, map[string]interface{}{
			"uuid":   testData.createBrandResponse.Body.ID,
			"status": BrandStatusDeleted,
		})

		sCtx.Assert().NotNil(brandFromDB, "Бренд найден в БД")
		sCtx.Assert().Equal(BrandStatusDeleted, brandFromDB.Status, "Статус бренда - удален")
	})
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
