package test

import (
	"fmt"
	"net/http"
	"testing"

	// "time"

	capAPI "CB_auto/internal/client/cap"
	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/factory"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/internal/repository/category"
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type CreateCategoryParam struct {
	Sort        int
	Alias       string
	Names       map[string]string
	Type        models.CategoryType
	Description string
}

type ParametrizedCreateCategorySuite struct {
	suite.Suite
	config              *config.Config
	capService          capAPI.CapAPI
	database            *repository.Connector
	CategoryRepo        *category.Repository
	ParamUpdateCategory []UpdateCategoryParam
}

func (s *ParametrizedCreateCategorySuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация HTTP-клиента и CAP API сервиса", func(sCtx provider.StepCtx) {
		s.capService = factory.InitClient[capAPI.CapAPI](sCtx, s.config, clientTypes.Cap)
	})

	t.WithNewStep("Соединение с базой данных", func(sCtx provider.StepCtx) {
		connector := repository.OpenConnector(t, &s.config.MySQL, repository.Core)
		s.database = &connector
		s.CategoryRepo = category.NewRepository(s.database.DB(), &s.config.MySQL)
	})

	s.ParamUpdateCategory = []UpdateCategoryParam{
		{
			Sort:  5,
			Alias: utils.Get(utils.ALIAS, 100),
			Names: map[string]string{
				"en": utils.Get(utils.CATEGORY_TITLE, 25),
				"ru": utils.Get(utils.CATEGORY_TITLE, 25),
			},
			Type:        models.TypeHorizontal,
			Description: "Создание коллекции (максимальные значения: Alias=100, Name=25)",
		},
		{
			Sort:  10,
			Alias: utils.Get(utils.ALIAS, 99),
			Names: map[string]string{
				"en": utils.Get(utils.CATEGORY_TITLE, 24),
				"ru": utils.Get(utils.CATEGORY_TITLE, 24),
			},
			Type:        models.TypeHorizontal,
			Description: "Создание коллекции (граничные значения: Alias=99, Name=24)",
		},
		{
			Sort:  1,
			Alias: utils.Get(utils.ALIAS, 2),
			Names: map[string]string{
				"en": utils.Get(utils.CATEGORY_TITLE, 2),
				"ru": utils.Get(utils.CATEGORY_TITLE, 2),
			},
			Type:        models.TypeHorizontal,
			Description: "Создание коллекции (граничные значения: Alias=2, Name=2)",
		},
		{
			Sort:  5,
			Alias: utils.Get(utils.ALIAS, 100),
			Names: map[string]string{
				"ru": utils.Get(utils.CATEGORY_TITLE, 25),
			},
			Type:        models.TypeHorizontal,
			Description: "Создание коллекции только русского языка (максимальные значения: Alias=100, Name=25)",
		},
		{
			Sort:  10,
			Alias: utils.Get(utils.ALIAS, 99),
			Names: map[string]string{
				"ru": utils.Get(utils.CATEGORY_TITLE, 24),
			},
			Type:        models.TypeHorizontal,
			Description: "Создание коллекции только русского языка (граничные значения: Alias=99, Name=24)",
		},
		{
			Sort:  1,
			Alias: utils.Get(utils.ALIAS, 2),
			Names: map[string]string{
				"ru": utils.Get(utils.CATEGORY_TITLE, 2),
			},
			Type:        models.TypeHorizontal,
			Description: "Создание коллекции только русского языка (граничные значения: Alias=2, Name=2)",
		},
	}
}

func (s *ParametrizedCreateCategorySuite) TestCreateCategory(t provider.T) {
	for _, param := range s.ParamUpdateCategory {
		t.Run(param.Description, func(t provider.T) {
			t.Epic("Categorys")
			t.Feature("Создание коллекции")
			t.Title(fmt.Sprintf("Проверка создания коллекции: %s", param.Description))
			t.Tags("CAP", "Categorys", "Positive")

			var testData struct {
				createCategoryRequest  *clientTypes.Request[models.CreateCapCategoryRequestBody]
				createCategoryResponse *clientTypes.Response[models.CreateCapCategoryResponseBody]
			}

			t.WithNewStep("Создание тестовой коллекции", func(sCtx provider.StepCtx) {
				testData.createCategoryRequest = &clientTypes.Request[models.CreateCapCategoryRequestBody]{
					Headers: map[string]string{
						"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
						"Platform-NodeId": s.config.Node.ProjectID,
					},
					Body: &models.CreateCapCategoryRequestBody{
						Sort:      param.Sort,
						Alias:     param.Alias,
						Names:     param.Names,
						Type:      param.Type,
						GroupID:   s.config.Node.GroupID,
						ProjectID: s.config.Node.ProjectID,
					},
				}
				testData.createCategoryResponse = s.capService.CreateCapCategory(sCtx, testData.createCategoryRequest)
				sCtx.Require().Equal(http.StatusOK, testData.createCategoryResponse.StatusCode, "Коллекция успешно создана")
				sCtx.Require().NotEmpty(testData.createCategoryResponse.Body.ID, "ID созданной коллекции не пустой")
			})

			t.WithNewStep("Ожидание доступности коллекции", func(sCtx provider.StepCtx) {
				statusReq := &clientTypes.Request[struct{}]{
					Headers: map[string]string{
						"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
						"Platform-NodeId": s.config.Node.ProjectID,
					},
					PathParams: map[string]string{
						"id": testData.createCategoryResponse.Body.ID,
					},
				}
				statusResp := s.capService.GetCapCategory(sCtx, statusReq)
				sCtx.Require().True(statusResp.StatusCode == http.StatusOK, "Коллекция доступна для использования")
			})

			t.WithNewStep("Проверка коллекции в БД", func(sCtx provider.StepCtx) {
				CategoryFromDB, err := s.CategoryRepo.GetCategory(sCtx, map[string]interface{}{
					"uuid": testData.createCategoryResponse.Body.ID,
				})
				sCtx.Require().NoError(err, "Ошибка при получении коллекции из БД")
				sCtx.Require().NotNil(CategoryFromDB, "Коллекция найдена в БД")
			})
		})
	}
}

func (s *ParametrizedCreateCategorySuite) AfterAll(t provider.T) {
	t.WithNewStep("Закрытие соединения с базой данных", func(sCtx provider.StepCtx) {
		if s.database != nil {
			if err := s.database.Close(); err != nil {
				t.Errorf("Ошибка при закрытии соединения с базой данных: %v", err)
			}
		}
	})
}

func TestParametrizedCreateCategorySuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(ParametrizedCreateCategorySuite))
}
