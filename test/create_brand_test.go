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

	t.WithNewStep("Инициализация Kafka consumer.", func(sCtx provider.StepCtx) {
		s.kafka = kafka.NewConsumer(
			[]string{s.config.Kafka.Brokers},
			s.config.Kafka.Topic,
			s.config.GroupID,
			s.config.Kafka.GetTimeout(),
		)
		s.kafka.StartReading(t)
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
		createRequest          *httpClient.Request[models.CreateCapBrandRequestBody]
		createCapBrandResponse *models.CreateCapBrandResponseBody
	}
	var testData TestData

	t.WithNewStep("Получение/Обновление токена авторизации CAP.", func(sCtx provider.StepCtx) {
		adminReq := &httpClient.Request[models.AdminCheckRequestBody]{
			Body: &models.AdminCheckRequestBody{
				UserName: s.config.UserName,
				Password: s.config.Password,
			},
		}
		adminResp := s.capService.CheckAdmin(adminReq)
		testData.adminResponse = &adminResp.Body

		sCtx.WithAttachments(allure.NewAttachment("CheckAdmin Request", allure.JSON, utils.CreateHttpAttachRequest(adminReq)))
		sCtx.WithAttachments(allure.NewAttachment("CheckAdmin Response", allure.JSON, utils.CreateHttpAttachResponse(adminResp)))
	})

	t.WithNewStep("Создание бренда в CAP.", func(sCtx provider.StepCtx) {
		brandName := utils.GenerateAlias()
		names := map[string]string{
			"en": brandName,
		}

		createReq := &httpClient.Request[models.CreateCapBrandRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.adminResponse.Token),
				"Platform-Nodeid": s.config.ProjectID,
			},
			Body: &models.CreateCapBrandRequestBody{
				Sort:        1,
				Alias:       brandName,
				Names:       names,
				Description: utils.GenerateAlias(),
			},
		}
		testData.createRequest = createReq
		createResp := s.capService.CreateCapBrand(createReq)
		testData.createCapBrandResponse = &createResp.Body

		sCtx.WithAttachments(allure.NewAttachment("CreateBrand Request", allure.JSON, utils.CreateHttpAttachRequest(createReq)))
		sCtx.WithAttachments(allure.NewAttachment("CreateBrand Response", allure.JSON, utils.CreateHttpAttachResponse(createResp)))
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
		getResp := s.capService.GetCapBrand(getReq)

		t.Require().Equal(testData.createCapBrandResponse.ID, getResp.Body.ID)
		t.Require().Equal(testData.createRequest.Body.Alias, getResp.Body.Alias)
		t.Require().Equal(testData.createRequest.Body.Names, getResp.Body.Names)
		t.Require().Equal(testData.createRequest.Body.Sort, getResp.Body.Sort)
		t.Require().Equal(s.config.ProjectID, getResp.Body.NodeID)
		t.Require().Equal(models.StatusDisabled, getResp.Body.Status)

		sCtx.WithAttachments(allure.NewAttachment("GetBrand Request", allure.JSON, utils.CreateHttpAttachRequest(getReq)))
		sCtx.WithAttachments(allure.NewAttachment("GetBrand Response", allure.JSON, utils.CreateHttpAttachResponse(getResp)))
	})

	t.WithNewStep("Проверка записи информации о бренде в БД.", func(sCtx provider.StepCtx) {
		brandUUID := uuid.MustParse(testData.createCapBrandResponse.ID)
		brandData, err := brandRepo.GetBrand(context.Background(), brandUUID)
		t.Require().NoError(err, "Ошибка при получении бренда из БД")

		var dbNames map[string]string
		err = json.Unmarshal(brandData.LocalizedNames, &dbNames)
		t.Require().NoError(err)
		t.Require().Equal(testData.createRequest.Body.Names, dbNames)
		t.Require().Equal(testData.createRequest.Body.Alias, brandData.Alias)
		t.Require().Equal(testData.createRequest.Body.Sort, brandData.Sort)
		t.Require().Equal(testData.createRequest.Body.Description, brandData.Description)
		t.Require().Equal(uuid.MustParse(s.config.ProjectID), brandData.NodeUUID)
		t.Require().Equal(models.StatusDisabled, brandData.Status)
		t.Require().NotZero(brandData.CreatedAt)
		t.Require().Zero(brandData.UpdatedAt)

		sCtx.WithAttachments(allure.NewAttachment("Бренд из БД", allure.JSON, utils.CreatePrettyJSON(brandData)))
	})

	t.WithNewStep("Проверка сообщения из Kafka.", func(sCtx provider.StepCtx) {
		expectedAlias := testData.createRequest.Body.Alias
		expectedUUID := testData.createCapBrandResponse.ID
		expectedNames := testData.createRequest.Body.Names

		message := s.kafka.FindMessage(t, func(brand kafka.Brand) bool {
			return brand.Brand.UUID == expectedUUID
		})

		brandMessage := kafka.ParseMessage[kafka.Brand](t, message)

		t.Require().Equal("gambling.gameBrandCreated", brandMessage.Message.EventType)
		t.Require().Equal(expectedUUID, brandMessage.Brand.UUID)
		t.Require().Equal(expectedAlias, brandMessage.Brand.Alias)
		t.Require().Equal(expectedNames, brandMessage.Brand.LocalizedNames)
		t.Require().False(brandMessage.Brand.StatusEnabled)
		t.Require().Equal(s.config.ProjectID, brandMessage.Brand.ProjectID)

		isDateValid, errMsg := utils.IsTimeInRange(brandMessage.Brand.CreatedAt, 5)
		t.Require().True(isDateValid, errMsg)

		sCtx.WithAttachments(allure.NewAttachment("Kafka Message", allure.JSON, utils.CreatePrettyJSON(message.Value)))
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
