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

type CategoryPositiveSuite struct {
	suite.Suite
	config       *config.Config
	capService   capAPI.CapAPI
	database     *repository.Connector
	categoryRepo *category.Repository
}

func (s *CategoryPositiveSuite) BeforeAll(t provider.T) {
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

func (s *CategoryPositiveSuite) TestCategoryFields(t provider.T) {
	testCases := []struct {
		name     string
		title    string
		alias    string
		expected int
	}{
		{"Минимальная длина названия", "Ab", utils.Get(utils.ALIAS, 10), http.StatusOK},
		{"Максимальная длина названия", utils.Get(utils.CATEGORY_TITLE, 2), utils.Get(utils.ALIAS, 10), http.StatusOK},
		{"Минимальная длина alias", utils.Get(utils.CATEGORY_TITLE, 2), utils.Get(utils.ALIAS, 10), http.StatusOK},
		{"Максимальная длина alias", utils.Get(utils.CATEGORY_TITLE, 2), utils.Get(utils.ALIAS, 10), http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t provider.T) {
			var categoryID string

			t.WithNewStep("Создание категории", func(sCtx provider.StepCtx) {
				req := &clientTypes.Request[models.CreateCapCategoryRequestBody]{
					Headers: map[string]string{
						"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
						"Platform-NodeId": s.config.Node.ProjectID,
					},
					Body: &models.CreateCapCategoryRequestBody{
						Sort:      1,
						Alias:     tc.alias,
						Names:     map[string]string{"en": tc.title, "ru": tc.title},
						Type:      models.TypeHorizontal,
						GroupID:   s.config.Node.GroupID,
						ProjectID: s.config.Node.ProjectID,
					},
				}

				resp := s.capService.CreateCapCategory(sCtx, req)
				sCtx.Assert().Equal(tc.expected, resp.StatusCode)
				categoryID = resp.Body.ID
			})

			t.WithNewStep("Проверка созданной категории", func(sCtx provider.StepCtx) {
				category, _ := s.categoryRepo.GetCategory(sCtx, map[string]interface{}{
					"uuid": categoryID,
				})

				sCtx.Assert().NotNil(category)
				if category != nil {
					dbNames := category.LocalizedNames
					sCtx.Assert().Equal(tc.title, dbNames["en"])
					sCtx.Assert().Equal(tc.title, dbNames["ru"])
					sCtx.Assert().Equal(tc.alias, category.Alias)
				}
			})

			t.WithNewStep("Удаление тестовой категории", func(sCtx provider.StepCtx) {
				req := &clientTypes.Request[struct{}]{
					Headers: map[string]string{
						"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
						"Platform-NodeId": s.config.Node.ProjectID,
					},
					PathParams: map[string]string{
						"id": categoryID,
					},
				}

				resp := s.capService.DeleteCapCategory(sCtx, req)
				sCtx.Assert().Equal(http.StatusNoContent, resp.StatusCode)
			})
		})
	}
}

func TestCategoryPositiveSuite(t *testing.T) {
	suite.RunSuite(t, new(CategoryPositiveSuite))
}
