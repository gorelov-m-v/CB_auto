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
	"CB_auto/internal/client/types"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/internal/transport/kafka"
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/allure"
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
		s.capService = factory.InitClient[capAPI.CapAPI](t, s.config, clientTypes.Cap)
	})

	t.WithNewStep("Соединение с базой данных.", func(sCtx provider.StepCtx) {
		connector := repository.OpenConnector(t, &s.config.MySQL, repository.Core)
		s.database = &connector
	})

	t.WithNewStep("Инициализация Kafka consumer.", func(sCtx provider.StepCtx) {
		s.kafka = kafka.NewConsumer(t, s.config, kafka.BrandTopic)
		s.kafka.StartReading(t)
	})
}

func (s *UpdateBrandPositiveSuite) AfterAll(t provider.T) {
	t.WithNewStep("Закрытие соединения с Kafka.", func(sCtx provider.StepCtx) {
		if s.kafka != nil {
			s.kafka.Close(t)
		}
	})

	t.WithNewStep("Закрытие соединения с базой данных.", func(sCtx provider.StepCtx) {
		if s.database != nil {
			if err := s.database.Close(); err != nil {
				sCtx.Assert().NoError(err, "Ошибка закрытия соединения с базой данных")
			}
		}
	})
}

func (s *UpdateBrandPositiveSuite) TestUpdateBrandWithRussianName(t provider.T) {
	var testData struct {
		createRequest          *clientTypes.Request[models.CreateCapBrandRequestBody]
		updateRequest          *clientTypes.Request[models.UpdateCapBrandRequestBody]
		createCapBrandResponse *models.CreateCapBrandResponseBody
		updateCapBrandResponse *models.UpdateCapBrandResponseBody
	}

	// Создаем бренд для тестирования
	t.WithNewStep("Создание тестового бренда", func(sCtx provider.StepCtx) {
		brandName := fmt.Sprintf("Test Brand %s", utils.GenerateAlias())
		alias := fmt.Sprintf("test-brand-%s", utils.GenerateAlias())
		testData.createRequest = s.createBrandRequest(brandName, alias, map[string]string{
			"en": brandName,
		})

		createResp := s.capService.CreateCapBrand(testData.createRequest)
		sCtx.Assert().Equal(http.StatusOK, createResp.StatusCode)
		testData.createCapBrandResponse = &createResp.Body

		s.attachCreateRequestResponse(sCtx, testData.createRequest, createResp)
	})

	time.Sleep(1 * time.Second)

	// Обновляем бренд
	t.WithNewStep("Обновление бренда с русским названием", func(sCtx provider.StepCtx) {
		brandName := fmt.Sprintf("Тестовый бренд %s", utils.GenerateAlias())
		alias := fmt.Sprintf("test-brand-ru-%s", utils.GenerateAlias())
		testData.updateRequest = s.updateBrandRequest(
			testData.createCapBrandResponse.ID,
			brandName,
			alias,
			map[string]string{"ru": brandName},
		)

		updateResp := s.capService.UpdateCapBrand(testData.updateRequest)
		sCtx.Assert().Equal(http.StatusOK, updateResp.StatusCode)
		testData.updateCapBrandResponse = &updateResp.Body

		s.attachRequestResponse(sCtx, testData.updateRequest, updateResp)
	})

	s.cleanupBrand(t, testData.createCapBrandResponse.ID)
}

