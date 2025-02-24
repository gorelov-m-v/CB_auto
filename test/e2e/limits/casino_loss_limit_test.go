//go:build limit
// +build limit

package test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

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

type CasinoLossLimitSuite struct {
	suite.Suite
	config       *config.Config
	publicClient publicAPI.PublicAPI
	kafka        *kafka.Kafka
	natsClient   *nats.NatsClient
}

func (s *CasinoLossLimitSuite) BeforeAll(t provider.T) {
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

func (s *CasinoLossLimitSuite) TestCasinoLossLimit(t provider.T) {
	t.Feature("casino-loss лимит")
	t.Title("Проверка создания casino-loss лимита в Kafka, NATS, Redis, MySQL, Public API, CAP API")

	var testData struct {
		registrationResponse    *clientTypes.Response[models.FastRegistrationResponseBody]
		authorizationResponse   *clientTypes.Response[models.TokenCheckResponseBody]
		registrationMessage     kafka.PlayerMessage
		limitMessage            kafka.LimitMessage
		casinoLossLimitResponse *clientTypes.Response[models.GetCasinoLossLimitsResponseBody]
		setCasinoLossLimitReq   *clientTypes.Request[models.SetCasinoLossLimitRequestBody]
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

	t.WithNewStep("Проверка сообщения в Kafka о регистрации игрока", func(sCtx provider.StepCtx) {
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

	t.WithNewStep("Установка лимита на проигрыш", func(sCtx provider.StepCtx) {
		testData.setCasinoLossLimitReq = &clientTypes.Request[models.SetCasinoLossLimitRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
			},
			Body: &models.SetCasinoLossLimitRequestBody{
				Amount:    "100",
				Currency:  s.config.Node.DefaultCurrency,
				Type:      "daily",
				StartedAt: time.Now().Unix(),
			},
		}
		resp := s.publicClient.SetCasinoLossLimit(sCtx, testData.setCasinoLossLimitReq)
		sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Лимит на проигрыш установлен")
	})

	t.WithNewStep("Получение сообщения из Kafka о создании лимита на проигрыш", func(sCtx provider.StepCtx) {
		testData.limitMessage = kafka.FindMessageByFilter[kafka.LimitMessage](sCtx, s.kafka, func(msg kafka.LimitMessage) bool {
			return msg.EventType == kafka.LimitEventCreated &&
				msg.LimitType == kafka.LimitTypeCasinoLoss &&
				msg.IntervalType == kafka.IntervalTypeDaily &&
				msg.PlayerID == testData.registrationMessage.Player.ExternalID &&
				msg.Amount == testData.setCasinoLossLimitReq.Body.Amount &&
				msg.CurrencyCode == testData.setCasinoLossLimitReq.Body.Currency
		})
		sCtx.Require().NotEmpty(testData.limitMessage.ID, "ID лимита на проигрыш не пустой")
	})

	t.WithNewAsyncStep("Получение лимита на проигрыш", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
			},
		}
		testData.casinoLossLimitResponse = s.publicClient.GetCasinoLossLimits(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, testData.casinoLossLimitResponse.StatusCode, "Лимиты получены")
		sCtx.Assert().NotEmpty(testData.casinoLossLimitResponse.Body, "Список лимитов не пустой")
		limit := testData.casinoLossLimitResponse.Body[0]
		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ID, "ID лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "Сумма лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.Currency, "Валюта лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.IntervalType, limit.Type, "Тип лимита совпадает")
		sCtx.Assert().True(limit.Status, "Статус лимита активен")
		sCtx.Assert().Empty(limit.UpcomingChanges, "Нет предстоящих изменений")
		sCtx.Assert().Nil(limit.DeactivatedAt, "Лимит не деактивируется")
		sCtx.Assert().Equal(testData.limitMessage.StartedAt, limit.StartedAt, "Время начала установлено")
		sCtx.Assert().Equal(testData.limitMessage.ExpiresAt, limit.ExpiresAt, "Время окончания установлено")
		sCtx.Assert().Equal("0", limit.Spent, "Потраченная сумма равна 0")
		sCtx.Assert().Equal(limit.Amount, limit.Rest, "Остаток равен сумме лимита")
	})

	t.WithNewAsyncStep("Проверка сообщения в NATS о создании лимита на проигрыш", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, testData.registrationMessage.Player.ExternalID)
		casinoLossEvent := nats.FindMessageInStream[nats.LimitChangedV2](sCtx, s.natsClient, subject, func(data nats.LimitChangedV2, msgType string) bool {
			return data.EventType == nats.EventTypeCreated &&
				len(data.Limits) > 0 &&
				data.Limits[0].LimitType == nats.LimitTypeCasinoLoss
		})
		sCtx.Require().NotNil(casinoLossEvent, "Должно получиться получить сообщение из NATS")
		limit := casinoLossEvent.Payload.Limits[0]
		eventType := casinoLossEvent.Payload.EventType
		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ExternalID, "ID лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "Сумма лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.CurrencyCode, "Валюта лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.LimitType, limit.LimitType, "Тип лимита на проигрыш совпадает")
		sCtx.Assert().True(limit.Status, "Статус лимита на проигрыш установлен")
		sCtx.Assert().Equal(testData.limitMessage.StartedAt, limit.StartedAt, "Время начала установлено")
		sCtx.Assert().Equal(testData.limitMessage.ExpiresAt, limit.ExpiresAt, "Время окончания установлено")
		sCtx.Assert().Equal(testData.limitMessage.IntervalType, limit.IntervalType, "Тип интервала совпадает")
		sCtx.Assert().Equal(nats.LimitTypeCasinoLoss, limit.LimitType, "Тип лимита совпадает")
		sCtx.Assert().Equal(eventType, casinoLossEvent.Payload.EventType, "Тип события совпадает")
	})
}

func (s *CasinoLossLimitSuite) AfterAll(t provider.T) {
	kafka.CloseInstance(t)
	if s.natsClient != nil {
		s.natsClient.Close()
	}
}

func TestCasinoLossLimitSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(CasinoLossLimitSuite))
}
