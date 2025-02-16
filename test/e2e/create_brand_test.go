package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	client "CB_auto/internal/client"
	capAPI "CB_auto/internal/client/cap"
	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/config"
	"CB_auto/internal/database"
	"CB_auto/internal/database/brand"
	"CB_auto/internal/transport/kafka"
	"CB_auto/pkg/utils"

	"github.com/google/uuid"
	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type CreateBrandSuite struct {
	suite.Suite
	client     *client.Client
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
		client, err := client.InitClient(s.config, client.Cap)
		if err != nil {
			t.Fatalf("InitClient не удался: %v", err)
		}
		s.client = client
		s.capService = capAPI.NewCapClient(s.client)
	})

	t.WithNewStep("Соединение с базой данных.", func(sCtx provider.StepCtx) {
		connector, err := database.OpenConnector(context.Background(), database.Config{
			DriverName:      s.config.MySQL.DriverName,
			DSN:             s.config.MySQL.DSNCore,
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
			s.config.Kafka.BrandTopic,
			s.config.Node.GroupID,
			s.config.Kafka.GetTimeout(),
		)
		s.kafka.StartReading(t)
	})
}

func (s *CreateBrandSuite) TestSetupSuite(t provider.T) {
	t.Epic("Brands.")
	t.Feature("Создание бренда.")
	t.Tags("CAP", "Brands", "Platform")
	t.Title("Проверка создания бренда.")

	brandRepo := brand.NewRepository(s.database.DB)

	type TestData struct {
		adminResponse          *models.AdminCheckResponseBody
		createRequest          *client.Request[models.CreateCapBrandRequestBody]
		createCapBrandResponse *models.CreateCapBrandResponseBody
	}
	var testData TestData

	t.WithNewStep("Получение/Обновление токена авторизации CAP.", func(sCtx provider.StepCtx) {
		adminReq := &client.Request[models.AdminCheckRequestBody]{
			Body: &models.AdminCheckRequestBody{
				UserName: s.config.HTTP.CapUsername,
				Password: s.config.HTTP.CapPassword,
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

		testData.createRequest = &client.Request[models.CreateCapBrandRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.adminResponse.Token),
				"Platform-Nodeid": s.config.Node.ProjectID,
			},
			Body: &models.CreateCapBrandRequestBody{
				Sort:        1,
				Alias:       brandName,
				Names:       names,
				Description: utils.GenerateAlias(),
			},
		}

		createResp := s.capService.CreateCapBrand(testData.createRequest)
		testData.createCapBrandResponse = &createResp.Body

		sCtx.WithAttachments(allure.NewAttachment("CreateBrand Request", allure.JSON, utils.CreateHttpAttachRequest(testData.createRequest)))
		sCtx.WithAttachments(allure.NewAttachment("CreateBrand Response", allure.JSON, utils.CreateHttpAttachResponse(createResp)))
	})

	t.WithNewAsyncStep("Проверка создания бренда в CAP.", func(sCtx provider.StepCtx) {
		getReq := &client.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.adminResponse.Token),
				"Platform-Nodeid": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": testData.createCapBrandResponse.ID,
			},
		}
		getResp := s.capService.GetCapBrand(getReq)

		sCtx.Assert().Equal(testData.createCapBrandResponse.ID, getResp.Body.ID, "ID бренда в CAP совпадает с ID в ответе")
		sCtx.Assert().Equal(testData.createRequest.Body.Alias, getResp.Body.Alias, "Alias бренда в CAP совпадает с Alias в запросе")
		sCtx.Assert().Equal(testData.createRequest.Body.Names, getResp.Body.Names, "Names бренда в CAP совпадают с Names в запросе")
		sCtx.Assert().Equal(testData.createRequest.Body.Sort, getResp.Body.Sort, "Sort бренда в CAP совпадает с Sort в запросе")
		sCtx.Assert().Equal(s.config.Node.ProjectID, getResp.Body.NodeID, "NodeID бренда в CAP совпадает с NodeID в запросе")
		sCtx.Assert().Equal(models.StatusDisabled, getResp.Body.Status, "Status бренда в CAP совпадает с Status в запросе")

		sCtx.WithAttachments(allure.NewAttachment("GetBrand Request", allure.JSON, utils.CreateHttpAttachRequest(getReq)))
		sCtx.WithAttachments(allure.NewAttachment("GetBrand Response", allure.JSON, utils.CreateHttpAttachResponse(getResp)))
	})

	t.WithNewAsyncStep("Проверка записи информации о бренде в БД.", func(sCtx provider.StepCtx) {
		brandUUID := uuid.MustParse(testData.createCapBrandResponse.ID)
		brandData := brandRepo.GetBrandByUUID(t, brandUUID)
		var dbNames map[string]string = make(map[string]string)
		json.Unmarshal(brandData.LocalizedNames, &dbNames)

		sCtx.Assert().Equal(testData.createRequest.Body.Names, dbNames, "Names бренда в БД совпадают с Names в запросе")
		sCtx.Assert().Equal(testData.createRequest.Body.Alias, brandData.Alias, "Alias бренда в БД совпадает с Alias в запросе")
		sCtx.Assert().Equal(testData.createRequest.Body.Sort, brandData.Sort, "Sort бренда в БД совпадает с Sort в запросе")
		sCtx.Assert().Equal(testData.createRequest.Body.Description, brandData.Description, "Description бренда в БД совпадает с Description в запросе")
		sCtx.Assert().Equal(uuid.MustParse(s.config.Node.ProjectID), brandData.NodeUUID, "NodeUUID бренда в БД совпадает с NodeUUID в запросе")
		sCtx.Assert().Equal(models.StatusDisabled, brandData.Status, "Status бренда в БД совпадает с Status в запросе")
		sCtx.Assert().NotZero(brandData.CreatedAt, "Время создания бренда в БД не равно нулю")
		sCtx.Assert().Zero(brandData.UpdatedAt, "Время обновления бренда в БД равно нулю")

		sCtx.WithAttachments(allure.NewAttachment("Бренд из БД", allure.JSON, utils.CreatePrettyJSON(brandData)))
	})

	t.WithNewAsyncStep("Проверка сообщения из Kafka.", func(sCtx provider.StepCtx) {
		expectedAlias := testData.createRequest.Body.Alias
		expectedUUID := testData.createCapBrandResponse.ID
		expectedNames := testData.createRequest.Body.Names

		message := kafka.FindMessageByFilter[kafka.Brand](s.kafka, t, func(brand kafka.Brand) bool {
			return brand.Brand.UUID == expectedUUID
		})

		brandMessage := kafka.ParseMessage[kafka.Brand](t, message)

		sCtx.Assert().Equal("gambling.gameBrandCreated", brandMessage.Message.EventType, "Тип события в Kafka совпадает с ожидаемым")
		sCtx.Assert().Equal(expectedUUID, brandMessage.Brand.UUID, "UUID бренда в Kafka совпадает с ожидаемым")
		sCtx.Assert().Equal(expectedAlias, brandMessage.Brand.Alias, "Alias бренда в Kafka совпадает с ожидаемым")
		sCtx.Assert().Equal(expectedNames, brandMessage.Brand.LocalizedNames, "LocalizedNames бренда в Kafka совпадают с ожидаемыми")
		sCtx.Assert().False(brandMessage.Brand.StatusEnabled, "Статус бренда в Kafka соответствует ожидаемому")
		sCtx.Assert().Equal(s.config.Node.ProjectID, brandMessage.Brand.ProjectID, "ProjectID бренда в Kafka совпадает с ожидаемым")
		sCtx.Assert().True(utils.IsTimeInRange(brandMessage.Brand.CreatedAt, 30), "Время создания бренда в Kafka находится в ожидаемом диапазоне")

		sCtx.WithAttachments(allure.NewAttachment("Kafka Message", allure.JSON, utils.CreatePrettyJSON(message.Value)))
	})

	t.WithNewStep("Удаление бренда.", func(sCtx provider.StepCtx) {
		deleteReq := &client.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.adminResponse.Token),
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
