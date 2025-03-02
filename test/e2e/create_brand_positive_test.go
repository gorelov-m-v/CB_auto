package test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	capAPI "CB_auto/internal/client/cap"
	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/factory"
	"CB_auto/internal/client/types"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/internal/repository/brand"
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type CreateBrandPositiveSuite struct {
	suite.Suite
	config     *config.Config
	capService capAPI.CapAPI
	database   *repository.Connector
	brandRepo  *brand.Repository
}

func (s *CreateBrandPositiveSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла.", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация http-клиента и CAP API сервиса.", func(sCtx provider.StepCtx) {
		s.capService = factory.InitClient[capAPI.CapAPI](sCtx, s.config, clientTypes.Cap)
	})

	t.WithNewStep("Соединение с базой данных.", func(sCtx provider.StepCtx) {
		connector := repository.OpenConnector(t, &s.config.MySQL, repository.Core)
		s.database = &connector
		s.brandRepo = brand.NewRepository(s.database.DB(), &s.config.MySQL)
	})
}

func (s *CreateBrandPositiveSuite) TestCreateBrandWithRussianName(t provider.T) {
	var testData struct {
		createRequest          *clientTypes.Request[models.CreateCapBrandRequestBody]
		createCapBrandResponse *models.CreateCapBrandResponseBody
	}

	t.WithNewStep("Подготовка запроса для создания бренда", func(sCtx provider.StepCtx) {
		brandName := fmt.Sprintf("Тестовый бренд %s", utils.GenerateBrandTitle(20))
		alias := fmt.Sprintf("test-brand-ru-%s", utils.GenerateAlias(20))
		testData.createRequest = &clientTypes.Request[models.CreateCapBrandRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-Nodeid": s.config.Node.ProjectID,
			},
			Body: &models.CreateCapBrandRequestBody{
				Sort:        3,
				Alias:       alias,
				Names:       map[string]string{"ru": brandName},
				Description: fmt.Sprintf("Description for %s", brandName),
			},
		}
	})

	t.WithNewStep("Создание бренда через API", func(sCtx provider.StepCtx) {
		createResp := s.capService.CreateCapBrand(sCtx, testData.createRequest)
		sCtx.Assert().Equal(http.StatusOK, createResp.StatusCode)
		testData.createCapBrandResponse = &createResp.Body

	})

	t.WithNewStep("Проверка создания бренда в БД", func(sCtx provider.StepCtx) {
		brandData := s.brandRepo.GetBrand(sCtx, map[string]interface{}{
			"uuid": testData.createCapBrandResponse.ID,
		})

		sCtx.Assert().NotNil(brandData, "Бренд найден в БД")
		sCtx.Assert().Equal(testData.createRequest.Body.Alias, brandData.Alias, "Alias бренда в БД совпадает с Alias в запросе")
		sCtx.Assert().Equal(testData.createRequest.Body.Sort, brandData.Sort, "Sort бренда в БД совпадает с Sort в запросе")
		sCtx.Assert().Equal(testData.createRequest.Body.Description, brandData.Description, "Description бренда в БД совпадает с Description в запросе")
		sCtx.Assert().Equal(models.StatusDisabled, brandData.Status, "Status бренда в БД совпадает с Status в запросе")
		sCtx.Assert().NotZero(brandData.CreatedAt, "Время создания бренда в БД не равно нулю")
		sCtx.Assert().Zero(brandData.UpdatedAt, "Время обновления бренда в БД равно нулю")

	})

	t.WithNewStep("Очистка тестовых данных", func(sCtx provider.StepCtx) {
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

func (s *CreateBrandPositiveSuite) TestCreateBrandWithEnglishName(t provider.T) {
	var testData struct {
		createRequest          *clientTypes.Request[models.CreateCapBrandRequestBody]
		createCapBrandResponse *models.CreateCapBrandResponseBody
	}

	t.WithNewStep("Создание бренда с английским названием", func(sCtx provider.StepCtx) {
		brandName := fmt.Sprintf("Test Brand %s", utils.GenerateBrandTitle(20))
		alias := fmt.Sprintf("test-brand-en-%s", utils.GenerateBrandTitle(20))
		testData.createRequest = s.createBrandRequest(sCtx, brandName, alias, map[string]string{
			"en": brandName,
		})

		createResp := s.capService.CreateCapBrand(sCtx, testData.createRequest)
		sCtx.Assert().Equal(http.StatusOK, createResp.StatusCode)
		testData.createCapBrandResponse = &createResp.Body

		s.attachRequestResponse(sCtx, testData.createRequest, createResp)

		err := repository.ExecuteWithRetry(sCtx, &s.config.MySQL, func(ctx context.Context) error {
			brandData := s.brandRepo.GetBrand(sCtx, map[string]interface{}{
				"uuid": testData.createCapBrandResponse.ID,
			})
			if brandData == nil {
				return fmt.Errorf("бренд не найден в БД")
			}
			return nil
		})
		sCtx.Assert().NoError(err, "Ошибка при получении бренда из БД")

		s.cleanupBrand(t, testData.createCapBrandResponse.ID)
	})
}

func (s *CreateBrandPositiveSuite) TestCreateBrandWithMinMaxNames(t provider.T) {
	var testData struct {
		createRequest          *clientTypes.Request[models.CreateCapBrandRequestBody]
		createCapBrandResponse *models.CreateCapBrandResponseBody
	}

	t.WithNewStep("Создание бренда с минимальной длиной имени (2 символа)", func(sCtx provider.StepCtx) {
		timestamp := time.Now().UnixNano()
		brandName := fmt.Sprintf("A%d", timestamp)[:2]
		alias := fmt.Sprintf("min-%d", timestamp)
		testData.createRequest = s.createBrandRequest(sCtx, brandName, alias, map[string]string{
			"en": brandName,
		})

		createResp := s.capService.CreateCapBrand(sCtx, testData.createRequest)
		sCtx.Assert().Equal(http.StatusOK, createResp.StatusCode)
		testData.createCapBrandResponse = &createResp.Body

		s.attachRequestResponse(sCtx, testData.createRequest, createResp)

		var brandFromDB *brand.Brand
		for i := 0; i < 3; i++ {
			brandFromDB = s.brandRepo.GetBrand(sCtx, map[string]interface{}{
				"uuid": testData.createCapBrandResponse.ID,
			})
			if brandFromDB != nil {
				break
			}
		}
		sCtx.Assert().NotNil(brandFromDB, "Бренд не найден в БД")

		names := brandFromDB.LocalizedNames
		sCtx.Assert().GreaterOrEqual(len(names["en"]), 2, "Длина имени бренда должна быть не менее 2 символов")

		s.cleanupBrand(t, testData.createCapBrandResponse.ID)
	})

	t.WithNewStep("Создание бренда с максимальной длиной имени (100 символов)", func(sCtx provider.StepCtx) {
		timestamp := time.Now().UnixNano()
		timestampStr := fmt.Sprintf("%d", timestamp)
		remainingLength := 100 - len(timestampStr)
		brandName := fmt.Sprintf("%s%d", strings.Repeat("A", remainingLength), timestamp)

		alias := fmt.Sprintf("max-%d", timestamp)
		testData.createRequest = s.createBrandRequest(sCtx, brandName, alias, map[string]string{
			"en": brandName,
		})

		createResp := s.capService.CreateCapBrand(sCtx, testData.createRequest)
		sCtx.Assert().Equal(http.StatusOK, createResp.StatusCode)
		testData.createCapBrandResponse = &createResp.Body

		s.attachRequestResponse(sCtx, testData.createRequest, createResp)

		var brandFromDB *brand.Brand
		for i := 0; i < 3; i++ {
			brandFromDB = s.brandRepo.GetBrand(sCtx, map[string]interface{}{
				"uuid": testData.createCapBrandResponse.ID,
			})
			if brandFromDB != nil {
				break
			}
		}
		sCtx.Assert().NotNil(brandFromDB, "Бренд не найден в БД")

		names := brandFromDB.LocalizedNames
		sCtx.Assert().LessOrEqual(len(names["en"]), 100, "Длина имени бренда не должна превышать 100 символов")

		s.cleanupBrand(t, testData.createCapBrandResponse.ID)
	})
}

func (s *CreateBrandPositiveSuite) TestCreateBrandWithDifferentAliases(t provider.T) {
	var testData struct {
		createRequest          *clientTypes.Request[models.CreateCapBrandRequestBody]
		createCapBrandResponse *models.CreateCapBrandResponseBody
	}

	aliases := []string{
		"brand-01",
		"testbrand",
		"TEST-BRAND",
		"brand-with-many-dashes",
	}

	for _, aliasBase := range aliases {
		t.WithNewStep(fmt.Sprintf("Создание бренда с alias: %s", aliasBase), func(sCtx provider.StepCtx) {
			brandName := fmt.Sprintf("Test Brand %s", utils.GenerateBrandTitle(20))
			alias := fmt.Sprintf("%s-%s", aliasBase, utils.GenerateAlias(20))
			testData.createRequest = s.createBrandRequest(sCtx, brandName, alias, map[string]string{
				"en": brandName,
			})

			createResp := s.capService.CreateCapBrand(sCtx, testData.createRequest)
			sCtx.Assert().Equal(http.StatusOK, createResp.StatusCode)
			testData.createCapBrandResponse = &createResp.Body

			s.attachRequestResponse(sCtx, testData.createRequest, createResp)

			err := repository.ExecuteWithRetry(sCtx, &s.config.MySQL, func(ctx context.Context) error {
				brandData := s.brandRepo.GetBrand(sCtx, map[string]interface{}{
					"uuid": testData.createCapBrandResponse.ID,
				})

				if brandData == nil {
					return fmt.Errorf("бренд не найден в БД")
				}

				sCtx.Assert().Equal(testData.createRequest.Body.Names, brandData.LocalizedNames, "Names бренда в БД совпадают с Names в запросе")
				sCtx.Assert().Equal(testData.createRequest.Body.Alias, brandData.Alias, "Alias бренда в БД совпадает с Alias в запросе")
				sCtx.Assert().Equal(testData.createRequest.Body.Sort, brandData.Sort, "Sort бренда в БД совпадает с Sort в запросе")
				sCtx.Assert().Equal(testData.createRequest.Body.Description, brandData.Description, "Description бренда в БД совпадает с Description в запросе")
				sCtx.Assert().Equal(s.config.Node.ProjectID, brandData.NodeUUID, "NodeUUID бренда в БД совпадает с NodeUUID в запросе")
				sCtx.Assert().Equal(models.StatusDisabled, brandData.Status, "Status бренда в БД совпадает с Status в запросе")
				sCtx.Assert().NotZero(brandData.CreatedAt, "Время создания бренда в БД не равно нулю")
				sCtx.Assert().Zero(brandData.UpdatedAt, "Время обновления бренда в БД равно нулю")

				return nil
			})
			sCtx.Assert().NoError(err, "Ошибка при проверке бренда в БД")

			s.cleanupBrand(t, testData.createCapBrandResponse.ID)
		})
	}
}

func (s *CreateBrandPositiveSuite) TestCreateBrandWithMultiLanguage(t provider.T) {
	var testData struct {
		createRequest          *clientTypes.Request[models.CreateCapBrandRequestBody]
		createCapBrandResponse *models.CreateCapBrandResponseBody
	}

	t.WithNewStep("Создание мультиязычного бренда", func(sCtx provider.StepCtx) {
		suffix := utils.GenerateAlias(20)
		testData.createRequest = s.createBrandRequest(sCtx, "Multilingual Brand", fmt.Sprintf("multi-lang-%s", suffix), map[string]string{
			"en": fmt.Sprintf("Test Brand %s", suffix),
			"ru": fmt.Sprintf("Тестовый бренд %s", suffix),
			"es": fmt.Sprintf("Marca de prueba %s", suffix),
		})

		createResp := s.capService.CreateCapBrand(sCtx, testData.createRequest)
		sCtx.Assert().Equal(http.StatusOK, createResp.StatusCode)
		testData.createCapBrandResponse = &createResp.Body

		s.attachRequestResponse(sCtx, testData.createRequest, createResp)

		err := repository.ExecuteWithRetry(sCtx, &s.config.MySQL, func(ctx context.Context) error {
			brandData := s.brandRepo.GetBrand(sCtx, map[string]interface{}{
				"uuid": testData.createCapBrandResponse.ID,
			})

			if brandData == nil {
				return fmt.Errorf("бренд не найден в БД")
			}

			sCtx.Assert().Equal(testData.createRequest.Body.Names, brandData.LocalizedNames, "Names бренда в БД совпадают с Names в запросе")
			sCtx.Assert().Equal(testData.createRequest.Body.Alias, brandData.Alias, "Alias бренда в БД совпадает с Alias в запросе")
			sCtx.Assert().Equal(testData.createRequest.Body.Sort, brandData.Sort, "Sort бренда в БД совпадает с Sort в запросе")
			sCtx.Assert().Equal(testData.createRequest.Body.Description, brandData.Description, "Description бренда в БД совпадает с Description в запросе")
			sCtx.Assert().Equal(s.config.Node.ProjectID, brandData.NodeUUID, "NodeUUID бренда в БД совпадает с NodeUUID в запросе")
			sCtx.Assert().Equal(models.StatusDisabled, brandData.Status, "Status бренда в БД совпадает с Status в запросе")
			sCtx.Assert().NotZero(brandData.CreatedAt, "Время создания бренда в БД не равно нулю")
			sCtx.Assert().Zero(brandData.UpdatedAt, "Время обновления бренда в БД равно нулю")

			return nil
		})
		sCtx.Assert().NoError(err, "Ошибка при проверке бренда в БД")

		s.cleanupBrand(t, testData.createCapBrandResponse.ID)
	})
}

func (s *CreateBrandPositiveSuite) TestCreateBrandWithSpecialCharacters(t provider.T) {
	var testData struct {
		createRequest          *clientTypes.Request[models.CreateCapBrandRequestBody]
		createCapBrandResponse *models.CreateCapBrandResponseBody
	}

	t.WithNewStep("Создание бренда со специальными символами", func(sCtx provider.StepCtx) {
		brandName := fmt.Sprintf("Test Brand & Special %s", utils.GenerateBrandTitle(20))
		alias := fmt.Sprintf("test-brand-special-%s", utils.GenerateBrandTitle(20))
		testData.createRequest = s.createBrandRequest(sCtx, brandName, alias, map[string]string{
			"en": brandName,
		})

		createResp := s.capService.CreateCapBrand(sCtx, testData.createRequest)
		sCtx.Assert().Equal(http.StatusOK, createResp.StatusCode)
		testData.createCapBrandResponse = &createResp.Body

		s.attachRequestResponse(sCtx, testData.createRequest, createResp)

		err := repository.ExecuteWithRetry(sCtx, &s.config.MySQL, func(ctx context.Context) error {
			brandData := s.brandRepo.GetBrand(sCtx, map[string]interface{}{
				"uuid": testData.createCapBrandResponse.ID,
			})

			if brandData == nil {
				return fmt.Errorf("бренд не найден в БД")
			}

			sCtx.Assert().Equal(testData.createRequest.Body.Names, brandData.LocalizedNames, "Names бренда в БД совпадают с Names в запросе")
			sCtx.Assert().Equal(testData.createRequest.Body.Alias, brandData.Alias, "Alias бренда в БД совпадает с Alias в запросе")
			sCtx.Assert().Equal(testData.createRequest.Body.Sort, brandData.Sort, "Sort бренда в БД совпадает с Sort в запросе")
			sCtx.Assert().Equal(testData.createRequest.Body.Description, brandData.Description, "Description бренда в БД совпадает с Description в запросе")
			sCtx.Assert().Equal(s.config.Node.ProjectID, brandData.NodeUUID, "NodeUUID бренда в БД совпадает с NodeUUID в запросе")
			sCtx.Assert().Equal(models.StatusDisabled, brandData.Status, "Status бренда в БД совпадает с Status в запросе")
			sCtx.Assert().NotZero(brandData.CreatedAt, "Время создания бренда в БД не равно нулю")
			sCtx.Assert().Zero(brandData.UpdatedAt, "Время обновления бренда в БД равно нулю")

			return nil
		})
		sCtx.Assert().NoError(err, "Ошибка при проверке бренда в БД")

		s.cleanupBrand(t, testData.createCapBrandResponse.ID)
	})
}

func (s *CreateBrandPositiveSuite) createBrandRequest(sCtx provider.StepCtx, name, alias string, names map[string]string) *clientTypes.Request[models.CreateCapBrandRequestBody] {
	return &clientTypes.Request[models.CreateCapBrandRequestBody]{
		Headers: map[string]string{
			"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
			"Platform-Nodeid": s.config.Node.ProjectID,
		},
		Body: &models.CreateCapBrandRequestBody{
			Sort:        3,
			Alias:       alias,
			Names:       names,
			Description: fmt.Sprintf("Description for %s", name),
		},
	}
}

func (s *CreateBrandPositiveSuite) attachRequestResponse(t provider.StepCtx, req *clientTypes.Request[models.CreateCapBrandRequestBody], resp *types.Response[models.CreateCapBrandResponseBody]) {

}

func (s *CreateBrandPositiveSuite) cleanupBrand(t provider.T, brandID string) {
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

func (s *CreateBrandPositiveSuite) AfterAll(t provider.T) {
	t.WithNewStep("Закрытие соединения с базой данных.", func(sCtx provider.StepCtx) {
		if s.database != nil {
			if err := s.database.Close(); err != nil {
				t.Fatalf("Ошибка при закрытии соединения с базой данных: %v", err)
			}
		}
	})
}

func TestCreateBrandPositiveSuite(t *testing.T) {
	suite.RunSuite(t, new(CreateBrandPositiveSuite))
}
