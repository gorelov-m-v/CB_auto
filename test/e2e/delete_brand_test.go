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
	"CB_auto/internal/repository/brand"
	"CB_auto/internal/transport/kafka"
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type DeleteBrandSuite struct {
	suite.Suite
	config     *config.Config
	database   *repository.Connector
	capService capAPI.CapAPI
	kafka      *kafka.Kafka
	brandRepo  *brand.Repository
}

const (
	BrandStatusDeleted = 2
)

func (s *DeleteBrandSuite) BeforeAll(t provider.T) {
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

	t.WithNewStep("Инициализация Kafka consumer.", func(sCtx provider.StepCtx) {
		s.kafka = kafka.GetInstance(t, s.config)
	})
}

func (s *DeleteBrandSuite) TestDeleteBrand(t provider.T) {
	t.Epic("Brands.")
	t.Feature("Удаление бренда.")
	t.Tags("CAP", "Brands", "Platform")
	t.Title("Проверка удаления бренда.")

	var testData struct {
		createCapBrandRequest *clientTypes.Request[models.CreateCapBrandRequestBody]
		createBrandResponse   *clientTypes.Response[models.CreateCapBrandResponseBody]
	}

	t.WithNewStep("Создание бренда.", func(sCtx provider.StepCtx) {
		testData.createCapBrandRequest = &clientTypes.Request[models.CreateCapBrandRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-NodeID": s.config.Node.ProjectID,
			},
			Body: &models.CreateCapBrandRequestBody{
				Sort:  1,
				Alias: utils.Get(utils.ALIAS, 10),
				Names: map[string]string{
					"en": utils.Get(utils.BRAND_TITLE, 20),
					"ru": utils.Get(utils.BRAND_TITLE, 20),
				},
				Description: utils.Get(utils.BRAND_TITLE, 50),
			},
		}
		testData.createBrandResponse = s.capService.CreateCapBrand(sCtx, testData.createCapBrandRequest)

		sCtx.Assert().NotEmpty(testData.createBrandResponse.Body.ID, "ID созданного бренда не пустой")
	})

	t.WithNewAsyncStep("Проверка наличия бренда в БД.", func(sCtx provider.StepCtx) {
		brandFromDB := s.brandRepo.GetBrandWithRetry(sCtx, map[string]interface{}{
			"uuid": testData.createBrandResponse.Body.ID,
		})

		sCtx.Assert().NotNil(brandFromDB, "Бренд найден в БД")
		sCtx.Assert().Equal(testData.createCapBrandRequest.Body.Alias, brandFromDB.Alias, "Алиас бренда совпадает")
	})

	t.WithNewStep("Удаление бренда.", func(sCtx provider.StepCtx) {
		deleteReq := &clientTypes.Request[struct{}]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
			},
			PathParams: map[string]string{
				"id": testData.createBrandResponse.Body.ID,
			},
		}
		deleteResp := s.capService.DeleteCapBrand(sCtx, deleteReq)

		sCtx.Assert().Equal(http.StatusNoContent, deleteResp.StatusCode, "Статус код ответа равен 204")
	})

	t.WithNewAsyncStep("Проверка удаления бренда в БД.", func(sCtx provider.StepCtx) {
		brandFromDB := s.brandRepo.GetBrandWithRetry(sCtx, map[string]interface{}{
			"uuid":   testData.createBrandResponse.Body.ID,
			"status": BrandStatusDeleted,
		})

		sCtx.Assert().NotNil(brandFromDB, "Бренд найден в БД")
		sCtx.Assert().Equal(BrandStatusDeleted, brandFromDB.Status, "Статус бренда - удален")
	})
}

func (s *DeleteBrandSuite) AfterAll(t provider.T) {
	kafka.CloseInstance(t)
	if s.database != nil {
		if err := s.database.Close(); err != nil {
			t.Errorf("Ошибка при закрытии соединения с базой данных: %v", err)
		}
	}
}

func TestDeleteBrandSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(DeleteBrandSuite))
}
