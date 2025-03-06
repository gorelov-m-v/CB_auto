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
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type CreateBrandNegativeSuite struct {
	suite.Suite
	config     *config.Config
	capService capAPI.CapAPI
}

func (s *CreateBrandNegativeSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла.", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация http-клиента и CAP API сервиса.", func(sCtx provider.StepCtx) {
		s.capService = factory.InitClient[capAPI.CapAPI](sCtx, s.config, clientTypes.Cap)
	})
}

func (s *CreateBrandNegativeSuite) TestCreateBrandWithoutName(t provider.T) {
	t.WithNewStep("Подготовка запроса для создания бренда без названия", func(sCtx provider.StepCtx) {
		alias := fmt.Sprintf("test-brand-%s", utils.Get(utils.ALIAS, 20))
		req := &clientTypes.Request[models.CreateCapBrandRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-Nodeid": s.config.Node.ProjectID,
			},
			Body: &models.CreateCapBrandRequestBody{
				Sort:        1,
				Alias:       alias,
				Names:       map[string]string{},
				Description: "Description for empty brand",
			},
		}

		resp := s.capService.CreateCapBrand(sCtx, req)
		sCtx.Assert().Equal(http.StatusBadRequest, resp.StatusCode)
	})
}

func (s *CreateBrandNegativeSuite) TestCreateBrandWithoutAlias(t provider.T) {
	t.WithNewStep("Попытка создания бренда без alias", func(sCtx provider.StepCtx) {
		brandName := fmt.Sprintf("Test Brand %s", utils.Get(utils.BRAND_TITLE, 20))
		req := &clientTypes.Request[models.CreateCapBrandRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-Nodeid": s.config.Node.ProjectID,
			},
			Body: &models.CreateCapBrandRequestBody{
				Sort:        1,
				Alias:       "",
				Names:       map[string]string{"en": brandName},
				Description: fmt.Sprintf("Description for %s", brandName),
			},
		}

		resp := s.capService.CreateCapBrand(sCtx, req)
		sCtx.Assert().Equal(http.StatusBadRequest, resp.StatusCode)
	})
}

func (s *CreateBrandNegativeSuite) TestCreateBrandWithAliasButNoName(t provider.T) {
	t.WithNewStep("Попытка создания бренда с alias но без названия", func(sCtx provider.StepCtx) {
		alias := fmt.Sprintf("test-brand-%s", utils.Get(utils.ALIAS, 20))
		req := &clientTypes.Request[models.CreateCapBrandRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-Nodeid": s.config.Node.ProjectID,
			},
			Body: &models.CreateCapBrandRequestBody{
				Sort:        1,
				Alias:       alias,
				Names:       map[string]string{},
				Description: "Description for brand without name",
			},
		}

		resp := s.capService.CreateCapBrand(sCtx, req)
		sCtx.Assert().Equal(http.StatusBadRequest, resp.StatusCode)
	})
}

func (s *CreateBrandNegativeSuite) TestCreateBrandWithNameButNoAlias(t provider.T) {
	t.WithNewStep("Попытка создания бренда с названием но без alias", func(sCtx provider.StepCtx) {
		brandName := fmt.Sprintf("Test Brand %s", utils.Get(utils.BRAND_TITLE, 20))
		req := &clientTypes.Request[models.CreateCapBrandRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-Nodeid": s.config.Node.ProjectID,
			},
			Body: &models.CreateCapBrandRequestBody{
				Sort:        1,
				Alias:       "",
				Names:       map[string]string{"en": brandName},
				Description: fmt.Sprintf("Description for %s", brandName),
			},
		}

		resp := s.capService.CreateCapBrand(sCtx, req)
		sCtx.Assert().Equal(http.StatusBadRequest, resp.StatusCode)
	})
}

func TestCreateBrandNegativeSuite(t *testing.T) {
	suite.RunSuite(t, new(CreateBrandNegativeSuite))
}
