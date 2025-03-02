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

type UpdateCollectionSuite struct {
	suite.Suite
	config         *config.Config
	capService     capAPI.CapAPI
	database       *repository.Connector
	collectionRepo *category.Repository
}

func (s *UpdateCollectionSuite) BeforeAll(t provider.T) {
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
}

func (s *UpdateCollectionSuite) AfterAll(t provider.T) {
	t.WithNewStep("Закрытие соединения с базой данных", func(sCtx provider.StepCtx) {
		if s.database != nil {
			if err := s.database.Close(); err != nil {
				t.Fatalf("Ошибка при закрытии соединения с базой данных: %v", err)
			}
		}
	})
}

func (s *UpdateCollectionSuite) TestUpdateCollection(t provider.T) {
	t.Epic("Collections")
	t.Feature("Редактирование коллекции")
	t.Title("Проверка обновления коллекции")
	t.Description("Создание коллекции, её обновление и проверка изменений в БД")
	t.Tags("CAP", "Collections", "Positive")

	var collectionID string
	var updateReq *clientTypes.Request[models.UpdateCapCategoryRequestBody]
	originalNames := map[string]string{
		"en": utils.GenerateAlias(20),
		"ru": utils.GenerateAlias(20),
	}

	updatedNames := map[string]string{
		"en": utils.GenerateAlias(20),
		"ru": utils.GenerateAlias(20),
	}

	t.WithNewStep("Создание коллекции", func(sCtx provider.StepCtx) {
		collectionAlias := utils.Get(utils.ALIAS, 10)
		req := &clientTypes.Request[models.CreateCapCategoryRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-Locale": "en",
				"Platform-NodeId": s.config.Node.ProjectID,
			},
			Body: &models.CreateCapCategoryRequestBody{
				Sort:      1,
				Alias:     collectionAlias,
				Names:     originalNames,
				Type:      models.TypeHorizontal,
				GroupID:   s.config.Node.GroupID,
				ProjectID: s.config.Node.ProjectID,
			},
		}

		resp := s.capService.CreateCapCategory(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, resp.StatusCode, "Коллекция успешно создана")
		collectionID = resp.Body.ID
		sCtx.Assert().NotEmpty(collectionID, "ID созданной коллекции не пустой")
	})

	t.WithNewStep("Обновление коллекции", func(sCtx provider.StepCtx) {
		updateReq = &clientTypes.Request[models.UpdateCapCategoryRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-NodeId": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": collectionID,
			},
			Body: &models.UpdateCapCategoryRequestBody{
				Sort:  2,
				Alias: fmt.Sprintf("updated-%s", utils.Get(utils.ALIAS, 10)),
				Names: updatedNames,
				Type:  models.TypeHorizontal,
			},
		}

		updateResp := s.capService.UpdateCapCategory(sCtx, updateReq)
		sCtx.Assert().Equal(http.StatusOK, updateResp.StatusCode, "Коллекция успешно обновлена")
	})

	t.WithNewStep("Проверка обновленной коллекции в БД", func(sCtx provider.StepCtx) {
		collectionFromDB := s.collectionRepo.GetCategory(sCtx, map[string]interface{}{
			"uuid": collectionID,
		})

		sCtx.Assert().NotNil(collectionFromDB, "Коллекция найдена в БД")
		sCtx.Assert().Equal(updatedNames, collectionFromDB.LocalizedNames, "Названия в БД обновлены")
		sCtx.Assert().Equal(models.TypeHorizontal, collectionFromDB.Type, "Тип коллекции обновлен")
		sCtx.Assert().Equal(uint32(2), uint32(collectionFromDB.Sort), "Sort обновлен")
		sCtx.Assert().Equal(updateReq.Body.Alias, collectionFromDB.Alias, "Alias обновлен")
		sCtx.Assert().NotEqual(collectionFromDB.UpdatedAt, collectionFromDB.CreatedAt, "Время обновления изменилось")
	})

	t.WithNewStep("Удаление коллекции", func(sCtx provider.StepCtx) {
		deleteReq := &clientTypes.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-NodeId": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": collectionID,
			},
		}

		deleteResp := s.capService.DeleteCapCategory(sCtx, deleteReq)
		sCtx.Assert().Equal(http.StatusNoContent, deleteResp.StatusCode, "Категория успешно удалена")

		collectionFromDB := s.collectionRepo.GetCategory(sCtx, map[string]interface{}{
			"uuid": collectionID,
		})
		sCtx.Assert().Nil(collectionFromDB, "Категория удалена из БД")
	})
}

func TestUpdateCollectionSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(UpdateCollectionSuite))
}
