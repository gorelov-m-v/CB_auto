package test

import (
	"fmt"
	"net/http"
	"testing"

	capAPI "CB_auto/internal/client/cap"
	capModels "CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/factory"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/internal/repository/label"
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type CreateLabelSuite struct {
	suite.Suite
	config    *config.Config
	capClient capAPI.CapAPI
	database  repository.Connector
	labelRepo *label.Repository
}

func (s *CreateLabelSuite) BeforeAll(t provider.T) {
	t.Epic("Лейблы")

	t.WithNewStep("Чтение конфигурационного файла и инициализация CAP API клиента", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
		s.capClient = factory.InitClient[capAPI.CapAPI](sCtx, s.config, clientTypes.Cap)
	})

	t.WithNewStep("Соединение с базой данных", func(sCtx provider.StepCtx) {
		connector := repository.OpenConnector(t, &s.config.MySQL, repository.Bonus)
		s.database = connector
		s.labelRepo = label.NewRepository(s.database.DB(), &s.config.MySQL)
	})
}

func (s *CreateLabelSuite) TestCreateLabel(t provider.T) {
	t.Feature("Создание лейбла")
	t.Title("Проверка создания лейбла через CAP API")

	var testData struct {
		createLabelRequest  *clientTypes.Request[capModels.CreateLabelRequestBody]
		createLabelResponse *clientTypes.Response[capModels.CreateLabelResponseBody]
	}

	t.WithNewStep("Создание лейбла через CAP API", func(sCtx provider.StepCtx) {
		testData.createLabelRequest = &clientTypes.Request[capModels.CreateLabelRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capClient.GetToken(sCtx)),
				"Platform-NodeId": s.config.Node.GroupID,
				"Platform-Locale": "ru",
				"Accept-Language": "ru",
			},
			Body: &capModels.CreateLabelRequestBody{
				Color:       "#CCCCCC",
				Description: utils.Get(utils.BRAND_TITLE, 50),
				Titles: []capModels.LabelTitle{
					{
						Language: "RU",
						Value:    utils.Get(utils.BRAND_TITLE, 20),
					},
					{
						Language: "LV",
						Value:    utils.Get(utils.BRAND_TITLE, 20),
					},
				},
			},
		}

		testData.createLabelResponse = s.capClient.CreateLabel(sCtx, testData.createLabelRequest)

		sCtx.Assert().Equal(http.StatusOK, testData.createLabelResponse.StatusCode, "Статус ответа корректен")
		sCtx.Assert().NotEmpty(testData.createLabelResponse.Body.UUID, "UUID лейбла сгенерирован")
	})

	t.WithNewStep("Получение информации о созданном лейбле через CAP API", func(sCtx provider.StepCtx) {
		getLabelRequest := &clientTypes.Request[struct{}]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capClient.GetToken(sCtx)),
				"Platform-NodeId": s.config.Node.GroupID,
				"Platform-Locale": "ru",
				"Accept-Language": "ru",
			},
			PathParams: map[string]string{
				"labelUUID": testData.createLabelResponse.Body.UUID,
			},
		}

		labelInfoResponse := s.capClient.GetLabel(sCtx, getLabelRequest)

		sCtx.Assert().Equal(http.StatusOK, labelInfoResponse.StatusCode, "Статус ответа корректен")
		sCtx.Assert().Equal(testData.createLabelResponse.Body.UUID, labelInfoResponse.Body.UUID, "UUID лейбла совпадает")
		sCtx.Assert().Equal(testData.createLabelRequest.Body.Color, labelInfoResponse.Body.Color, "Цвет лейбла совпадает")
		sCtx.Assert().Equal(s.config.Node.GroupID, labelInfoResponse.Body.Node, "ID ноды совпадает")
		sCtx.Assert().Equal(testData.createLabelRequest.Body.Description, labelInfoResponse.Body.Description, "Описание лейбла совпадает")
		sCtx.Assert().Len(labelInfoResponse.Body.Titles, 2, "Количество заголовков")
		sCtx.Assert().Equal("RU", labelInfoResponse.Body.Titles[0].Language, "Язык заголовка")
		sCtx.Assert().NotEmpty(labelInfoResponse.Body.Titles[0].Value, "Значение заголовка не пустое")
		sCtx.Assert().Equal("LV", labelInfoResponse.Body.Titles[1].Language, "Язык заголовка")
		sCtx.Assert().NotEmpty(labelInfoResponse.Body.Titles[1].Value, "Значение заголовка не пустое")
	})

	t.WithNewStep("Проверка лейбла в базе данных", func(sCtx provider.StepCtx) {
		dbLabel := s.labelRepo.GetLabelWithRetry(sCtx, map[string]interface{}{"uuid": testData.createLabelResponse.Body.UUID})

		sCtx.Require().NotNil(dbLabel, "Лейбл найден в базе данных")
		sCtx.Assert().Equal(testData.createLabelRequest.Body.Color, dbLabel.Color, "Цвет лейбла совпадает")
		sCtx.Assert().Equal(testData.createLabelRequest.Body.Description, dbLabel.Description, "Описание лейбла совпадает")
		sCtx.Assert().Equal(s.config.Node.GroupID, dbLabel.Node, "ID ноды совпадает")
		sCtx.Assert().NotEmpty(dbLabel.AuthorCreation, "Автор создания указан")
	})
}

func (s *CreateLabelSuite) AfterAll(t provider.T) {
	t.WithNewStep("Закрытие соединения с базой данных.", func(sCtx provider.StepCtx) {
		if err := s.database.Close(); err != nil {
			t.Errorf("Ошибка при закрытии соединения с wallet DB: %v", err)
		} else {
		}
	})
}

func TestCreateLabelSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(CreateLabelSuite))
}
