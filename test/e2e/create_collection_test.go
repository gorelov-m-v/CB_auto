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

type CreateCollectionSuite struct {
	suite.Suite
	config       *config.Config
	capService   capAPI.CapAPI
	database     *repository.Connector
	categoryRepo *category.Repository
}

func (s *CreateCollectionSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла.", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация http-клиента и CAP API сервиса.", func(sCtx provider.StepCtx) {
		s.capService = factory.InitClient[capAPI.CapAPI](sCtx, s.config, clientTypes.Cap)
	})

	t.WithNewStep("Соединение с базой данных.", func(sCtx provider.StepCtx) {
		connector := repository.OpenConnector(t, &s.config.MySQL, repository.Core)
		s.database = &connector
		s.categoryRepo = category.NewRepository(s.database.DB(), &s.config.MySQL)
	})
}

func (s *CreateCollectionSuite) AfterAll(t provider.T) {
	t.WithNewStep("Закрытие соединения с базой данных.", func(sCtx provider.StepCtx) {
		if s.database != nil {
			if err := s.database.Close(); err != nil {
				t.Fatalf("Ошибка при закрытии соединения с базой данных: %v", err)
			}
		}
	})
}

func (s *CreateCollectionSuite) TestCreateCollection(t provider.T) {
	t.Epic("Categories")
	t.Feature("Создание коллекции")
	t.Title("Проверка создания коллекции на двух языках")
	t.Description("Создание коллекции с названиями на русском и английском языках")
	t.Tags("CAP", "Categories", "Positive")

	var categoryID string
	names := map[string]string{
		"en": utils.Get(utils.COLLECTION_TITLE, 20),
		"ru": utils.Get(utils.COLLECTION_TITLE, 20),
	}

	t.WithNewStep("Создание коллекции с русским и английским названием", func(sCtx provider.StepCtx) {
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
				Type:      models.TypeHorizontal,
				GroupID:   s.config.Node.GroupID,
				ProjectID: s.config.Node.ProjectID,
			},
		}

		resp := s.capService.CreateCapCategory(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, resp.StatusCode, "Коллекция успешно создана")
		categoryID = resp.Body.ID
		sCtx.Assert().NotEmpty(categoryID, "ID созданной коллекции не пустой")
	})

	t.WithNewStep("Проверка созданной коллекции", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-NodeId": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": categoryID,
			},
		}

		resp := s.capService.GetCapCategory(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, resp.StatusCode, "Коллекция успешно получена")
		sCtx.Assert().NotEmpty(resp.Body.Names["en"], "Английское название коллекции не пустое")
		sCtx.Assert().NotEmpty(resp.Body.Names["ru"], "Русское название коллекции не пустое")
		sCtx.Assert().Equal(models.TypeHorizontal, resp.Body.Type, "Тип коллекции корректен")
		sCtx.Assert().Equal(s.config.Node.ProjectID, resp.Body.ProjectId, "ProjectID коллекции корректен")
		sCtx.Assert().Equal(1, resp.Body.Sort, "Sort коллекции корректен")
		sCtx.Assert().Equal(models.StatusDisabled, resp.Body.Status, "Status коллекции корректен")
		sCtx.Assert().False(resp.Body.IsDefault, "IsDefault коллекции корректен")
	})

	t.WithNewStep("Проверка коллекции в БД", func(sCtx provider.StepCtx) {
		category := s.categoryRepo.GetCategoryWithRetry(sCtx, map[string]interface{}{
			"uuid": categoryID,
		})

		sCtx.Require().NotNil(category, "Коллекция найдена в БД")
		if category != nil {
			sCtx.Assert().Equal(string(models.TypeHorizontal), category.Type, "Тип коллекции в БД совпадает")
			sCtx.Assert().Equal(uint32(1), uint32(category.Sort), "Sort в БД совпадает")
			sCtx.Assert().Equal(int16(2), int16(category.StatusID), "Status в БД совпадает")
			sCtx.Assert().False(category.IsDefault, "IsDefault в БД совпадает")
			sCtx.Assert().NotEmpty(category.LocalizedNames["en"], "Английское название в БД не пустое")
			sCtx.Assert().NotEmpty(category.LocalizedNames["ru"], "Русское название в БД не пустое")
		}
	})

	t.WithNewStep("Удаление тестовой коллекции", func(sCtx provider.StepCtx) {

		req := &clientTypes.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-NodeId": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": categoryID,
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
}

func TestCreateCollectionSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(CreateCollectionSuite))
}
