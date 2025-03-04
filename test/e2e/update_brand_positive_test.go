package test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	capAPI "CB_auto/internal/client/cap"
	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/factory"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/internal/transport/kafka"
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type UpdateBrandPositiveSuite struct {
	suite.Suite
	config     *config.Config
	database   *repository.Connector
	capService capAPI.CapAPI
	kafka      *kafka.Kafka
}

func (s *UpdateBrandPositiveSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла.", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация http-клиента и CAP API сервиса.", func(sCtx provider.StepCtx) {
		s.capService = factory.InitClient[capAPI.CapAPI](sCtx, s.config, clientTypes.Cap)
	})

	t.WithNewStep("Соединение с базой данных.", func(sCtx provider.StepCtx) {
		connector := repository.OpenConnector(t, &s.config.MySQL, repository.Core)
		s.database = &connector
	})
}

func (s *UpdateBrandPositiveSuite) TestUpdateBrandWithRussianName(t provider.T) {
	var testData struct {
		createRequest          *clientTypes.Request[models.CreateCapBrandRequestBody]
		updateRequest          *clientTypes.Request[models.UpdateCapBrandRequestBody]
		createCapBrandResponse *models.CreateCapBrandResponseBody
		updateCapBrandResponse *models.UpdateCapBrandResponseBody
	}

	t.WithNewStep("Создание тестового бренда", func(sCtx provider.StepCtx) {
		brandName := utils.Get(utils.BRAND_TITLE, 20)
		alias := utils.Get(utils.ALIAS, 20)
		testData.createRequest = &clientTypes.Request[models.CreateCapBrandRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-Nodeid": s.config.Node.ProjectID,
			},
			Body: &models.CreateCapBrandRequestBody{
				Sort:        1,
				Alias:       alias,
				Names:       map[string]string{"en": brandName},
				Description: fmt.Sprintf("Description for %s", brandName),
			},
		}

		createResp := s.capService.CreateCapBrand(sCtx, testData.createRequest)
		sCtx.Assert().Equal(http.StatusOK, createResp.StatusCode)
		testData.createCapBrandResponse = &createResp.Body
	})

	t.WithNewStep("Обновление бренда с русским названием", func(sCtx provider.StepCtx) {
		brandName := utils.Get(utils.BRAND_TITLE, 20)
		alias := utils.Get(utils.ALIAS, 20)
		testData.updateRequest = &clientTypes.Request[models.UpdateCapBrandRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-Nodeid": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": testData.createCapBrandResponse.ID,
			},
			Body: &models.UpdateCapBrandRequestBody{
				Sort:        1,
				Alias:       alias,
				Names:       map[string]string{"ru": brandName},
				Description: fmt.Sprintf("Updated description for %s", brandName),
			},
		}

		updateResp := s.capService.UpdateCapBrand(sCtx, testData.updateRequest)
		sCtx.Assert().Equal(http.StatusOK, updateResp.StatusCode)
		testData.updateCapBrandResponse = &updateResp.Body
	})

	t.WithNewStep("Удаление тестового бренда", func(sCtx provider.StepCtx) {
		deleteReq := &clientTypes.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-Nodeid": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": testData.createCapBrandResponse.ID,
			},
		}
		deleteResp := s.capService.DeleteCapBrand(sCtx, deleteReq)
		sCtx.Assert().Equal(http.StatusNoContent, deleteResp.StatusCode)
	})
}