func (s *UpdateBrandPositiveSuite) TestUpdateBrandWithEnglishName(t provider.T) {
	var testData struct {
		createRequest          *clientTypes.Request[models.CreateCapBrandRequestBody]
		updateRequest          *clientTypes.Request[models.UpdateCapBrandRequestBody]
		createCapBrandResponse *models.CreateCapBrandResponseBody
		updateCapBrandResponse *models.UpdateCapBrandResponseBody
	}

	// Создаем бренд для тестирования
	t.WithNewStep("Создание тестового бренда", func(sCtx provider.StepCtx) {
		brandName := fmt.Sprintf("Test Brand %s", utils.GenerateAlias())
		alias := fmt.Sprintf("test-brand-%s", utils.GenerateAlias())
		testData.createRequest = s.createBrandRequest(brandName, alias, map[string]string{
			"en": brandName,
		})

		createResp := s.capService.CreateCapBrand(testData.createRequest)
		sCtx.Assert().Equal(http.StatusOK, createResp.StatusCode)
		testData.createCapBrandResponse = &createResp.Body

		s.attachCreateRequestResponse(sCtx, testData.createRequest, createResp)
	})

	time.Sleep(1 * time.Second)

	// Обновляем бренд
	t.WithNewStep("Обновление бренда с английским названием", func(sCtx provider.StepCtx) {
		brandName := fmt.Sprintf("Updated Test Brand %s", utils.GenerateAlias())
		alias := fmt.Sprintf("test-brand-en-%s", utils.GenerateAlias())
		testData.updateRequest = s.updateBrandRequest(
			testData.createCapBrandResponse.ID,
			brandName,
			alias,
			map[string]string{"en": brandName},
		)

		updateResp := s.capService.UpdateCapBrand(testData.updateRequest)
		sCtx.Assert().Equal(http.StatusOK, updateResp.StatusCode)
		testData.updateCapBrandResponse = &updateResp.Body

		s.attachRequestResponse(sCtx, testData.updateRequest, updateResp)
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

	// Создаем бренд для тестирования
	t.WithNewStep("Создание тестового бренда", func(sCtx provider.StepCtx) {
		brandName := fmt.Sprintf("Test Brand %s", utils.GenerateAlias())
		alias := fmt.Sprintf("test-brand-%s", utils.GenerateAlias())
		testData.createRequest = s.createBrandRequest(brandName, alias, map[string]string{
			"en": brandName,
		})

		createResp := s.capService.CreateCapBrand(testData.createRequest)
		sCtx.Assert().Equal(http.StatusOK, createResp.StatusCode)
		testData.createCapBrandResponse = &createResp.Body

		s.attachCreateRequestResponse(sCtx, testData.createRequest, createResp)
	})

	time.Sleep(1 * time.Second)

	// Обновляем до минимального значения
	t.WithNewStep("Обновление бренда с минимальной длиной имени (2 символа)", func(sCtx provider.StepCtx) {
		// Генерируем уникальное двухбуквенное имя с помощью timestamp
		timestamp := time.Now().UnixNano()
		suffix := fmt.Sprintf("%x", timestamp)[:2] // Берем первые 2 символа hex представления timestamp
		minName := fmt.Sprintf("A%s", suffix)      // Префикс + 2 уникальных символа
		minAlias := fmt.Sprintf("min-%s", utils.GenerateAlias())

		testData.updateRequest = s.updateBrandRequest(
			testData.createCapBrandResponse.ID,
			minName,
			minAlias,
			map[string]string{"en": minName},
		)

		updateResp := s.capService.UpdateCapBrand(testData.updateRequest)
		sCtx.Assert().Equal(http.StatusOK, updateResp.StatusCode)

		s.attachRequestResponse(sCtx, testData.updateRequest, updateResp)
	})

	time.Sleep(1 * time.Second)

	// Обновляем до максимального значения
	t.WithNewStep("Обновление бренда с максимальной длиной имени (100 символов)", func(sCtx provider.StepCtx) {
		timestamp := time.Now().UnixNano()
		// Гарантируем длину 100 символов
		maxName := fmt.Sprintf("%s%d", strings.Repeat("B", 80), timestamp) // timestamp примерно 19 символов
		if len(maxName) > 100 {
			maxName = maxName[:100]
		}
		maxAlias := fmt.Sprintf("max-%s", utils.GenerateAlias())

		testData.updateRequest = s.updateBrandRequest(
			testData.createCapBrandResponse.ID,
			maxName,
			maxAlias,
			map[string]string{"en": maxName},
		)

		updateResp := s.capService.UpdateCapBrand(testData.updateRequest)
		sCtx.Assert().Equal(http.StatusOK, updateResp.StatusCode)

		s.attachRequestResponse(sCtx, testData.updateRequest, updateResp)
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

	// Создаем бренд для тестирования
	t.WithNewStep("Создание тестового бренда", func(sCtx provider.StepCtx) {
		brandName := fmt.Sprintf("Test Brand %s", utils.GenerateAlias())
		alias := fmt.Sprintf("test-brand-%s", utils.GenerateAlias())
		testData.createRequest = s.createBrandRequest(brandName, alias, map[string]string{
			"en": brandName,
		})

		createResp := s.capService.CreateCapBrand(testData.createRequest)
		sCtx.Assert().Equal(http.StatusOK, createResp.StatusCode)
		testData.createCapBrandResponse = &createResp.Body
	})

	time.Sleep(1 * time.Second)

	t.WithNewStep("Обновление мультиязычного бренда", func(sCtx provider.StepCtx) {
		suffix := utils.GenerateAlias()
		alias := fmt.Sprintf("multi-lang-%s", suffix)
		testData.updateRequest = s.updateBrandRequest(testData.createCapBrandResponse.ID, "Multilingual Brand", alias, map[string]string{
			"en": fmt.Sprintf("Updated Test Brand %s", suffix),
			"ru": fmt.Sprintf("Обновленный тестовый бренд %s", suffix),
			"es": fmt.Sprintf("Marca de prueba actualizada %s", suffix),
		})

		updateResp := s.capService.UpdateCapBrand(testData.updateRequest)
		sCtx.Assert().Equal(http.StatusOK, updateResp.StatusCode)
		testData.updateCapBrandResponse = &updateResp.Body

		s.attachRequestResponse(sCtx, testData.updateRequest, updateResp)
	})

	s.cleanupBrand(t, testData.createCapBrandResponse.ID)
}

// Вспомогательные методы
func (s *UpdateBrandPositiveSuite) createBrandRequest(name, alias string, names map[string]string) *clientTypes.Request[models.CreateCapBrandRequestBody] {
	// Проверяем минимальную длину alias
	if len(alias) < 2 {
		alias = fmt.Sprintf("%s-%s", alias, utils.GenerateAlias())
	}

	// Проверяем минимальную длину названий
	for lang, localName := range names {
		if len(localName) < 2 {
			names[lang] = fmt.Sprintf("%s-%s", localName, utils.GenerateAlias())
		}
	}

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

func (s *UpdateBrandPositiveSuite) updateBrandRequest(id, name, alias string, names map[string]string) *clientTypes.Request[models.UpdateCapBrandRequestBody] {
	// Проверяем минимальную длину alias
	if len(alias) < 2 {
		alias = fmt.Sprintf("%s-%s", alias, utils.GenerateAlias())
	}

	// Проверяем минимальную длину названий
	for lang, localName := range names {
		if len(localName) < 2 {
			names[lang] = fmt.Sprintf("%s-%s", localName, utils.GenerateAlias())
		}
	}

	return &clientTypes.Request[models.UpdateCapBrandRequestBody]{
		Headers: map[string]string{
			"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken()),
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

func (s *UpdateBrandPositiveSuite) attachRequestResponse(t provider.StepCtx, req *clientTypes.Request[models.UpdateCapBrandRequestBody], resp *types.Response[models.UpdateCapBrandResponseBody]) {
	t.WithAttachments(allure.NewAttachment("UpdateBrand Request", allure.JSON, utils.CreateHttpAttachRequest(req)))
	t.WithAttachments(allure.NewAttachment("UpdateBrand Response", allure.JSON, utils.CreateHttpAttachResponse(resp)))
}

func (s *UpdateBrandPositiveSuite) attachCreateRequestResponse(t provider.StepCtx, req *clientTypes.Request[models.CreateCapBrandRequestBody], resp *types.Response[models.CreateCapBrandResponseBody]) {
	t.WithAttachments(allure.NewAttachment("CreateBrand Request", allure.JSON, utils.CreateHttpAttachRequest(req)))
	t.WithAttachments(allure.NewAttachment("CreateBrand Response", allure.JSON, utils.CreateHttpAttachResponse(resp)))
}

func (s *UpdateBrandPositiveSuite) cleanupBrand(t provider.T, brandID string) {
	t.WithNewStep("Удаление тестового бренда", func(sCtx provider.StepCtx) {
		deleteReq := &clientTypes.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken()),
				"Platform-Nodeid": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"id": brandID,
			},
		}

		deleteResp := s.capService.DeleteCapBrand(deleteReq)
		sCtx.Assert().Equal(http.StatusNoContent, deleteResp.StatusCode)
	})
}

func TestUpdateBrandPositiveSuite(t *testing.T) {
	suite.RunSuite(t, new(UpdateBrandPositiveSuite))
}
