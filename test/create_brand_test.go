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
		createCapBrandResponse *requests.CreateCapBrandResponseBody
		createCapBrandRequest  *requests.CreateCapBrandRequest
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
			testData.adminResponse, _ = requests.CheckAdmin1(s.client, adminCheckRequest)
			fmt.Println(testData.adminResponse.StatusCode)
			fmt.Println(testData.adminResponse.Body)
		}
	})

	//t.WithNewStep("Создание бренда в CAP: POST /_cap/api/v1/brands", func(sCtx provider.StepCtx) {
	//	testData.createCapBrandRequest = &requests.CreateCapBrandRequest{
	//		Headers: requests.CreateCapBrandRequestHeaders{
	//			Authorization:  "Bearer " + testData.adminResponse.Body.Token,
	//			PlatformNodeID: s.config.ProjectID,
	//		},
	//		Body: requests.CreateCapBrandRequestBody{
	//			Sort:        1,
	//			Alias:       utils.GenerateAlias(),
	//			Names:       map[string]string{"en": utils.GenerateAlias()},
	//			Description: utils.GenerateAlias(),
	//		},
	//	}
	//	testData.createCapBrandResponse, _ = requests.CreateCapBrand(s.client, testData.createCapBrandRequest)
	//
	//	requestAttachment, _ := utils.FormatRequest(testData.createCapBrandRequest, s.config)
	//	responseAttachment, _ := json.Marshal(testData.createCapBrandResponse)
	//	sCtx.WithAttachments(allure.NewAttachment("request", allure.JSON, []byte(requestAttachment)))
	//	sCtx.WithAttachments(allure.NewAttachment("response", allure.JSON, responseAttachment))
	//})

	//t.WithNewStep("Проверка создания бренда в CAP: GET /_cap/api/v1/brands/{uuid}", func(sCtx provider.StepCtx) {
	//	request := requests.GetCapBrandRequest{
	//		PathParams: requests.GetCapBrandRequestPathParams{
	//			UUID: testData.createCapBrandResponse.ID,
	//		},
	//		Headers: requests.GetCapBrandRequestHeaders{
	//			Authorization:  "Bearer " + testData.adminResponse.Body.Token,
	//			PlatformNodeID: s.config.ProjectID,
	//		},
	//	}
	//	getCapBrandResponse, _ := requests.GetCapBrand(s.client, request)
	//
	//	sCtx.Require().Equal(testData.createCapBrandRequest.Body.Alias, getCapBrandResponse.Alias)
	//	responseBody, _ := json.Marshal(getCapBrandResponse)
	//	sCtx.WithAttachments(allure.NewAttachment("Response JSON", allure.JSON, responseBody))
	//})
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
