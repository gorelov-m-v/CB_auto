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

type DeleteCollectionSuite struct {
	suite.Suite
	config         *config.Config
	capService     capAPI.CapAPI
	database       *repository.Connector
	collectionRepo *category.Repository
}

func (s *DeleteCollectionSuite) BeforeAll(t provider.T) {
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

func (s *DeleteCollectionSuite) AfterAll(t provider.T) {
	t.WithNewStep("Закрытие соединения с базой данных", func(sCtx provider.StepCtx) {
		if s.database != nil {
			if err := s.database.Close(); err != nil {
				t.Fatalf("Ошибка при закрытии соединения с базой данных: %v", err)
			}
		}
	})
}

func (s *DeleteCollectionSuite) TestDeleteCollection(t provider.T) {
	t.Epic("Collections")
	t.Feature("Удаление коллекции")
	t.Title("Проверка удаления коллекции")
	t.Description("Создание коллекции, проверка наличия в БД и её удаление")
	t.Tags("CAP", "Collections", "Positive")

	var collectionID string
	names := map[string]string{
		"en": utils.Get(utils.COLLECTION_TITLE, 20),
		"ru": utils.Get(utils.COLLECTION_TITLE, 20),
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
				Names:     names,
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

	t.WithNewStep("Проверка наличия коллекции в БД", func(sCtx provider.StepCtx) {
		collectionFromDB := s.collectionRepo.GetCategory(sCtx, map[string]interface{}{
			"uuid": collectionID,
		})

		sCtx.Assert().NotNil(collectionFromDB, "Коллекция найдена в БД")
		sCtx.Assert().Equal(names, collectionFromDB.LocalizedNames, "Названия в БД совпадают")
	})

	t.WithNewStep("Удаление коллекции", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-NodeId": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": collectionID,
			},
		}

		var resp *clientTypes.Response[struct{}]
		for i := 0; i < 3; i++ {
			resp = s.capService.DeleteCapCategory(sCtx, req)
			if resp.StatusCode == http.StatusNoContent {
				break
			}
		}
		sCtx.Assert().Equal(http.StatusNoContent, resp.StatusCode, "Коллекция успешно удалена")
	})

	t.WithNewStep("Проверка удаления коллекции в БД", func(sCtx provider.StepCtx) {
		collectionFromDB := s.collectionRepo.GetCategory(sCtx, map[string]interface{}{
			"uuid": collectionID,
		})

		sCtx.Assert().Nil(collectionFromDB, "Коллекция удалена из БД")
	})
}

func TestDeleteCollectionSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(DeleteCollectionSuite))
}
