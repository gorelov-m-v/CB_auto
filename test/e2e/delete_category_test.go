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

type DeleteCategorySuite struct {
	suite.Suite
	config       *config.Config
	capService   capAPI.CapAPI
	database     *repository.Connector
	categoryRepo *category.Repository
}

func (s *DeleteCategorySuite) BeforeAll(t provider.T) {
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
}

func (s *DeleteCategorySuite) AfterAll(t provider.T) {
	t.WithNewStep("Закрытие соединения с базой данных", func(sCtx provider.StepCtx) {
		if s.database != nil {
			if err := s.database.Close(); err != nil {
				t.Fatalf("Ошибка при закрытии соединения с базой данных: %v", err)
			}
		}
	})
}

func (s *DeleteCategorySuite) TestDeleteCategory(t provider.T) {
	t.Epic("Categories")
	t.Feature("Удаление категории")
	t.Title("Проверка удаления категории")
	t.Description("Создание категории, проверка наличия в БД и её удаление")
	t.Tags("CAP", "Categories", "Positive")

	var categoryID string
	names := map[string]string{
		"en": utils.Get(utils.CATEGORY_TITLE, 20),
		"ru": utils.Get(utils.CATEGORY_TITLE, 20),
	}

	t.WithNewStep("Создание категории", func(sCtx provider.StepCtx) {
		categoryAlias := utils.Get(utils.ALIAS, 10)
		req := &clientTypes.Request[models.CreateCapCategoryRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-Locale": "en",
				"Platform-NodeId": s.config.Node.ProjectID,
			},
			Body: &models.CreateCapCategoryRequestBody{
				Sort:      1,
				Alias:     categoryAlias,
				Names:     names,
				Type:      models.TypeVertical,
				GroupID:   s.config.Node.GroupID,
				ProjectID: s.config.Node.ProjectID,
			},
		}

		resp := s.capService.CreateCapCategory(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, resp.StatusCode, "Категория успешно создана")
		categoryID = resp.Body.ID
		sCtx.Assert().NotEmpty(categoryID, "ID созданной категории не пустой")
	})

	t.WithNewStep("Проверка наличия категории в БД", func(sCtx provider.StepCtx) {
		categoryFromDB, _ := s.categoryRepo.GetCategory(sCtx, map[string]interface{}{
			"uuid": categoryID,
		})

		sCtx.Assert().NotNil(categoryFromDB, "Категория найдена в БД")
		sCtx.Assert().NotEmpty(categoryFromDB.LocalizedNames["en"], "Английское название в БД не пустое")
		sCtx.Assert().NotEmpty(categoryFromDB.LocalizedNames["ru"], "Русское название в БД не пустое")
	})

	t.WithNewStep("Удаление категории", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-NodeId": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": categoryID,
			},
		}

		statusCode := s.capService.DeleteCapCategory(sCtx, req).StatusCode
		sCtx.Assert().Equal(http.StatusNoContent, statusCode, "Категория успешно удалена")
	})

	t.WithNewStep("Проверка удаления категории в БД", func(sCtx provider.StepCtx) {
		categoryFromDB, _ := s.categoryRepo.GetCategory(sCtx, map[string]interface{}{
			"uuid": categoryID,
		})

		sCtx.Assert().Nil(categoryFromDB, "Категория удалена из БД")
	})
}

func TestDeleteCategorySuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(DeleteCategorySuite))
}
