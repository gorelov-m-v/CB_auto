//go:build limit
// +build limit

package test

import (
	"fmt"
	"net/http"
	"testing"

	"CB_auto/internal/client/factory"
	publicAPI "CB_auto/internal/client/public"
	"CB_auto/internal/client/public/models"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/internal/transport/kafka"
	"CB_auto/internal/transport/nats"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type SingleBetLimitSuite struct {
	suite.Suite
	config       *config.Config
	publicClient publicAPI.PublicAPI
	kafka        *kafka.Kafka
	natsClient   *nats.NatsClient
}

func (s *SingleBetLimitSuite) BeforeAll(t provider.T) {
	t.Epic("Лимиты")

	t.WithNewStep("Чтение конфигурационного файла", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация Public API клиента", func(sCtx provider.StepCtx) {
		s.publicClient = factory.InitClient[publicAPI.PublicAPI](sCtx, s.config, clientTypes.Public)
	})

	t.WithNewStep("Инициализация Kafka клиента", func(sCtx provider.StepCtx) {
		s.kafka = kafka.GetInstance(t, s.config)
	})

	t.WithNewStep("Инициализация NATS клиента", func(sCtx provider.StepCtx) {
		s.natsClient = nats.NewClient(&s.config.Nats)
	})
}

func (s *SingleBetLimitSuite) TestSingleBetLimit(t provider.T) {
	t.Feature("single-bet лимит")
	t.Title("Проверка создания single-bet лимита в Kafka, NATS, Redis, MySQL, Public API, CAP API")

	var testData struct {
		registrationResponse   *clientTypes.Response[models.FastRegistrationResponseBody]
		authorizationResponse  *clientTypes.Response[models.TokenCheckResponseBody]
		registrationMessage    kafka.PlayerMessage
		singleBetLimitResponse *clientTypes.Response[models.GetSingleBetLimitsResponseBody] // изменить тип
		setSingleBetLimitReq   *clientTypes.Request[models.SetSingleBetLimitRequestBody]
		limitMessage           kafka.LimitMessage
	}

	t.WithNewStep("Регистрация нового игрока", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[models.FastRegistrationRequestBody]{
			Body: &models.FastRegistrationRequestBody{
				Country:  s.config.Node.DefaultCountry,
				Currency: s.config.Node.DefaultCurrency,
			},
		}

		testData.registrationResponse = s.publicClient.FastRegistration(sCtx, req)

		sCtx.Assert().Equal(http.StatusOK, testData.registrationResponse.StatusCode, "Успешная регистрация")
	})

	t.WithNewStep("Проверка Kafka-сообщения о регистрации игрока", func(sCtx provider.StepCtx) {
		testData.registrationMessage = kafka.FindMessageByFilter[kafka.PlayerMessage](sCtx, s.kafka, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == "player.signUpFast" &&
				msg.Player.AccountID == testData.registrationResponse.Body.Username
		})

		sCtx.Require().NotEmpty(testData.registrationMessage.Player.ID, "ID игрока не пустой")
	})

	t.WithNewStep("Получение токена авторизации", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[models.TokenCheckRequestBody]{
			Body: &models.TokenCheckRequestBody{
				Username: testData.registrationResponse.Body.Username,
				Password: testData.registrationResponse.Body.Password,
			},
		}

		testData.authorizationResponse = s.publicClient.TokenCheck(sCtx, req)

		sCtx.Require().Equal(http.StatusOK, testData.authorizationResponse.StatusCode, "Успешная авторизация")
	})

	t.WithNewStep("Установка лимита на одиночную ставку", func(sCtx provider.StepCtx) {
		testData.setSingleBetLimitReq = &clientTypes.Request[models.SetSingleBetLimitRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
			},
			Body: &models.SetSingleBetLimitRequestBody{
				Amount:   "100",
				Currency: s.config.Node.DefaultCurrency,
			},
		}

		resp := s.publicClient.SetSingleBetLimit(sCtx, testData.setSingleBetLimitReq)

		sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Лимит на ставку установлен")
	})

	t.WithNewStep("Проверка Kafka-сообщения о создании лимита", func(sCtx provider.StepCtx) {
		testData.limitMessage = kafka.FindMessageByFilter[kafka.LimitMessage](sCtx, s.kafka, func(msg kafka.LimitMessage) bool {
			return msg.EventType == kafka.LimitEventCreated &&
				msg.LimitType == kafka.LimitTypeSingleBet &&
				msg.PlayerID == testData.registrationMessage.Player.ExternalID &&
				msg.Amount == testData.setSingleBetLimitReq.Body.Amount &&
				msg.CurrencyCode == s.config.Node.DefaultCurrency
		})

		sCtx.Require().NotEmpty(testData.limitMessage.ID, "ID лимита не пустой")
	})

	t.WithNewAsyncStep("Получение лимита через Public API", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
			},
		}

		testData.singleBetLimitResponse = s.publicClient.GetSingleBetLimits(sCtx, req)

		sCtx.Assert().Equal(http.StatusOK, testData.singleBetLimitResponse.StatusCode, "Лимиты получены")
		sCtx.Assert().NotEmpty(testData.singleBetLimitResponse.Body, "Список лимитов не пустой")

		limit := testData.singleBetLimitResponse.Body[0]
		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ID, "ID лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "Сумма лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.Currency, "Валюта лимита совпадает")
		sCtx.Assert().True(limit.Status, "Лимит активен")
		sCtx.Assert().True(limit.Required, "Лимит обязательный")
		sCtx.Assert().Empty(limit.UpcomingChanges, "Нет предстоящих изменений")
		sCtx.Assert().Nil(limit.DeactivatedAt, "Лимит не деактивирован")
	})

	t.WithNewAsyncStep("Проверка NATS-сообщения о создании лимита", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, testData.registrationMessage.Player.ExternalID)

		singleBetEvent := nats.FindMessageInStream(sCtx, s.natsClient, subject, func(msg nats.LimitChangedV2, msgType string) bool {
			return msg.EventType == nats.EventTypeCreated &&
				len(msg.Limits) > 0 &&
				msg.Limits[0].LimitType == nats.LimitTypeSingleBet
		})

		sCtx.Require().NotNil(singleBetEvent, "Получено NATS-сообщение о создании лимита")

		limit := singleBetEvent.Payload.Limits[0]
		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ExternalID, "ID лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "Сумма лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.CurrencyCode, "Валюта лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.LimitType, limit.LimitType, "Тип лимита совпадает")
		sCtx.Assert().True(limit.Status, "Лимит активен")
		sCtx.Assert().Equal(testData.limitMessage.StartedAt, limit.StartedAt, "Время начала установлено")
		sCtx.Assert().Zero(limit.ExpiresAt, "Время окончания не установлено")
		sCtx.Assert().Equal(nats.LimitTypeSingleBet, limit.LimitType, "Тип лимита совпадает")
		sCtx.Assert().Equal(singleBetEvent.Payload.EventType, testData.limitMessage.EventType, "Тип события совпадает")
	})
}

func (s *SingleBetLimitSuite) AfterAll(t provider.T) {
	kafka.CloseInstance(t)
	if s.natsClient != nil {
		s.natsClient.Close()
	}
}

func TestSingleBetLimitSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(SingleBetLimitSuite))
}
