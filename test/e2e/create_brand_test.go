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

type CreateBrandSuite struct {
	suite.Suite
	config     *config.Config
	database   *repository.Connector
	capService capAPI.CapAPI
	kafka      *kafka.Kafka
	brandRepo  *brand.Repository
}

func (s *CreateBrandSuite) BeforeAll(t provider.T) {
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

func (s *CreateBrandSuite) TestGetBrandByFilters(t provider.T) {
	t.Epic("Brands.")
	t.Feature("Получение бренда по фильтрам.")
	t.Tags("CAP", "Brands", "Platform")
	t.Title("Проверка получения бренда из БД по универсальным фильтрам.")

	var testData struct {
		createCapBrandRequest  *clientTypes.Request[models.CreateCapBrandRequestBody]
		createCapBrandResponse *clientTypes.Response[models.CreateCapBrandResponseBody]
	}

	t.WithNewStep("Создание бренда в CAP.", func(sCtx provider.StepCtx) {
		names := map[string]string{
			"en": utils.Get(utils.BRAND_TITLE, 20),
		}

		testData.createCapBrandRequest = &clientTypes.Request[models.CreateCapBrandRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-Locale": "en",
				"Platform-NodeId": s.config.Node.ProjectID,
			},
			Body: &models.CreateCapBrandRequestBody{
				Sort:        1,
				Alias:       utils.Get(utils.ALIAS, 10),
				Names:       names,
				Description: utils.Get(utils.BRAND_TITLE, 50),
			},
		}

		testData.createCapBrandResponse = s.capService.CreateCapBrand(sCtx, testData.createCapBrandRequest)
	})

	t.WithNewStep("Получение бренда из БД по универсальным фильтрам.", func(sCtx provider.StepCtx) {
		filters := map[string]interface{}{
			"uuid": testData.createCapBrandResponse.Body.ID,
		}
		brandData := s.brandRepo.GetBrand(sCtx, filters)

		sCtx.Assert().Equal(testData.createCapBrandRequest.Body.Names, brandData.LocalizedNames, "Names бренда в БД совпадают с Names в запросе")
		sCtx.Assert().Equal(testData.createCapBrandRequest.Body.Description, brandData.Description, "Description бренда в БД совпадает с Description в запросе")
		sCtx.Assert().Equal(testData.createCapBrandRequest.Body.Alias, brandData.Alias, "Alias бренда в БД совпадает с Alias в запросе")
		sCtx.Assert().Equal(testData.createCapBrandRequest.Body.Sort, brandData.Sort, "Sort бренда в БД совпадает с Sort в запросе")
		sCtx.Assert().Equal(s.config.Node.ProjectID, brandData.NodeUUID, "NodeUUID бренда в БД совпадает с NodeUUID в запросе")
		sCtx.Assert().Equal(models.StatusDisabled, brandData.Status, "Status бренда в БД равен StatusDisabled")
		sCtx.Assert().NotZero(brandData.CreatedAt, "CreatedAt бренда в БД не равен нулю")
		sCtx.Assert().Zero(brandData.UpdatedAt, "UpdatedAt бренда в БД равен нулю для нового бренда")
		sCtx.Assert().Equal(testData.createCapBrandResponse.Body.ID, brandData.UUID, "UUID бренда в БД совпадает с UUID в ответе")
	})

	t.WithNewStep("Удаление бренда.", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-Nodeid": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": testData.createCapBrandResponse.Body.ID,
			},
		}

		resp := s.capService.DeleteCapBrand(sCtx, req)

		sCtx.Assert().Equal(http.StatusNoContent, resp.StatusCode, "Статус ответа при удалении бренда в CAP соответствует ожидаемому")
	})
}

func (s *CreateBrandSuite) AfterAll(t provider.T) {
	t.WithNewStep("Закрытие соединения с Kafka.", func(sCtx provider.StepCtx) {
		kafka.CloseInstance(t)
	})

	t.WithNewStep("Закрытие соединения с базой данных.", func(sCtx provider.StepCtx) {
		if s.database != nil {
			if err := s.database.Close(); err != nil {
				t.Fatalf("Ошибка при закрытии соединения с базой данных: %v", err)
			}
		}
	})
}

func TestCreateBrandSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(CreateBrandSuite))
}
