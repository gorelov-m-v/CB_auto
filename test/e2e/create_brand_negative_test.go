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
		alias := fmt.Sprintf("test-brand-%s", utils.GenerateAlias())
		req := &clientTypes.Request[models.CreateCapBrandRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken()),
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
		brandName := fmt.Sprintf("Test Brand %s", utils.GenerateAlias())
		req := s.createBrandRequest(brandName, "", map[string]string{
			"en": brandName,
		})

		resp := s.capService.CreateCapBrand(sCtx, req)
		sCtx.Assert().Equal(http.StatusBadRequest, resp.StatusCode)

		s.attachRequestResponse(sCtx, req, resp)
	})
}

func (s *CreateBrandNegativeSuite) TestCreateBrandWithAliasButNoName(t provider.T) {
	t.WithNewStep("Попытка создания бренда с alias но без названия", func(sCtx provider.StepCtx) {
		alias := fmt.Sprintf("test-brand-%s", utils.GenerateAlias())
		req := s.createBrandRequest("", alias, map[string]string{})

		resp := s.capService.CreateCapBrand(sCtx, req)
		sCtx.Assert().Equal(http.StatusBadRequest, resp.StatusCode)

		s.attachRequestResponse(sCtx, req, resp)
	})
}

func (s *CreateBrandNegativeSuite) TestCreateBrandWithNameButNoAlias(t provider.T) {
	t.WithNewStep("Попытка создания бренда с названием но без alias", func(sCtx provider.StepCtx) {
		brandName := fmt.Sprintf("Test Brand %s", utils.GenerateAlias())
		req := s.createBrandRequest(brandName, "", map[string]string{
			"en": brandName,
		})

		resp := s.capService.CreateCapBrand(sCtx, req)
		sCtx.Assert().Equal(http.StatusBadRequest, resp.StatusCode)

		s.attachRequestResponse(sCtx, req, resp)
	})
}

func (s *CreateBrandNegativeSuite) createBrandRequest(name, alias string, names map[string]string) *clientTypes.Request[models.CreateCapBrandRequestBody] {
	return &clientTypes.Request[models.CreateCapBrandRequestBody]{
		Headers: map[string]string{
			"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken()),
			"Platform-Nodeid": s.config.Node.ProjectID,
		},
		Body: &models.CreateCapBrandRequestBody{
			Sort:        1,
			Alias:       alias,
			Names:       names,
			Description: fmt.Sprintf("Description for %s", name),
		},
	}
}

func (s *CreateBrandNegativeSuite) attachRequestResponse(t provider.StepCtx, req *clientTypes.Request[models.CreateCapBrandRequestBody], resp *clientTypes.Response[models.CreateCapBrandResponseBody]) {

}

func TestCreateBrandNegativeSuite(t *testing.T) {
	suite.RunSuite(t, new(CreateBrandNegativeSuite))
}
