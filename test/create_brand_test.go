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
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
	"testing"
)

type CreateBrandSuite struct {
	suite.Suite
	client     *httpClient.Client
	config     *config.Config
	database   *database.Connector
	capService capAPI.CapAPI
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
}

func (s *CreateBrandSuite) TestSetupSuite(t provider.T) {
	t.Epic("Brands.")
	t.Feature("Создание бренда.")
	t.Tags("CAP", "Brands")
	t.Title("Проверка создания бренда и получения его данных через GET /_cap/api/v1/brands/{uuid}")

	brandRepo := brand.NewRepository(s.database)

	type TestData struct {
		adminResponse          *models.AdminCheckResponseBody
		createCapBrandResponse *models.CreateCapBrandResponseBody
		createdAlias           string
	}
	var testData TestData

	t.WithNewStep("Получение/Обновление токена авторизации CAP.", func(sCtx provider.StepCtx) {
		if testData.adminResponse == nil || !utils.CheckTokenExpiry(testData.adminResponse) {
			adminBody := models.AdminCheckRequestBody{
				UserName: s.config.UserName,
				Password: s.config.Password,
			}
			adminResp, reqDetails, respDetails, err := s.capService.CheckAdmin(adminBody)
			if err != nil {
				t.Fatalf("CheckAdmin не удался: %v", err)
			}
			testData.adminResponse = adminResp

			sCtx.WithAttachments(allure.NewAttachment("CheckAdmin Request (Full)", allure.JSON, utils.CreateAttach(reqDetails)))
			sCtx.WithAttachments(allure.NewAttachment("CheckAdmin Response (Full)", allure.JSON, utils.CreateAttach(respDetails)))
		}
	})

	t.WithNewStep("Создание бренда в CAP.", func(sCtx provider.StepCtx) {
		alias := utils.GenerateAlias()
		testData.createdAlias = alias

		createBody := models.CreateCapBrandRequestBody{
			Sort:        1,
			Alias:       alias,
			Names:       map[string]string{"en": utils.GenerateAlias()},
			Description: utils.GenerateAlias(),
		}
		headers := models.CreateCapBrandRequestHeaders{
			Authorization:  fmt.Sprintf("Bearer %s", testData.adminResponse.Token),
			PlatformNodeID: s.config.ProjectID,
		}
		createResp, reqDetails, respDetails, err := s.capService.CreateCapBrand(createBody, headers)
		if err != nil {
			t.Fatalf("CreateCapBrand не удался: %v", err)
		}
		testData.createCapBrandResponse = createResp

		sCtx.WithAttachments(allure.NewAttachment("CreateBrand Request (Full)", allure.JSON, utils.CreateAttach(reqDetails)))
		sCtx.WithAttachments(allure.NewAttachment("CreateBrand Response (Full)", allure.JSON, utils.CreateAttach(respDetails)))
	})

	t.WithNewStep("Проверка создания бренда в CAP через GET.", func(sCtx provider.StepCtx) {
		headers := models.GetCapBrandRequestHeaders{
			Authorization:  fmt.Sprintf("Bearer %s", testData.adminResponse.Token),
			PlatformNodeID: s.config.ProjectID,
		}
		getResp, reqDetails, respDetails, err := s.capService.GetCapBrand(testData.createCapBrandResponse.ID, headers)
		if err != nil {
			t.Fatalf("GetCapBrand не удался: %v", err)
		}
		if testData.createdAlias != getResp.Alias {
			t.Fatalf("Ожидали alias %s, получили %s", testData.createdAlias, getResp.Alias)
		}

		sCtx.WithAttachments(allure.NewAttachment("GetBrand Request (Full)", allure.JSON, utils.CreateAttach(reqDetails)))
		sCtx.WithAttachments(allure.NewAttachment("GetBrand Response (Full)", allure.JSON, utils.CreateAttach(respDetails)))
	})

	t.WithNewStep("Проверка записи информации о бренде в БД.", func(sCtx provider.StepCtx) {
		brandUUID := uuid.MustParse(testData.createCapBrandResponse.ID)
		brandData, err := brandRepo.GetBrand(context.Background(), brandUUID)
		if err != nil {
			t.Fatalf("Не удалось получить бренд из БД: %v", err)
		}

		jsonData, _ := json.MarshalIndent(brandData, "", "  ")
		sCtx.WithAttachments(allure.NewAttachment("Бренд из БД", allure.JSON, jsonData))
	})
}

func (s *CreateBrandSuite) AfterAll(t provider.T) {
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