func (s *UpdateBrandPositiveSuite) TestUpdateBrandWithEnglishName(t provider.T) {
	var testData struct {
		createRequest          *clientTypes.Request[models.CreateCapBrandRequestBody]
		updateRequest          *clientTypes.Request[models.UpdateCapBrandRequestBody]
		createCapBrandResponse *models.CreateCapBrandResponseBody
		updateCapBrandResponse *models.UpdateCapBrandResponseBody
	}

	t.WithNewStep("Создание тестового бренда", func(sCtx provider.StepCtx) {
		brandName := utils.Get(utils.BRAND_TITLE, 20)
		alias := utils.Get(utils.ALIAS, 20)
		testData.createRequest = s.createBrandRequest(sCtx, brandName, alias, map[string]string{
			"en": brandName,
		})

		createResp := s.capService.CreateCapBrand(sCtx, testData.createRequest)
		sCtx.Assert().Equal(http.StatusOK, createResp.StatusCode)
		testData.createCapBrandResponse = &createResp.Body
	})

	t.WithNewStep("Обновление бренда с английским названием", func(sCtx provider.StepCtx) {
		brandName := utils.Get(utils.BRAND_TITLE, 20)
		alias := utils.Get(utils.ALIAS, 20)
		testData.updateRequest = s.updateBrandRequest(sCtx, testData.createCapBrandResponse.ID, brandName, alias, map[string]string{"en": brandName})

		updateResp := s.capService.UpdateCapBrand(sCtx, testData.updateRequest)
		sCtx.Assert().Equal(http.StatusOK, updateResp.StatusCode)
		testData.updateCapBrandResponse = &updateResp.Body
	})

	s.cleanupBrand(t, testData.createCapBrandResponse.ID)
}

func (s *UpdateBrandPositiveSuite) TestUpdateBrandWithMinMaxNames(t provider.T) {
	var testData struct {
		createRequest          *clientTypes.Request[models.CreateCapBrandRequestBody]
		updateRequest          *clientTypes.Request[models.UpdateCapBrandRequestBody]
		createCapBrandResponse *models.CreateCapBrandResponseBody
		updateCapBrandResponse *models.UpdateCapBrandResponseBody
	}

	t.WithNewStep("Создание тестового бренда", func(sCtx provider.StepCtx) {
		brandName := utils.Get(utils.BRAND_TITLE, 20)
		alias := utils.Get(utils.ALIAS, 20)
		testData.createRequest = s.createBrandRequest(sCtx, brandName, alias, map[string]string{
			"en": brandName,
		})

		createResp := s.capService.CreateCapBrand(sCtx, testData.createRequest)
		sCtx.Assert().Equal(http.StatusOK, createResp.StatusCode)
		testData.createCapBrandResponse = &createResp.Body
	})

	t.WithNewStep("Обновление бренда с минимальной длиной имени (2 символа)", func(sCtx provider.StepCtx) {
		timestamp := time.Now().UnixNano()
		suffix := fmt.Sprintf("%x", timestamp)[:2]
		minName := fmt.Sprintf("A%s", suffix)
		minAlias := fmt.Sprintf("min-%s", utils.Get(utils.ALIAS, 20))

		testData.updateRequest = s.updateBrandRequest(sCtx, testData.createCapBrandResponse.ID, minName, minAlias, map[string]string{"en": minName})

		updateResp := s.capService.UpdateCapBrand(sCtx, testData.updateRequest)
		sCtx.Assert().Equal(http.StatusOK, updateResp.StatusCode)
	})

	t.WithNewStep("Обновление бренда с максимальной длиной имени (100 символов)", func(sCtx provider.StepCtx) {
		timestamp := time.Now().UnixNano()

		maxName := fmt.Sprintf("%s%d", strings.Repeat("B", 80), timestamp)
		if len(maxName) > 100 {
			maxName = maxName[:100]
		}
		maxAlias := fmt.Sprintf("max-%s", utils.Get(utils.ALIAS, 20))

		testData.updateRequest = s.updateBrandRequest(sCtx, testData.createCapBrandResponse.ID, maxName, maxAlias, map[string]string{"en": maxName})

		updateResp := s.capService.UpdateCapBrand(sCtx, testData.updateRequest)
		sCtx.Assert().Equal(http.StatusOK, updateResp.StatusCode)
	})

	s.cleanupBrand(t, testData.createCapBrandResponse.ID)
}

