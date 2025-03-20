package test

import (
	"fmt"
	"net/http"
	"testing"

	capAPI "CB_auto/internal/client/cap"
	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/factory"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/internal/repository/brand"
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type CreateBrandParam struct {
	Sort        int
	Alias       string
	Names       map[string]string
	Description string
}

type ParametrizedCreateBrandSuite struct {
	suite.Suite
	config           *config.Config
	capService       capAPI.CapAPI
	database         *repository.Connector
	brandRepo        *brand.Repository
	ParamCreateBrand []CreateBrandParam
}

func (s *ParametrizedCreateBrandSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация HTTP-клиента и CAP API сервиса", func(sCtx provider.StepCtx) {
		s.capService = factory.InitClient[capAPI.CapAPI](sCtx, s.config, clientTypes.Cap)
	})

	t.WithNewStep("Соединение с базой данных", func(sCtx provider.StepCtx) {
		connector := repository.OpenConnector(t, &s.config.MySQL, repository.Core)
		s.database = &connector
		s.brandRepo = brand.NewRepository(s.database.DB(), &s.config.MySQL)
	})

	s.ParamCreateBrand = []CreateBrandParam{
		{
			Sort:  5,
			Alias: utils.Get(utils.ALIAS, 100),
			Names: map[string]string{
				"en": utils.Get(utils.BRAND_TITLE, 100),
				"ru": utils.Get(utils.BRAND_TITLE, 100),
			},
			Description: "Создание бренда (максимальные значения: Alias=100, Name=100)",
		},
		{
			Sort:  10,
			Alias: utils.Get(utils.ALIAS, 99),
			Names: map[string]string{
				"en": utils.Get(utils.BRAND_TITLE, 99),
				"ru": utils.Get(utils.BRAND_TITLE, 99),
			},
			Description: "Создание бренда (граничные значения: Alias=99, Name=99)",
		},
		{
			Sort:  1,
			Alias: utils.Get(utils.ALIAS, 2),
			Names: map[string]string{
				"en": utils.Get(utils.BRAND_TITLE, 2),
				"ru": utils.Get(utils.BRAND_TITLE, 2),
			},
			Description: "Создание бренда (граничные значения: Alias=2, Name=2)",
		},
		{
			Sort:  5,
			Alias: utils.Get(utils.ALIAS, 100),
			Names: map[string]string{
				"ru": utils.Get(utils.BRAND_TITLE, 99),
			},
			Description: "Создание бренда только русского языка (максимальные значения: Alias=100, Name=100)",
		},
		{
			Sort:  10,
			Alias: utils.Get(utils.ALIAS, 99),
			Names: map[string]string{
				"ru": utils.Get(utils.BRAND_TITLE, 99),
			},
			Description: "Создание бренда только русского языка (граничные значения: Alias=99, Name=99)",
		},
		{
			Sort:  1,
			Alias: utils.Get(utils.ALIAS, 2),
			Names: map[string]string{
				"ru": utils.Get(utils.BRAND_TITLE, 2),
			},
			Description: "Создание бренда только русского языка (граничные значения: Alias=2, Name=2)",
		},
	}
}

func (s *ParametrizedCreateBrandSuite) TestCreateBrand(t provider.T) {
	for _, param := range s.ParamCreateBrand {
		t.Run(param.Description, func(t provider.T) {
			t.Epic("Brands")
			t.Feature("Создание бренда")
			t.Title(fmt.Sprintf("Проверка создания бренда: %s", param.Description))
			t.Tags("CAP", "Brands", "Positive")

			var testData struct {
				createBrandRequest  *clientTypes.Request[models.CreateCapBrandRequestBody]
				createBrandResponse *clientTypes.Response[models.CreateCapBrandResponseBody]
			}

			t.WithNewStep("Создание тестового бренда", func(sCtx provider.StepCtx) {
				testData.createBrandRequest = &clientTypes.Request[models.CreateCapBrandRequestBody]{
					Headers: map[string]string{
						"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
						"Platform-NodeId": s.config.Node.ProjectID,
					},
					Body: &models.CreateCapBrandRequestBody{
						Sort:        param.Sort,
						Alias:       param.Alias,
						Names:       param.Names,
						Description: utils.Get(utils.BRAND_TITLE, 50),
					},
				}
				testData.createBrandResponse = s.capService.CreateCapBrand(sCtx, testData.createBrandRequest)
				sCtx.Require().Equal(http.StatusOK, testData.createBrandResponse.StatusCode, "Бренд успешно создан")
				sCtx.Require().NotEmpty(testData.createBrandResponse.Body.ID, "ID созданного бренда не пустой")
			})

			t.WithNewStep("Проверка бренда в БД", func(sCtx provider.StepCtx) {
				brandFromDB := s.brandRepo.GetBrandWithRetry(sCtx, map[string]interface{}{
					"uuid": testData.createBrandResponse.Body.ID,
				})

				sCtx.Assert().NotNil(brandFromDB, "Бренд найден в БД")
				if brandFromDB != nil {
					sCtx.Assert().Equal(testData.createBrandRequest.Body.Names, brandFromDB.LocalizedNames, "Names бренда в БД совпадают с Names в запросе")
					sCtx.Assert().Equal(testData.createBrandRequest.Body.Description, brandFromDB.Description, "Description бренда в БД совпадает с Description в запросе")
					sCtx.Assert().Equal(testData.createBrandRequest.Body.Alias, brandFromDB.Alias, "Alias бренда в БД совпадает с Alias в запросе")
					sCtx.Assert().Equal(testData.createBrandRequest.Body.Sort, brandFromDB.Sort, "Sort бренда в БД совпадает с Sort в запросе")
					sCtx.Assert().Equal(s.config.Node.ProjectID, brandFromDB.NodeUUID, "NodeUUID бренда в БД совпадает с NodeUUID в запросе")
					sCtx.Assert().Equal(models.StatusDisabled, models.StatusType(brandFromDB.Status), "Status бренда в БД равен StatusDisabled")
					sCtx.Assert().NotZero(brandFromDB.CreatedAt, "CreatedAt бренда в БД не равен нулю")
					sCtx.Assert().Zero(brandFromDB.UpdatedAt, "UpdatedAt бренда в БД равен нулю для нового бренда")
					sCtx.Assert().Equal(testData.createBrandResponse.Body.ID, brandFromDB.UUID, "UUID бренда в БД совпадает с UUID в ответе")
				}
			})

			t.WithNewStep("Удаление тестового бренда", func(sCtx provider.StepCtx) {
				req := &clientTypes.Request[struct{}]{
					Headers: map[string]string{
						"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
						"Platform-NodeId": s.config.Node.ProjectID,
					},
					PathParams: map[string]string{
						"id": testData.createBrandResponse.Body.ID,
					},
				}

				resp := s.capService.DeleteCapBrand(sCtx, req)
				sCtx.Assert().Equal(http.StatusNoContent, resp.StatusCode, "Бренд успешно удален")
			})
		})
	}
}

func (s *ParametrizedCreateBrandSuite) AfterAll(t provider.T) {
	t.WithNewStep("Закрытие соединения с базой данных", func(sCtx provider.StepCtx) {
		if s.database != nil {
			if err := s.database.Close(); err != nil {
				t.Errorf("Ошибка при закрытии соединения с базой данных: %v", err)
			}
		}
	})
}

func TestParametrizedCreateBrandSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(ParametrizedCreateBrandSuite))
}
