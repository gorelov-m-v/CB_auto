package test

import (
	"CB_auto/test/config"
	"CB_auto/test/transport/database"
	"CB_auto/test/transport/http"
	"CB_auto/test/transport/http/requests"
	"CB_auto/test/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
	"testing"
)

type SetupSuite1 struct {
	suite.Suite
	client   *http.Client
	config   *config.Config
	database *sql.DB
}

func (s *SetupSuite1) BeforeAll(t provider.T) {
	t.Epic("Brands.")
	t.Feature("Создание бренда.")
	t.Title("Создание бренда.")
	t.Description("Проверка содания бренда.")
	t.Tags("BeforeAfter")

	t.WithNewStep("Чтение конфигурационного файла.", func(sCtx provider.StepCtx) {
		cfg, _ := config.ReadConfig()
		s.config = cfg
	})
	t.WithNewStep("Инициализация http-клиента.", func(sCtx provider.StepCtx) {
		client, err := http.InitClient(s.config)
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

func (s *SetupSuite1) TestSetupSuite(t provider.T) {
	var (
		adminResponse          *requests.AdminCheckResponse
		createCapBrandResponse *requests.CreateCapBrandResponse
	)

	t.WithNewStep("Получение/Обновление токена авторизации CAP.", func(sCtx provider.StepCtx) {
		if adminResponse == nil || !utils.CheckTokenExpiry(adminResponse) {
			adminCheckRequest := requests.AdminCheckRequest{
				Body: requests.AdminCheckBody{
					UserName: s.config.UserName,
					Password: s.config.Password,
				},
			}
			adminResponse, _ = requests.CheckAdmin(s.client, adminCheckRequest)
		}
	})
	t.WithNewStep("Получение списка игр в CAP: GET /_cap/api/v1/games", func(sCtx provider.StepCtx) {
		getCapGameListRequest := requests.CapGameListRequest{
			QueryParams: requests.CapGameListQueryParams{
				GroupID:       s.config.GroupID,
				ProjectID:     s.config.ProjectID,
				SortField:     "updatedAt",
				SortDirection: 1,
				Page:          1,
				PerPage:       100,
			},
			Headers: requests.CapGameListHeaders{
				Authorization:  "Bearer " + adminResponse.Token,
				PlatformLocale: "ru",
			},
		}
		getCapGameListResponse, _ := requests.GetCapGameList(s.client, getCapGameListRequest)

		requestAttachment, _ := utils.FormatRequest(getCapGameListRequest, s.config)
		responseAttachment, _ := json.Marshal(getCapGameListResponse)
		sCtx.WithAttachments(allure.NewAttachment("request", allure.JSON, []byte(requestAttachment)))
		sCtx.WithAttachments(allure.NewAttachment("response", allure.JSON, responseAttachment))
	})
	t.WithNewStep("Создание бренда в CAP: POST /_cap/api/v1/brands", func(sCtx provider.StepCtx) {
		createCapBrandRequest := requests.CreateCapBrandRequest{
			Headers: requests.CreateCapBrandHeaders{
				Authorization:  "Bearer " + adminResponse.Token,
				PlatformNodeID: s.config.ProjectID,
			},
			Body: requests.CreateCapBrandBody{
				Sort:        1,
				Alias:       utils.GenerateAlias(),
				Names:       map[string]string{"en": utils.GenerateAlias()},
				Description: utils.GenerateAlias(),
			},
		}
		createCapBrandResponse, _ = requests.CreateCapBrand(s.client, createCapBrandRequest)

		requestAttachment, _ := utils.FormatRequest(createCapBrandRequest, s.config)
		responseAttachment, _ := json.Marshal(createCapBrandResponse)
		sCtx.WithAttachments(allure.NewAttachment("request", allure.JSON, []byte(requestAttachment)))
		sCtx.WithAttachments(allure.NewAttachment("response", allure.JSON, responseAttachment))
	})
	t.WithNewStep("Call getCapBrand", func(sCtx provider.StepCtx) {
		request := requests.GetCapBrandRequest{
			PathParams: requests.GetCapBrandPathParams{
				UUID: createCapBrandResponse.ID,
			},
			Headers: requests.GetCapBrandHeaders{
				Authorization:  "Bearer " + adminResponse.Token,
				PlatformNodeID: s.config.ProjectID,
			},
		}

		response, _ := requests.GetCapBrand(s.client, request)

		responseBody, _ := json.Marshal(response)
		sCtx.WithAttachments(allure.NewAttachment("Response JSON", allure.JSON, responseBody))
		fmt.Println(response)
	})
}

func (s *SetupSuite1) AfterAll(t provider.T) {
	t.WithNewStep("Close Database Connection", func(sCtx provider.StepCtx) {
		if err := database.CloseDB(s.database); err != nil {
			t.Fatalf("Failed to close database connection: %v", err)
		}
	})
}

func TestSetup1Suite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(SetupSuite1))
}
