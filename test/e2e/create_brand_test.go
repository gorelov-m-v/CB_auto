package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	client "CB_auto/internal/client"
	capAPI "CB_auto/internal/client/cap"
	"CB_auto/internal/client/cap/models"
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
	client     *client.Client
	config     *config.Config
	database   *repository.Connector
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
		connector, err := repository.OpenConnector(&s.config.MySQL, repository.Core)
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

func (s *CreateBrandSuite) TestGetBrandByFilters(t provider.T) {
	t.Epic("Brands.")
	t.Feature("Получение бренда по фильтрам.")
	t.Tags("CAP", "Brands", "Platform")
	t.Title("Проверка получения бренда из БД по универсальным фильтрам.")

	brandRepo := brand.NewRepository(s.database.DB(), &s.config.MySQL)

	var testData struct {
		adminResponse          *models.AdminCheckResponseBody
		createRequest          *client.Request[models.CreateCapBrandRequestBody]
		createCapBrandResponse *models.CreateCapBrandResponseBody
	}

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

	t.WithNewStep("Получение бренда из БД по универсальным фильтрам.", func(sCtx provider.StepCtx) {
		filters := map[string]interface{}{
			"uuid": testData.createCapBrandResponse.ID,
		}
		brandData := brandRepo.GetBrand(t, filters)

		var dbNames map[string]string
		if err := json.Unmarshal(brandData.LocalizedNames, &dbNames); err != nil {
			t.Fatalf("Ошибка при парсинге LocalizedNames: %v", err)
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
