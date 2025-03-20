package test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

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

type UpdateCategoryParam struct {
	Sort        int
	Alias       string
	Names       map[string]string
	Type        models.CategoryType
	Status      models.StatusType
	Description string
}

type ParametrizedUpdateCategorySuite struct {
	suite.Suite
	config              *config.Config
	capService          capAPI.CapAPI
	database            *repository.Connector
	categoryRepo        *category.Repository
	ParamUpdateCategory []UpdateCategoryParam
}

func (s *ParametrizedUpdateCategorySuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация HTTP-клиента и CAP API сервиса", func(sCtx provider.StepCtx) {
		s.capService = factory.InitClient[capAPI.CapAPI](sCtx, s.config, clientTypes.Cap)
	})

	t.WithNewStep("Соединение с базой данных", func(sCtx provider.StepCtx) {
		connector := repository.OpenConnector(t, &s.config.MySQL, repository.Core)
		s.database = &connector
		s.categoryRepo = category.NewRepository(s.database.DB(), &s.config.MySQL)
	})

	// Параметры для тестов
	s.ParamUpdateCategory = []UpdateCategoryParam{
		{
			Sort:  5,
			Alias: utils.Get(utils.ALIAS, 100),
			Names: map[string]string{
				"en": utils.Get(utils.CATEGORY_TITLE, 25),
				"ru": utils.Get(utils.CATEGORY_TITLE, 25),
			},
			Type:        models.TypeHorizontal,
			Description: "Обновление коллекции (максимальные значения: Alias=100, Name=25)",
		},
		{
			Sort:  10,
			Alias: utils.Get(utils.ALIAS, 99),
			Names: map[string]string{
				"en": utils.Get(utils.CATEGORY_TITLE, 24),
				"ru": utils.Get(utils.CATEGORY_TITLE, 24),
			},
			Type:        models.TypeHorizontal,
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
			Names:       map[string]string{"en": utils.Get(utils.CATEGORY_TITLE, 25), "ru": utils.Get(utils.CATEGORY_TITLE, 25)},
			Description: "Максимальное значение Name (25), Alias не используется",
		},
		{
			Names:       map[string]string{"en": utils.Get(utils.CATEGORY_TITLE, 24), "ru": utils.Get(utils.CATEGORY_TITLE, 24)},
			Description: "Граничное значение Name (24), Alias не используется",
		},
		{
			Names:       map[string]string{"en": utils.Get(utils.CATEGORY_TITLE, 2), "ru": utils.Get(utils.CATEGORY_TITLE, 2)},
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

func (s *ParametrizedUpdateCategorySuite) TestAll(t provider.T) {
	for _, param := range s.ParamUpdateCategory {
		t.Run(param.Description, func(t provider.T) {
			t.Epic("Categorys")
			t.Feature("Редактирование коллекции")
			t.Title(fmt.Sprintf("Проверка обновления коллекции: %s", param.Description))
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
						Sort:  1,
						Alias: utils.Get(utils.ALIAS, 10),
						Names: map[string]string{
							"en": utils.Get(utils.CATEGORY_TITLE, 20),
							"ru": utils.Get(utils.CATEGORY_TITLE, 20),
						},
						Type:      models.TypeHorizontal,
						GroupID:   s.config.Node.GroupID,
						ProjectID: s.config.Node.ProjectID,
					},
				}
				testData.createCategoryResponse = s.capService.CreateCapCategory(sCtx, testData.createCategoryRequest)
				sCtx.Require().Equal(http.StatusOK, testData.createCategoryResponse.StatusCode, "Коллекция успешно создана")
				sCtx.Require().NotEmpty(testData.createCategoryResponse.Body.ID, "ID созданной коллекции не пустой")
			})

			t.WithNewStep("Ожидание доступности коллекции", func(sCtx provider.StepCtx) {
				isCreated := false
				for i := 0; i < 5; i++ {
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
					if statusResp.StatusCode == http.StatusOK {
						isCreated = true
						break
					}
					sCtx.Logf("Попытка %d: коллекция еще не доступна, статус: %d", i+1, statusResp.StatusCode)
					time.Sleep(time.Second)
				}
				sCtx.Require().True(isCreated, "Коллекция доступна для обновления")
			})

			if param.Status > 0 {
				t.WithNewStep(fmt.Sprintf("Изменение статуса коллекции: %s", param.Description), func(sCtx provider.StepCtx) {
					statusReq := &clientTypes.Request[models.UpdateCapCategoryStatusRequestBody]{
						Headers: map[string]string{
							"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
							"Platform-NodeId": s.config.Node.ProjectID,
						},
						PathParams: map[string]string{
							"id": testData.createCategoryResponse.Body.ID,
						},
						Body: &models.UpdateCapCategoryStatusRequestBody{
							Status: param.Status,
						},
					}

					statusResp := s.capService.UpdateCapCategoryStatus(sCtx, statusReq)
					sCtx.Logf("Ответ от API при обновлении статуса: %+v", statusResp)
					sCtx.Assert().Equal(http.StatusNoContent, statusResp.StatusCode, "Статус категории успешно обновлен")

					// Проверка статуса в БД
					CategoryFromDB, err := s.categoryRepo.GetCategory(sCtx, map[string]interface{}{
						"uuid": testData.createCategoryResponse.Body.ID,
					})
					sCtx.Logf("Данные коллекции из БД: %+v", CategoryFromDB)
					sCtx.Require().NoError(err, "Ошибка при получении коллекции из БД")
					sCtx.Require().NotNil(CategoryFromDB, "Коллекция найдена в БД")
					sCtx.Assert().Equal(int16(param.Status), CategoryFromDB.StatusID, "Статус коллекции в БД соответствует ожидаемому")
				})
			} else {
				t.WithNewStep(fmt.Sprintf("Обновление коллекции: %s", param.Description), func(sCtx provider.StepCtx) {
					body := &models.UpdateCapCategoryRequestBody{}
					if param.Alias != "" {
						body.Alias = param.Alias
					} else {
						body.Alias = testData.createCategoryRequest.Body.Alias
					}
					if param.Names != nil {
						body.Names = param.Names
					} else {
						body.Names = testData.createCategoryRequest.Body.Names
					}
					body.Type = testData.createCategoryRequest.Body.Type
					if param.Sort > 0 {
						body.Sort = param.Sort
					} else {
						body.Sort = testData.createCategoryRequest.Body.Sort
					}

					updateReq := &clientTypes.Request[models.UpdateCapCategoryRequestBody]{
						Headers: map[string]string{
							"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
							"Platform-NodeId": s.config.Node.ProjectID,
						},
						PathParams: map[string]string{
							"id": testData.createCategoryResponse.Body.ID,
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
						time.Sleep(time.Second)
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
				CategoryFromDB, err := s.categoryRepo.GetCategory(sCtx, map[string]interface{}{
					"uuid": testData.createCategoryResponse.Body.ID,
				})
				sCtx.Require().NoError(err, "Ошибка при получении коллекции из БД")
				sCtx.Require().NotNil(CategoryFromDB, "Коллекция найдена в БД")

				if param.Sort > 0 {
					sCtx.Assert().Equal(uint32(param.Sort), uint32(CategoryFromDB.Sort), "Sort обновлен")
				}
				if param.Alias != "" {
					sCtx.Assert().Equal(param.Alias, CategoryFromDB.Alias, "Alias обновлен")
				}
				if param.Names != nil {
					sCtx.Assert().NotEmpty(CategoryFromDB.LocalizedNames["en"], "Английское название в БД не пустое")
					sCtx.Assert().NotEmpty(CategoryFromDB.LocalizedNames["ru"], "Русское название в БД не пустое")
				}
				if param.Type != "" {
					sCtx.Assert().Equal(string(param.Type), CategoryFromDB.Type, "Тип коллекции обновлен")
				}
				if param.Status > 0 {
					sCtx.Assert().Equal(int16(param.Status), CategoryFromDB.StatusID, "Статус коллекции обновлен")
				}
			})

			t.WithNewStep("Удаление тестовой коллекции", func(sCtx provider.StepCtx) {
				deleteReq := &clientTypes.Request[struct{}]{
					Headers: map[string]string{
						"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
						"Platform-NodeId": s.config.Node.ProjectID,
					},
					PathParams: map[string]string{
						"id": testData.createCategoryResponse.Body.ID,
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
					time.Sleep(time.Second)
				}

				if deleteResp.StatusCode != http.StatusNoContent {
					sCtx.Require().NoError(lastErr, "Коллекция должна быть успешно удалена")
				}
				sCtx.Assert().Equal(http.StatusNoContent, deleteResp.StatusCode, "Коллекция успешно удалена")
			})

			t.WithNewStep("Проверка удаления коллекции из БД", func(sCtx provider.StepCtx) {
				isDeleted := false
				for i := 0; i < 5; i++ {
					Category, err := s.categoryRepo.GetCategory(sCtx, map[string]interface{}{"uuid": testData.createCategoryResponse.Body.ID})
					if err != nil {
						sCtx.Logf("Ошибка при проверке удаления коллекции: %v", err)
						time.Sleep(time.Second)
						continue
					}
					if Category == nil {
						isDeleted = true
						break
					}
					sCtx.Logf("Попытка %d: коллекция все еще существует", i+1)
					time.Sleep(time.Second)
				}
				sCtx.Require().True(isDeleted, "Коллекция удалена из БД")
			})
		})
	}
}

func (s *ParametrizedUpdateCategorySuite) AfterAll(t provider.T) {
	t.WithNewStep("Закрытие соединения с базой данных", func(sCtx provider.StepCtx) {
		if s.database != nil {
			if err := s.database.Close(); err != nil {
				t.Errorf("Ошибка при закрытии соединения с базой данных: %v", err)
			}
		}
	})
}

func TestParametrizedUpdateCategorySuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(ParametrizedUpdateCategorySuite))
}
