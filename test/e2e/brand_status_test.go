package test

import (
	"fmt"
	"net/http"
	"testing"

	capAPI "CB_auto/internal/client/cap"
	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/factory"
	"CB_auto/internal/client/types"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type BrandStatusSuite struct {
	suite.Suite
	config     *config.Config
	capService capAPI.CapAPI
}

func (s *BrandStatusSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла.", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация http-клиента и CAP API сервиса.", func(sCtx provider.StepCtx) {
		s.capService = factory.InitClient[capAPI.CapAPI](sCtx, s.config, clientTypes.Cap)
	})
}

func (s *BrandStatusSuite) TestBrandStatusManagement(t provider.T) {
	t.Epic("Brands")
	t.Feature("Управление статусом бренда")
	t.Tags("CAP", "Brands", "Platform", "Status")
	t.Title("Включение и отключение бренда")

	var brandID string

	t.WithNewStep("Создание тестового бренда", func(sCtx provider.StepCtx) {
		brandName := fmt.Sprintf("test-brand-%s", utils.GenerateAlias())
		createRequest := &types.Request[models.CreateCapBrandRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken()),
				"Platform-Nodeid": s.config.Node.ProjectID,
			},
			Body: &models.CreateCapBrandRequestBody{
				Sort:  1,
				Alias: brandName,
				Names: map[string]string{
					"en": brandName,
				},
				Description: "Test brand for status management",
			},
		}

		createResp := s.capService.CreateCapBrand(sCtx, createRequest)
		sCtx.Assert().Equal(http.StatusOK, createResp.StatusCode, "Бренд должен быть успешно создан")
		brandID = createResp.Body.ID

	})

	t.WithNewStep("Включение бренда", func(sCtx provider.StepCtx) {
		updateRequest := &types.Request[models.UpdateBrandStatusRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken()),
				"Platform-Nodeid": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": brandID,
			},
			Body: &models.UpdateBrandStatusRequestBody{
				Status: models.StatusEnabled,
			},
		}

		updateResp := s.capService.UpdateBrandStatus(sCtx, updateRequest)
		sCtx.Assert().Equal(http.StatusNoContent, updateResp.StatusCode, "Статус бренда должен быть успешно обновлен")

		getBrandResp := s.getBrandStatus(sCtx, brandID)
		sCtx.Assert().Equal(models.StatusEnabled, getBrandResp.Body.Status, "Статус бренда должен быть Enabled")
	})

	t.WithNewStep("Отключение бренда", func(sCtx provider.StepCtx) {
		updateRequest := &types.Request[models.UpdateBrandStatusRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken()),
				"Platform-Nodeid": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": brandID,
			},
			Body: &models.UpdateBrandStatusRequestBody{
				Status: models.StatusDisabled,
			},
		}

		updateResp := s.capService.UpdateBrandStatus(sCtx, updateRequest)
		sCtx.Assert().Equal(http.StatusNoContent, updateResp.StatusCode, "Статус бренда должен быть успешно обновлен")

		getBrandResp := s.getBrandStatus(sCtx, brandID)
		sCtx.Assert().Equal(models.StatusDisabled, getBrandResp.Body.Status, "Статус бренда должен быть Disabled")
	})

	t.WithNewStep("Удаление тестового бренда", func(sCtx provider.StepCtx) {
		deleteRequest := &types.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken()),
				"Platform-Nodeid": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": brandID,
			},
		}

		deleteResp := s.capService.DeleteCapBrand(sCtx, deleteRequest)
		sCtx.Assert().Equal(http.StatusNoContent, deleteResp.StatusCode, "Бренд должен быть успешно удален")
	})
}

func (s *BrandStatusSuite) getBrandStatus(sCtx provider.StepCtx, brandID string) *types.Response[models.GetCapBrandResponseBody] {
	getBrandRequest := &types.Request[struct{}]{
		Headers: map[string]string{
			"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken()),
			"Platform-Nodeid": s.config.Node.ProjectID,
		},
		PathParams: map[string]string{
			"id": brandID,
		},
	}

	getBrandResp := s.capService.GetCapBrand(sCtx, getBrandRequest)
	sCtx.Assert().Equal(http.StatusOK, getBrandResp.StatusCode, "Получение бренда должно быть успешным")

	return getBrandResp
}

func (s *BrandStatusSuite) AfterAll(t provider.T) {

}

func TestBrandStatusSuite(t *testing.T) {
	suite.RunSuite(t, new(BrandStatusSuite))
}
