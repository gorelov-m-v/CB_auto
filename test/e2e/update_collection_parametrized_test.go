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
	"CB_auto/internal/repository/category"
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type UpdateCollectionParam struct {
	Sort        int
	Alias       string
	Names       map[string]string
	Type        models.CategoryType
	Status      models.StatusType
	Description string
}

type ParametrizedUpdateCollectionSuite struct {
	suite.Suite
	config                *config.Config
	capService            capAPI.CapAPI
	database              *repository.Connector
	collectionRepo        *category.Repository
	ParamUpdateCollection []UpdateCollectionParam
}

func (s *ParametrizedUpdateCollectionSuite) BeforeAll(t provider.T) {
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
			Description: "Обновление коллекции (максимальные значения: Alias=100, Name=25)",
		},
		{
			Sort:  10,
			Alias: utils.Get(utils.ALIAS, 99),
			Names: map[string]string{
				"en": utils.Get(utils.COLLECTION_TITLE, 24),
				"ru": utils.Get(utils.COLLECTION_TITLE, 24),
			},
			Type:        models.TypeVertical,
			Description: "Обновление коллекции (граничные значения: Alias=99, Name=24)",
		},
		{
			Alias:       utils.Get(utils.ALIAS, 100),
			Description: "Максимальное значение Alias (100), Names не используются",
		},
		{
			Alias:       utils.Get(utils.ALIAS, 99),
			Description: "Граничное значение Alias (99), Names не используются",
		},
		{
			Alias:       utils.Get(utils.ALIAS, 2),
			Description: "Минимальное значение Alias (2), Names не используются",
		},
		{
			Names:       map[string]string{"en": utils.Get(utils.COLLECTION_TITLE, 25), "ru": utils.Get(utils.COLLECTION_TITLE, 25)},
			Description: "Максимальное значение Name (25), Alias не используется",
		},
		{
			Names:       map[string]string{"en": utils.Get(utils.COLLECTION_TITLE, 24), "ru": utils.Get(utils.COLLECTION_TITLE, 24)},
			Description: "Граничное значение Name (24), Alias не используется",
		},
		{
			Names:       map[string]string{"en": utils.Get(utils.COLLECTION_TITLE, 2), "ru": utils.Get(utils.COLLECTION_TITLE, 2)},
			Description: "Минимальное значение Name (2), Alias не используется",
		},
		{
			Status:      models.StatusEnabled,
			Description: "Включение коллекции (status=1)",
		},
		{
			Status:      models.StatusDisabled,
			Description: "Выключение коллекции (status=2)",
		},
	}
}

