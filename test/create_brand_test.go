package test

import (
	"CB_auto/test/config"
	"CB_auto/test/transport/database"
	httpClient "CB_auto/test/transport/http"
	"CB_auto/test/transport/http/requests"
	"CB_auto/test/utils"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
	"testing"
)

type CreateBrandSuite struct {
	suite.Suite
	client   *httpClient.Client
	config   *config.Config
	database *sql.DB
}

func (s *CreateBrandSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла.", func(sCtx provider.StepCtx) {
		cfg, _ := config.ReadConfig()
		s.config = cfg
	})
	t.WithNewStep("Инициализация http-клиента.", func(sCtx provider.StepCtx) {
		client, err := httpClient.InitClient(s.config)
		if err != nil {
			t.Fatalf("InitClient Failed: %v", err)
		}
		s.client = client
	})
	t.WithNewStep("Соединение с базой данных.", func(sCtx provider.StepCtx) {
		db, err := database.InitDB(&s.config.MySQL)
		if err != nil {
			t.Fatalf("InitDB Failed: %v", err)
		}
		s.database = db
	})
}

func (s *CreateBrandSuite) TestSetupSuite(t provider.T) {
	t.Epic("Brands.")
	t.Feature("Создание бренда.")
	t.Tags("CAP", "Brands")
	t.Title("Проверка ответа метода GET /_cap/api/v1/brands/{uuid} при создании бренда")

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
}

func (s *CreateBrandSuite) AfterAll(t provider.T) {
	t.WithNewStep("Close Database Connection", func(sCtx provider.StepCtx) {
		if err := database.CloseDB(s.database); err != nil {
			t.Fatalf("Failed to close database connection: %v", err)
		}
	})
}

func TestSetup1Suite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(CreateBrandSuite))
}
