package test

import (
	"CB_auto/test/config"
	"CB_auto/test/transport/database"
	"CB_auto/test/transport/database/brand"
	httpClient "CB_auto/test/transport/http"
	"CB_auto/test/transport/http/requests"
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
	client   *httpClient.Client
	config   *config.Config
	database *database.Connector
}

func (s *CreateBrandSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла.", func(sCtx provider.StepCtx) {
		cfg, err := config.ReadConfig()
		if err != nil {
			t.Fatalf("Ошибка при чтении конфигурации: %v", err)
		}
		s.config = cfg
	})

	t.WithNewStep("Инициализация http-клиента.", func(sCtx provider.StepCtx) {
		client, err := httpClient.InitClient(s.config)
		if err != nil {
			t.Fatalf("InitClient не удался: %v", err)
		}
		s.client = client
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
	t.Title("Проверка ответа метода GET /_cap/api/v1/brands/{uuid} при создании бренда")

	brandRepo := brand.NewRepository(s.database)

	type TestData struct {
		adminResponse          *httpClient.Response[requests.AdminCheckResponseBody]
		createCapBrandRequest  *httpClient.Request[requests.CreateCapBrandRequestBody]
		createCapBrandResponse *httpClient.Response[requests.CreateCapBrandResponseBody]
	}

	testData := TestData{
		adminResponse:          nil,
		createCapBrandResponse: nil,
		createCapBrandRequest:  nil,
	}

	t.WithNewStep("Получение/Обновление токена авторизации CAP.", func(sCtx provider.StepCtx) {
		if testData.adminResponse == nil || !utils.CheckTokenExpiry(&testData.adminResponse.Body) {
			adminCheckRequest := &httpClient.Request[requests.AdminCheckRequestBody]{
				Body: &requests.AdminCheckRequestBody{
					UserName: s.config.UserName,
					Password: s.config.Password,
				},
			}

			testData.adminResponse, _ = requests.CheckAdmin(s.client, adminCheckRequest)

			requestAtt := httpClient.FormatRequest(adminCheckRequest)
			responseAtt := httpClient.FormatResponse(testData.adminResponse)
			sCtx.WithAttachments(allure.NewAttachment("response", allure.JSON, requestAtt))
			sCtx.WithAttachments(allure.NewAttachment("response", allure.JSON, responseAtt))
		}
	})

	t.WithNewStep("Создание бренда в CAP: POST /_cap/api/v1/brands", func(sCtx provider.StepCtx) {
		testData.createCapBrandRequest = &httpClient.Request[requests.CreateCapBrandRequestBody]{
			Headers: map[string]string{
				httpClient.AuthorizationHeader: fmt.Sprintf("Bearer %s", testData.adminResponse.Body.Token),
				httpClient.PlatformNodeHeader:  s.config.ProjectID,
			},
			Body: &requests.CreateCapBrandRequestBody{
				Sort:        1,
				Alias:       utils.GenerateAlias(),
				Names:       map[string]string{"en": utils.GenerateAlias()},
				Description: utils.GenerateAlias(),
			},
		}

		testData.createCapBrandResponse, _ = requests.CreateCapBrand(s.client, testData.createCapBrandRequest)

		requestAtt := httpClient.FormatRequest(testData.createCapBrandRequest)
		responseAtt := httpClient.FormatResponse(testData.createCapBrandResponse)
		sCtx.WithAttachments(allure.NewAttachment("request", allure.JSON, requestAtt))
		sCtx.WithAttachments(allure.NewAttachment("response", allure.JSON, responseAtt))
	})

	t.WithNewStep("Проверка создания бренда в CAP: GET /_cap/api/v1/brands/{uuid}", func(sCtx provider.StepCtx) {
		pathParams := map[string]string{
			"UUID": testData.createCapBrandResponse.Body.ID,
		}

		request := &httpClient.Request[requests.GetCapBrandRequestPathParams]{
			PathParams: pathParams,
			Headers: map[string]string{
				httpClient.AuthorizationHeader: fmt.Sprintf("Bearer %s", testData.adminResponse.Body.Token),
				httpClient.PlatformNodeHeader:  s.config.ProjectID,
			},
		}

		getCapBrandResponse, _ := requests.GetCapBrand(s.client, request)
		sCtx.Require().Equal(testData.createCapBrandRequest.Body.Alias, getCapBrandResponse.Body.Alias)

		requestAtt := httpClient.FormatRequest(testData.createCapBrandRequest)
		responseAtt := httpClient.FormatResponse(testData.createCapBrandResponse)
		sCtx.WithAttachments(allure.NewAttachment("request", allure.JSON, requestAtt))
		sCtx.WithAttachments(allure.NewAttachment("response", allure.JSON, responseAtt))
	})

	t.WithNewStep("Проверка записи информации о бренде в БД.", func(sCtx provider.StepCtx) {
		brandUUID := uuid.MustParse(testData.createCapBrandResponse.Body.ID)
		brand, _ := brandRepo.GetBrand(context.Background(), brandUUID)

		jsonData, _ := json.MarshalIndent(brand, "", "  ")

		sCtx.WithAttachments(allure.NewAttachment("result", allure.JSON, jsonData))

	})
}

func (s *CreateBrandSuite) AfterAll(t provider.T) {
	t.WithNewStep("Закрытие соединения с базой данных.", func(sCtx provider.StepCtx) {
		if err := s.database.Close(); err != nil { // Закрываем соединение через метод Close
			t.Fatalf("Ошибка при закрытии соединения с базой данных: %v", err)
		}
	})
}

func TestSetup1Suite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(CreateBrandSuite))
}