func (s *UpdateBrandPositiveSuite) TestUpdateBrandWithMultiLanguage(t provider.T) {
	var testData struct {
		createRequest          *clientTypes.Request[models.CreateCapBrandRequestBody]
		updateRequest          *clientTypes.Request[models.UpdateCapBrandRequestBody]
		createCapBrandResponse *models.CreateCapBrandResponseBody
		updateCapBrandResponse *models.UpdateCapBrandResponseBody
	}

	t.WithNewStep("Создание тестового бренда", func(sCtx provider.StepCtx) {
		brandName := utils.Get(utils.BRAND_TITLE, 20)
		alias := utils.Get(utils.ALIAS, 20)
		testData.createRequest = s.createBrandRequest(sCtx, brandName, alias, map[string]string{
			"en": brandName,
		})

		createResp := s.capService.CreateCapBrand(sCtx, testData.createRequest)
		sCtx.Assert().Equal(http.StatusOK, createResp.StatusCode)
		testData.createCapBrandResponse = &createResp.Body
	})

	t.WithNewStep("Обновление мультиязычного бренда", func(sCtx provider.StepCtx) {
		suffix := utils.Get(utils.ALIAS, 20)
		alias := fmt.Sprintf("multi-lang-%s", suffix)
		testData.updateRequest = s.updateBrandRequest(sCtx, testData.createCapBrandResponse.ID, "Multilingual Brand", alias, map[string]string{
			"en": fmt.Sprintf("Updated Test Brand %s", suffix),
			"ru": fmt.Sprintf("Обновленный тестовый бренд %s", suffix),
			"es": fmt.Sprintf("Marca de prueba actualizada %s", suffix),
		})

		updateResp := s.capService.UpdateCapBrand(sCtx, testData.updateRequest)
		sCtx.Assert().Equal(http.StatusOK, updateResp.StatusCode)
		testData.updateCapBrandResponse = &updateResp.Body
	})

	s.cleanupBrand(t, testData.createCapBrandResponse.ID)
}

func (s *UpdateBrandPositiveSuite) createBrandRequest(sCtx provider.StepCtx, name, alias string, names map[string]string) *clientTypes.Request[models.CreateCapBrandRequestBody] {
	if len(alias) < 2 {
		alias = fmt.Sprintf("%s-%s", alias, utils.Get(utils.ALIAS, 20))
	}

	for lang, localName := range names {
		if len(localName) < 2 {
			names[lang] = fmt.Sprintf("%s-%s", localName, utils.Get(utils.ALIAS, 20))
		}
	}

	return &clientTypes.Request[models.CreateCapBrandRequestBody]{
		Headers: map[string]string{
			"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
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

func (s *UpdateBrandPositiveSuite) updateBrandRequest(sCtx provider.StepCtx, id, name, alias string, names map[string]string) *clientTypes.Request[models.UpdateCapBrandRequestBody] {
	if len(alias) < 2 {
		alias = fmt.Sprintf("%s-%s", alias, utils.Get(utils.ALIAS, 20))
	}

	for lang, localName := range names {
		if len(localName) < 2 {
			names[lang] = fmt.Sprintf("%s-%s", localName, utils.Get(utils.ALIAS, 20))
		}
	}

	return &clientTypes.Request[models.UpdateCapBrandRequestBody]{
		Headers: map[string]string{
			"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
			"Platform-Nodeid": s.config.Node.ProjectID,
		},
		PathParams: map[string]string{
			"id": id,
		},
		Body: &models.UpdateCapBrandRequestBody{
			Sort:        1,
			Alias:       alias,
			Names:       names,
			Description: fmt.Sprintf("Updated description for %s", name),
		},
	}
}

func (s *UpdateBrandPositiveSuite) cleanupBrand(t provider.T, brandID string) {
	t.WithNewStep("Удаление тестового бренда", func(sCtx provider.StepCtx) {
		deleteReq := &clientTypes.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-Nodeid": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": brandID,
			},
		}
		deleteResp := s.capService.DeleteCapBrand(sCtx, deleteReq)
		sCtx.Assert().Equal(http.StatusNoContent, deleteResp.StatusCode)
	})
}

func TestUpdateBrandPositiveSuite(t *testing.T) {
	suite.RunSuite(t, new(UpdateBrandPositiveSuite))
}
