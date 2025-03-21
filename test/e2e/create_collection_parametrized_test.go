package test

import (
	capAPI "CB_auto/internal/client/cap"
	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/factory"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/internal/repository/category"
	"CB_auto/pkg/utils"
	"fmt"
	"net/http"
	"testing"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type CreateCollectionParam struct {
	Sort        int
	Alias       string
	Names       map[string]string
	Type        models.CategoryType
	Description string
}

type ParametrizedCreateCollectionSuite struct {
	suite.Suite
	config                *config.Config
	capService            capAPI.CapAPI
	database              *repository.Connector
	collectionRepo        *category.Repository
	ParamUpdateCollection []UpdateCollectionParam
}

func (s *ParametrizedCreateCollectionSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация HTTP-клиента и CAP API сервиса", func(sCtx provider.StepCtx) {
		s.capService = factory.InitClient[capAPI.CapAPI](sCtx, s.config, clientTypes.Cap)
	})

	t.WithNewStep("Соединение с базой данных", func(sCtx provider.StepCtx) {
		connector := repository.OpenConnector(t, &s.config.MySQL, repository.Core)
		s.database = &connector
		s.collectionRepo = category.NewRepository(s.database.DB(), &s.config.MySQL)
	})

	s.ParamUpdateCollection = []UpdateCollectionParam{
		{
			Sort:  5,
			Alias: utils.Get(utils.ALIAS, 100),
			Names: map[string]string{
				"en": utils.Get(utils.COLLECTION_TITLE, 25),
				"ru": utils.Get(utils.COLLECTION_TITLE, 25),
			},
			Type:        models.TypeVertical,
			Description: "Создание коллекции (максимальные значения: Alias=100, Name=25)",
		},
		{
			Sort:  10,
			Alias: utils.Get(utils.ALIAS, 99),
			Names: map[string]string{
				"en": utils.Get(utils.COLLECTION_TITLE, 24),
				"ru": utils.Get(utils.COLLECTION_TITLE, 24),
			},
			Type:        models.TypeVertical,
			Description: "Создание коллекции (граничные значения: Alias=99, Name=24)",
		},
		{
			Sort:  1,
			Alias: utils.Get(utils.ALIAS, 2),
			Names: map[string]string{
				"en": utils.Get(utils.COLLECTION_TITLE, 2),
				"ru": utils.Get(utils.COLLECTION_TITLE, 2),
			},
			Type:        models.TypeVertical,
			Description: "Создание коллекции (граничные значения: Alias=2, Name=2)",
		},
		{
			Sort:  5,
			Alias: utils.Get(utils.ALIAS, 100),
			Names: map[string]string{
				"ru": utils.Get(utils.COLLECTION_TITLE, 25),
			},
			Type:        models.TypeVertical,
			Description: "Создание коллекции только русского языка (максимальные значения: Alias=100, Name=25)",
		},
		{
			Sort:  10,
			Alias: utils.Get(utils.ALIAS, 99),
			Names: map[string]string{
				"ru": utils.Get(utils.COLLECTION_TITLE, 24),
			},
			Type:        models.TypeVertical,
			Description: "Создание коллекции только русского языка (граничные значения: Alias=99, Name=24)",
		},
		{
			Sort:  1,
			Alias: utils.Get(utils.ALIAS, 2),
			Names: map[string]string{
				"ru": utils.Get(utils.COLLECTION_TITLE, 2),
			},
			Type:        models.TypeVertical,
			Description: "Создание коллекции только русского языка (граничные значения: Alias=2, Name=2)",
		},
	}
}

func (s *ParametrizedCreateCollectionSuite) TestCreateCollection(t provider.T) {
	for _, param := range s.ParamUpdateCollection {
		t.Run(param.Description, func(t provider.T) {
			t.Epic("Collections")
			t.Feature("Создание коллекции")
			t.Title(fmt.Sprintf("Проверка создания коллекции: %s", param.Description))
			t.Tags("CAP", "Collections", "Positive")

			var testData struct {
				createCollectionRequest  *clientTypes.Request[models.CreateCapCategoryRequestBody]
				createCollectionResponse *clientTypes.Response[models.CreateCapCategoryResponseBody]
			}

			t.WithNewStep("Создание тестовой коллекции", func(sCtx provider.StepCtx) {
				testData.createCollectionRequest = &clientTypes.Request[models.CreateCapCategoryRequestBody]{
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
				testData.createCollectionResponse = s.capService.CreateCapCategory(sCtx, testData.createCollectionRequest)
				sCtx.Require().Equal(http.StatusOK, testData.createCollectionResponse.StatusCode, "Коллекция успешно создана")
				sCtx.Require().NotEmpty(testData.createCollectionResponse.Body.ID, "ID созданной коллекции не пустой")
			})

			t.WithNewStep("Ожидание доступности коллекции", func(sCtx provider.StepCtx) {
				statusReq := &clientTypes.Request[struct{}]{
					Headers: map[string]string{
						"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
						"Platform-NodeId": s.config.Node.ProjectID,
					},
					PathParams: map[string]string{
						"id": testData.createCollectionResponse.Body.ID,
					},
				}
				statusResp := s.capService.GetCapCategory(sCtx, statusReq)
				sCtx.Logf("Попытка %d: коллекция еще не доступна, статус: %d", statusResp.StatusCode)
				sCtx.Require().True(statusResp.StatusCode == http.StatusOK, "Коллекция доступна для использования")
			})

			t.WithNewStep("Проверка коллекции в БД", func(sCtx provider.StepCtx) {
				collectionFromDB, err := s.collectionRepo.GetCategory(sCtx, map[string]interface{}{
					"uuid": testData.createCollectionResponse.Body.ID,
				})
				sCtx.Require().NoError(err, "Ошибка при получении коллекции из БД")
				sCtx.Require().NotNil(collectionFromDB, "Коллекция найдена в БД")
			})
		})
	}
}

func (s *ParametrizedCreateCollectionSuite) AfterAll(t provider.T) {
	t.WithNewStep("Закрытие соединения с базой данных", func(sCtx provider.StepCtx) {
		if s.database != nil {
			if err := s.database.Close(); err != nil {
				t.Errorf("Ошибка при закрытии соединения с базой данных: %v", err)
			}
		}
	})
}

func TestParametrizedCreateCollectionSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(ParametrizedCreateCollectionSuite))
}
