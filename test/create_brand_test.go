package test

import (
	"CB_auto/test/config"
	"CB_auto/test/transport/database"
	"CB_auto/test/transport/database/brand"
	httpClient "CB_auto/test/transport/http"
	capAPI "CB_auto/test/transport/http/cap"
	"CB_auto/test/transport/http/cap/models"
	"CB_auto/test/utils"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"CB_auto/test/transport/kafka"

	"github.com/google/uuid"
	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type CreateBrandSuite struct {
	suite.Suite
	client     *httpClient.Client
	config     *config.Config
	database   *database.Connector
	capService capAPI.CapAPI
	kafka      *kafka.Kafka
}

func (s *CreateBrandSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла.", func(sCtx provider.StepCtx) {
		cfg, err := config.ReadConfig()
		if err != nil {
			t.Fatalf("Ошибка при чтении конфигурации: %v", err)
		}
		s.config = cfg
	})

	t.WithNewStep("Инициализация http-клиента и CAP API сервиса.", func(sCtx provider.StepCtx) {
		client, err := httpClient.InitClient(s.config)
		if err != nil {
			t.Fatalf("InitClient не удался: %v", err)
		}
		s.client = client
		s.capService = capAPI.NewCapClient(s.client)
	})

	t.WithNewStep("Соединение с базой данных.", func(sCtx provider.StepCtx) {
		connector, err := database.OpenConnector(context.Background(), database.Config{
			DriverName:      s.config.MySQL.DriverName,
			DSN:             s.config.MySQL.DSN,
			PingTimeout:     s.config.MySQL.PingTimeout,
			ConnMaxLifetime: s.config.MySQL.ConnMaxLifetime,
			ConnMaxIdleTime: s.config.MySQL.ConnMaxIdleTime,
			MaxOpenConns:    s.config.MySQL.MaxOpenConns,
			MaxIdleConns:    s.config.MySQL.MaxIdleConns,
		})
		if err != nil {
			t.Fatalf("OpenConnector не удался: %v", err)
		}
		s.database = &connector
	})

	t.WithNewStep("Инициализация Kafka utils.", func(sCtx provider.StepCtx) {
		s.kafka = kafka.NewTestUtils(
			[]string{s.config.Kafka.Brokers},
			s.config.Kafka.Topic,
			s.config.GroupID,
		)
	})
}

func (s *CreateBrandSuite) TestSetupSuite(t provider.T) {
	t.Epic("Brands.")
	t.Feature("Создание бренда.")
	t.Tags("CAP", "Brands")
	t.Title("Проверка создания бренда и получения его данных через GET /_cap/api/v1/brands/{id}")

	brandRepo := brand.NewRepository(s.database)

	type TestData struct {
		adminResponse          *models.AdminCheckResponseBody
		createCapBrandResponse *models.CreateCapBrandResponseBody
		createdAlias           string
	}
	var testData TestData

	t.WithNewStep("Получение/Обновление токена авторизации CAP.", func(sCtx provider.StepCtx) {
		adminReq := &httpClient.Request[models.AdminCheckRequestBody]{
			Body: &models.AdminCheckRequestBody{
				UserName: s.config.UserName,
				Password: s.config.Password,
			},
		}
		adminResp, err := s.capService.CheckAdmin(adminReq)
		if err != nil {
			t.Fatalf("CheckAdmin не удался: %v", err)
		}
		testData.adminResponse = &adminResp.Body

		sCtx.WithAttachments(allure.NewAttachment("CheckAdmin Request", allure.JSON, utils.CreateHttpAttachRequest(adminReq)))
		sCtx.WithAttachments(allure.NewAttachment("CheckAdmin Response", allure.JSON, utils.CreateHttpAttachResponse(adminResp)))
	})

	t.WithNewStep("Создание бренда в CAP.", func(sCtx provider.StepCtx) {
		alias := utils.GenerateAlias()
		testData.createdAlias = alias

		createReq := &httpClient.Request[models.CreateCapBrandRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.adminResponse.Token),
				"Platform-Nodeid": s.config.ProjectID,
			},
			Body: &models.CreateCapBrandRequestBody{
				Sort:        1,
				Alias:       alias,
				Names:       map[string]string{"en": utils.GenerateAlias()},
				Description: utils.GenerateAlias(),
			},
		}
		createResp, err := s.capService.CreateCapBrand(createReq)
		if err != nil {
			t.Fatalf("CreateCapBrand не удался: %v", err)
		}
		testData.createCapBrandResponse = &createResp.Body

		sCtx.WithAttachments(allure.NewAttachment("CreateBrand Request", allure.JSON, utils.CreateHttpAttachRequest(createReq)))
		sCtx.WithAttachments(allure.NewAttachment("CreateBrand Response", allure.JSON, utils.CreateHttpAttachResponse(createResp)))
	})

	t.WithNewStep("Проверка сообщения из Kafka.", func(sCtx provider.StepCtx) {
		expectedAlias := testData.createdAlias
		expectedUUID := testData.createCapBrandResponse.ID

		message := s.kafka.ReadMessageWithFilter(t, 10*time.Second, func(brand kafka.Brand) bool {
			return brand.Brand.Alias == expectedAlias && brand.Brand.UUID == expectedUUID
		})

		var kafkaMessage kafka.Brand
		if err := json.Unmarshal(message.Value, &kafkaMessage); err != nil {
			t.Fatalf("Ошибка при парсинге сообщения Kafka: %v", err)
		}

		sCtx.WithAttachments(allure.NewAttachment("Kafka Message", allure.JSON, utils.CreatePrettyJSON(kafkaMessage)))
	})

	t.WithNewStep("Проверка создания бренда в CAP через GET.", func(sCtx provider.StepCtx) {
		getReq := &httpClient.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.adminResponse.Token),
				"Platform-Nodeid": s.config.ProjectID,
			},
			PathParams: map[string]string{
				"id": testData.createCapBrandResponse.ID,
			},
		}
		getResp, err := s.capService.GetCapBrand(getReq)
		if err != nil {
			t.Fatalf("GetCapBrand не удался: %v", err)
		}
		if testData.createdAlias != getResp.Body.Alias {
			t.Fatalf("Ожидали alias %s, получили %s", testData.createdAlias, getResp.Body.Alias)
		}

		sCtx.WithAttachments(allure.NewAttachment("GetBrand Request", allure.JSON, utils.CreateHttpAttachRequest(getReq)))
		sCtx.WithAttachments(allure.NewAttachment("GetBrand Response", allure.JSON, utils.CreateHttpAttachResponse(getResp)))
	})

	t.WithNewStep("Проверка записи информации о бренде в БД.", func(sCtx provider.StepCtx) {
		brandUUID := uuid.MustParse(testData.createCapBrandResponse.ID)
		brandData, err := brandRepo.GetBrand(context.Background(), brandUUID)
		if err != nil {
			t.Fatalf("Не удалось получить бренд из БД: %v", err)
		}

		sCtx.WithAttachments(allure.NewAttachment("Бренд из БД", allure.JSON, utils.CreatePrettyJSON(brandData)))
	})
}

func (s *CreateBrandSuite) AfterAll(t provider.T) {
	t.WithNewStep("Закрытие соединения с Kafka.", func(sCtx provider.StepCtx) {
		s.kafka.Close(t)
	})

	t.WithNewStep("Закрытие соединения с базой данных.", func(sCtx provider.StepCtx) {
		if err := s.database.Close(); err != nil {
			t.Fatalf("Ошибка при закрытии соединения с базой данных: %v", err)
		}
	})
}

func TestCreateBrandSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(CreateBrandSuite))
}