func (s *ParametrizedUpdateCollectionSuite) TestAll(t provider.T) {
	for _, param := range s.ParamUpdateCollection {
		t.Run(param.Description, func(t provider.T) {
			t.Epic("Collections")
			t.Feature("Редактирование коллекции")
			t.Title(fmt.Sprintf("Проверка обновления коллекции: %s", param.Description))
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
						Sort:  1,
						Alias: utils.Get(utils.ALIAS, 10),
						Names: map[string]string{
							"en": utils.Get(utils.COLLECTION_TITLE, 20),
							"ru": utils.Get(utils.COLLECTION_TITLE, 20),
						},
						Type:      models.TypeVertical,
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
				sCtx.Require().True(statusResp.StatusCode == http.StatusOK, "Коллекция доступна для обновления")
			})

			if param.Status > 0 {
				t.WithNewStep(fmt.Sprintf("Изменение статуса коллекции: %s", param.Description), func(sCtx provider.StepCtx) {
					statusReq := &clientTypes.Request[models.UpdateCapCollectionStatusRequestBody]{
						Headers: map[string]string{
							"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
							"Platform-NodeId": s.config.Node.ProjectID,
						},
						PathParams: map[string]string{
							"id": testData.createCollectionResponse.Body.ID,
						},
						Body: &models.UpdateCapCollectionStatusRequestBody{
							Status: param.Status,
						},
					}

					statusResp := s.capService.UpdateCapCollectionStatus(sCtx, statusReq)
					sCtx.Logf("Ответ от API при обновлении статуса: %+v", statusResp)
					sCtx.Assert().Equal(http.StatusNoContent, statusResp.StatusCode, "Статус коллекции успешно обновлен")

					// Проверка статуса в БД
					collectionFromDB, err := s.collectionRepo.GetCategory(sCtx, map[string]interface{}{
						"uuid": testData.createCollectionResponse.Body.ID,
					})
					sCtx.Logf("Данные коллекции из БД: %+v", collectionFromDB)
					sCtx.Require().NoError(err, "Ошибка при получении коллекции из БД")
					sCtx.Require().NotNil(collectionFromDB, "Коллекция найдена в БД")
					sCtx.Assert().Equal(int16(param.Status), collectionFromDB.StatusID, "Статус коллекции в БД соответствует ожидаемому")
				})
			} else {
				t.WithNewStep(fmt.Sprintf("Обновление коллекции: %s", param.Description), func(sCtx provider.StepCtx) {
					body := &models.UpdateCapCategoryRequestBody{}
					if param.Alias != "" {
						body.Alias = param.Alias
					} else {
						body.Alias = testData.createCollectionRequest.Body.Alias
					}
					if param.Names != nil {
						body.Names = param.Names
					} else {
						body.Names = testData.createCollectionRequest.Body.Names
					}
					body.Type = testData.createCollectionRequest.Body.Type
					if param.Sort > 0 {
						body.Sort = param.Sort
					} else {
						body.Sort = testData.createCollectionRequest.Body.Sort
					}

					updateReq := &clientTypes.Request[models.UpdateCapCategoryRequestBody]{
						Headers: map[string]string{
							"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
							"Platform-NodeId": s.config.Node.ProjectID,
						},
						PathParams: map[string]string{
							"id": testData.createCollectionResponse.Body.ID,
						},
						Body: body,
					}

					var updateResp *clientTypes.Response[models.UpdateCapCategoryResponseBody]
					var lastErr error
					for i := 0; i < 3; i++ {
						updateResp = s.capService.UpdateCapCategory(sCtx, updateReq)
						if updateResp.StatusCode == http.StatusOK {
							break
						}
						lastErr = fmt.Errorf("попытка %d: статус %d", i+1, updateResp.StatusCode)
						sCtx.Logf("Ошибка обновления коллекции: %v", lastErr)

					}

					if updateResp.StatusCode != http.StatusOK {
						sCtx.Logf("Ошибка обновления коллекции: %+v", updateResp)
						sCtx.Logf("Параметры запроса: %+v", param)
						sCtx.Require().NoError(lastErr, "Коллекция должна быть успешно обновлена")
					}
					sCtx.Assert().Equal(http.StatusOK, updateResp.StatusCode, "Коллекция успешно обновлена")
				})
			}

			t.WithNewStep(fmt.Sprintf("Проверка обновления коллекции в БД: %s", param.Description), func(sCtx provider.StepCtx) {
				collectionFromDB, err := s.collectionRepo.GetCategory(sCtx, map[string]interface{}{
					"uuid": testData.createCollectionResponse.Body.ID,
				})
				sCtx.Require().NoError(err, "Ошибка при получении коллекции из БД")
				sCtx.Require().NotNil(collectionFromDB, "Коллекция найдена в БД")

				if param.Sort > 0 {
					sCtx.Assert().Equal(uint32(param.Sort), uint32(collectionFromDB.Sort), "Sort обновлен")
				}
				if param.Alias != "" {
					sCtx.Assert().Equal(param.Alias, collectionFromDB.Alias, "Alias обновлен")
				}
				if param.Names != nil {
					sCtx.Assert().NotEmpty(collectionFromDB.LocalizedNames["en"], "Английское название в БД не пустое")
					sCtx.Assert().NotEmpty(collectionFromDB.LocalizedNames["ru"], "Русское название в БД не пустое")
				}
				if param.Type != "" {
					sCtx.Assert().Equal(string(param.Type), collectionFromDB.Type, "Тип коллекции обновлен")
				}
				if param.Status > 0 {
					sCtx.Assert().Equal(int16(param.Status), collectionFromDB.StatusID, "Статус коллекции обновлен")
				}
			})

			t.WithNewStep("Удаление тестовой коллекции", func(sCtx provider.StepCtx) {
				deleteReq := &clientTypes.Request[struct{}]{
					Headers: map[string]string{
						"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
						"Platform-NodeId": s.config.Node.ProjectID,
					},
					PathParams: map[string]string{
						"id": testData.createCollectionResponse.Body.ID,
					},
				}

				var deleteResp *clientTypes.Response[struct{}]
				var lastErr error
				for i := 0; i < 3; i++ {
					deleteResp = s.capService.DeleteCapCategory(sCtx, deleteReq)
					if deleteResp.StatusCode == http.StatusNoContent {
						break
					}
					lastErr = fmt.Errorf("попытка %d: статус %d", i+1, deleteResp.StatusCode)
					sCtx.Logf("Ошибка удаления коллекции: %v", lastErr)
				}

				if deleteResp.StatusCode != http.StatusNoContent {
					sCtx.Require().NoError(lastErr, "Коллекция должна быть успешно удалена")
				}
				sCtx.Assert().Equal(http.StatusNoContent, deleteResp.StatusCode, "Коллекция успешно удалена")
			})

			t.WithNewStep("Проверка удаления коллекции из БД", func(sCtx provider.StepCtx) {
				isDeleted := false
				for i := 0; i < 5; i++ {
					collection, err := s.collectionRepo.GetCategory(sCtx, map[string]interface{}{"uuid": testData.createCollectionResponse.Body.ID})
					if err != nil {
						sCtx.Logf("Ошибка при проверке удаления коллекции: %v", err)
						continue
					}
					if collection == nil {
						isDeleted = true
						break
					}
					sCtx.Logf("Попытка %d: коллекция все еще существует", i+1)
				}
				sCtx.Require().True(isDeleted, "Коллекция удалена из БД")
			})
		})
	}
}

func (s *ParametrizedUpdateCollectionSuite) AfterAll(t provider.T) {
	t.WithNewStep("Закрытие соединения с базой данных", func(sCtx provider.StepCtx) {
		if s.database != nil {
			if err := s.database.Close(); err != nil {
				t.Errorf("Ошибка при закрытии соединения с базой данных: %v", err)
			}
		}
	})
}

func TestParametrizedUpdateCollectionSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(ParametrizedUpdateCollectionSuite))
}
