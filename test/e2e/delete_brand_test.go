package test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	client "CB_auto/internal/client"
	capAPI "CB_auto/internal/client/cap"
	"CB_auto/internal/client/cap/models"
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
	client     *client.Client
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
		s.client = client.InitClient(t, s.config, client.Cap)
		s.capService = capAPI.NewCapClient(t, s.config, s.client)
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

func generateUniqueName() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("test_brand_%d", rand.Int())
}

func (s *DeleteBrandSuite) TestDeleteBrand(t provider.T) {
	t.Epic("Brands.")
	t.Feature("Удаление бренда.")
	t.Tags("CAP", "Brands", "Platform")
	t.Title("Проверка удаления бренда.")

	brandRepo := brand.NewRepository(s.database.DB(), &s.config.MySQL)

	var testData struct {
		createResponse *models.CreateCapBrandResponseBody
		brandAlias     string
		deletedMessage *BrandDeletedMessage
	}

	t.WithNewStep("Создание бренда.", func(sCtx provider.StepCtx) {
		uniqueAlias := generateUniqueName()
		testData.brandAlias = uniqueAlias
		createReq := &client.Request[models.CreateCapBrandRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken()),
				"Platform-NodeID": s.config.Node.ProjectID,
			},
			Body: &models.CreateCapBrandRequestBody{
				Sort:  1,
				Alias: uniqueAlias,
				Names: map[string]string{
					"en": uniqueAlias,
					"ru": uniqueAlias,
				},
				Description: "Test brand description",
			},
		}
		createResp := s.capService.CreateCapBrand(createReq)
		testData.createResponse = &createResp.Body

		sCtx.Assert().NotEmpty(createResp.Body.ID, "ID созданного бренда не пустой")

		sCtx.WithAttachments(allure.NewAttachment("Create Request", allure.JSON, utils.CreateHttpAttachRequest(createReq)))
		sCtx.WithAttachments(allure.NewAttachment("Create Response", allure.JSON, utils.CreateHttpAttachResponse(createResp)))
	})

	t.WithNewAsyncStep("Проверка наличия бренда в БД.", func(sCtx provider.StepCtx) {
		brandFromDB := brandRepo.GetBrand(t, map[string]interface{}{
			"uuid": testData.createResponse.ID,
		})

		sCtx.Assert().NotNil(brandFromDB, "Бренд найден в БД")
		sCtx.Assert().Equal(testData.brandAlias, brandFromDB.Alias, "Алиас бренда совпадает")

		sCtx.WithAttachments(allure.NewAttachment("Brand DB Data", allure.JSON, utils.CreatePrettyJSON(brandFromDB)))
	})

	t.WithNewStep("Удаление бренда.", func(sCtx provider.StepCtx) {
		deleteReq := &client.Request[struct{}]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", s.capService.GetToken()),
			},
			PathParams: map[string]string{
				"id": testData.createResponse.ID,
			},
		}
		deleteResp := s.capService.DeleteCapBrand(deleteReq)

		sCtx.Assert().Equal(204, deleteResp.StatusCode, "Статус код ответа равен 204")

		sCtx.WithAttachments(allure.NewAttachment("Delete Request", allure.JSON, utils.CreateHttpAttachRequest(deleteReq)))
		sCtx.WithAttachments(allure.NewAttachment("Delete Response", allure.JSON, utils.CreateHttpAttachResponse(deleteResp)))
	})

	t.WithNewAsyncStep("Проверка удаления бренда в БД.", func(sCtx provider.StepCtx) {
		brandFromDB := brandRepo.GetBrand(t, map[string]interface{}{
			"uuid":   testData.createResponse.ID,
			"status": BrandStatusDeleted,
		})

		sCtx.Assert().NotNil(brandFromDB, "Бренд найден в БД")
		sCtx.Assert().Equal(BrandStatusDeleted, brandFromDB.Status, "Статус бренда - удален")

		sCtx.WithAttachments(allure.NewAttachment("Brand DB Data", allure.JSON, utils.CreatePrettyJSON(brandFromDB)))
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
