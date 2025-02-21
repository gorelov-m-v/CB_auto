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
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type LimitsSuite struct {
	suite.Suite
	config       *config.Config
	publicClient publicAPI.PublicAPI
	kafka        *kafka.Kafka
	natsClient   *nats.NatsClient
}

func (s *LimitsSuite) BeforeAll(t provider.T) {
	t.Epic("Лимиты")

	t.WithNewStep("Чтение конфигурационного файла", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация Public API клиента", func(sCtx provider.StepCtx) {
		s.publicClient = factory.InitClient[publicAPI.PublicAPI](sCtx, s.config, clientTypes.Public)
	})

	t.WithNewStep("Инициализация Kafka клиента", func(sCtx provider.StepCtx) {
		s.kafka = kafka.NewConsumer(t, s.config, kafka.LimitTopic, kafka.PlayerTopic)
		s.kafka.StartReading(t)
	})

	t.WithNewStep("Инициализация NATS клиента", func(sCtx provider.StepCtx) {
		s.natsClient = nats.NewClient(t, &s.config.Nats)
	})
}

func (s *LimitsSuite) AfterAll(t provider.T) {
	if s.kafka != nil {
		s.kafka.Close(t)
	}
	if s.natsClient != nil {
		s.natsClient.Close()
	}
}

func (s *LimitsSuite) TestLimits(t provider.T) {
	t.Feature("Установка и получение лимитов")
	t.Title("Проверка установки и получения всех типов лимитов")

	var testData struct {
		registrationResponse    *clientTypes.Response[models.FastRegistrationResponseBody]
		authorizationResponse   *clientTypes.Response[models.TokenCheckResponseBody]
		registrationMessage     kafka.PlayerMessage
		turnoverLimitMessage    kafka.LimitMessage
		singleBetLimitResponse  *clientTypes.Response[models.GetSingleBetLimitsResponseBody]
		casinoLossLimitResponse *clientTypes.Response[models.GetCasinoLossLimitsResponseBody]
		setSingleBetLimitReq    *clientTypes.Request[models.SetSingleBetLimitRequestBody]
		setCasinoLossLimitReq   *clientTypes.Request[models.SetCasinoLossLimitRequestBody]
		setTurnoverLimitReq     *clientTypes.Request[models.SetTurnoverLimitRequestBody]
	}

	t.WithNewStep("Регистрация нового игрока", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[models.FastRegistrationRequestBody]{
			Body: &models.FastRegistrationRequestBody{
				Country:  s.config.Node.DefaultCountry,
				Currency: s.config.Node.DefaultCurrency,
			},
		}
		testData.registrationResponse = s.publicClient.FastRegistration(sCtx, req)

		sCtx.Assert().Equal(200, testData.registrationResponse.StatusCode, "Успешная регистрация")
	})

	t.WithNewStep("Получение токена авторизации", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[models.TokenCheckRequestBody]{
			Body: &models.TokenCheckRequestBody{
				Username: testData.registrationResponse.Body.Username,
				Password: testData.registrationResponse.Body.Password,
			},
		}
		testData.authorizationResponse = s.publicClient.TokenCheck(sCtx, req)

		sCtx.Assert().Equal(200, testData.authorizationResponse.StatusCode, "Успешная авторизация")
	})

	t.WithNewStep("Проверка сообщения в Kafka о регистрации игрока", func(sCtx provider.StepCtx) {
		message := kafka.FindMessageByFilter(s.kafka, t, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == "player.signUpFast" &&
				msg.Player.AccountID == testData.registrationResponse.Body.Username
		})
		testData.registrationMessage = kafka.ParseMessage[kafka.PlayerMessage](sCtx, message)

		sCtx.Assert().NotEmpty(testData.registrationMessage.Player.ID, "ID игрока не пустой")

		sCtx.WithAttachments(allure.NewAttachment("Player Registration Kafka Message", allure.JSON, utils.CreatePrettyJSON(testData.registrationMessage)))
	})

	t.WithNewStep("Установка лимита на одиночную ставку", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[models.SetSingleBetLimitRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
			},
			Body: &models.SetSingleBetLimitRequestBody{
				Amount:   "100",
				Currency: s.config.Node.DefaultCurrency,
			},
		}
		testData.setSingleBetLimitReq = req

		resp := s.publicClient.SetSingleBetLimit(sCtx, req)

		sCtx.Assert().Equal(http.StatusCreated, resp.StatusCode, "Лимит на ставку установлен")
	})

	t.WithNewStep("Установка лимита на проигрыш", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[models.SetCasinoLossLimitRequestBody]{
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
		testData.setCasinoLossLimitReq = req

		resp := s.publicClient.SetCasinoLossLimit(sCtx, req)

		sCtx.Assert().Equal(201, resp.StatusCode, "Лимит на проигрыш установлен")

		sCtx.WithAttachments(allure.NewAttachment("Set Casino Loss Limit Request", allure.JSON, utils.CreateHttpAttachRequest(req)))
		sCtx.WithAttachments(allure.NewAttachment("Set Casino Loss Limit Response", allure.JSON, utils.CreateHttpAttachResponse(resp)))
	})

	t.WithNewStep("Установка лимита на оборот средств", func(sCtx provider.StepCtx) {
		testData.setTurnoverLimitReq = &clientTypes.Request[models.SetTurnoverLimitRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
			},
			Body: &models.SetTurnoverLimitRequestBody{
				Amount:    "100",
				Currency:  s.config.Node.DefaultCurrency,
				Type:      "daily",
				StartedAt: time.Now().Unix(),
			},
		}

		resp := s.publicClient.SetTurnoverLimit(sCtx, testData.setTurnoverLimitReq)

		sCtx.Assert().Equal(http.StatusCreated, resp.StatusCode, "Лимит на оборот средств установлен")
	})

	t.WithNewStep("Проверка сообщения в Kafka о создании лимита на одиночную ставку", func(sCtx provider.StepCtx) {
		message := kafka.FindMessageByFilter(s.kafka, t, func(msg kafka.LimitMessage) bool {
			match := msg.EventType == kafka.LimitEventCreated &&
				msg.LimitType == kafka.LimitTypeSingleBet &&
				msg.PlayerID == testData.registrationMessage.Player.ExternalID &&
				msg.Amount == testData.setTurnoverLimitReq.Body.Amount &&
				msg.CurrencyCode == s.config.Node.DefaultCurrency
			return match
		})

		limitMsg := kafka.ParseMessage[kafka.LimitMessage](sCtx, message)

		sCtx.Assert().NotEmpty(limitMsg.ID, "ID лимита на ставку не пустой")
		sCtx.Assert().Equal(testData.registrationMessage.Player.ExternalID, limitMsg.PlayerID, "ID игрока в лимите на ставку совпадает с ID игрока в регистрационном сообщении")
		sCtx.Assert().Equal(testData.setSingleBetLimitReq.Body.Amount, limitMsg.Amount, "Тип интервала лимита на ставку совпадает")
		sCtx.Assert().Equal(testData.setSingleBetLimitReq.Body.Amount, limitMsg.Amount, "Сумма лимита на ставку совпадает")
		sCtx.Assert().Equal(testData.setSingleBetLimitReq.Body.Currency, limitMsg.CurrencyCode, "Валюта лимита на ставку совпадает")
		sCtx.Assert().Equal(limitMsg.LimitType, kafka.LimitTypeSingleBet, "Тип лимита на ставку совпадает")
		sCtx.Assert().True(limitMsg.Status, "Статус лимита на ставку установлен")
		sCtx.Assert().Nil(limitMsg.ExpiresAt, "Время начала лимита на ставку установлено")
		sCtx.Assert().True(utils.IsTimeInRange(limitMsg.StartedAt, 10), "Время начала лимита на ставку входит в интервал времени")

		sCtx.WithAttachments(allure.NewAttachment("Single Bet Limit Kafka Message", allure.JSON, utils.CreatePrettyJSON(limitMsg)))
	})

	t.WithNewStep("Проверка сообщения в Kafka о создании лимита на проигрыш", func(sCtx provider.StepCtx) {
		message := kafka.FindMessageByFilter(s.kafka, t, func(msg kafka.LimitMessage) bool {
			match := msg.EventType == kafka.LimitEventCreated &&
				msg.LimitType == kafka.LimitTypeCasinoLoss &&
				msg.IntervalType == kafka.IntervalTypeDaily &&
				msg.Amount == "100" &&
				msg.PlayerID == testData.registrationMessage.Player.ExternalID &&
				msg.Amount == testData.setTurnoverLimitReq.Body.Amount &&
				msg.CurrencyCode == s.config.Node.DefaultCurrency
			return match
		})
		casinoLimitMsg := kafka.ParseMessage[kafka.LimitMessage](sCtx, message)

		sCtx.Assert().NotEmpty(casinoLimitMsg.ID, "ID лимита на проигрыш не пустой")
		sCtx.Assert().Equal(testData.registrationMessage.Player.ExternalID, casinoLimitMsg.PlayerID, "ID игрока в лимите на проигрыш совпадает с ID игрока в регистрационном сообщении")
		sCtx.Assert().Equal(testData.setCasinoLossLimitReq.Body.Amount, casinoLimitMsg.Amount, "Сумма лимита на проигрыш совпадает")
		sCtx.Assert().Equal(testData.setCasinoLossLimitReq.Body.Currency, casinoLimitMsg.CurrencyCode, "Валюта лимита на проигрыш совпадает")
		sCtx.Assert().Equal(kafka.LimitTypeCasinoLoss, casinoLimitMsg.LimitType, "Тип лимита на проигрыш совпадает")
		sCtx.Assert().True(casinoLimitMsg.Status, "Статус лимита на проигрыш установлен")
		sCtx.Assert().NotNil(casinoLimitMsg.ExpiresAt, "Время окончания лимита на проигрыш установлено")
		sCtx.Assert().True(utils.IsTimeInRange(casinoLimitMsg.StartedAt, 10), "Время начала лимита на проигрыш входит в интервал времени")

		sCtx.WithAttachments(allure.NewAttachment("Casino Loss Limit Kafka Message", allure.JSON, utils.CreatePrettyJSON(casinoLimitMsg)))
	})

	t.WithNewStep("Проверка сообщения в Kafka о создании лимита на оборот средств", func(sCtx provider.StepCtx) {
		message := kafka.FindMessageByFilter(s.kafka, t, func(msg kafka.LimitMessage) bool {
			match := msg.EventType == kafka.LimitEventCreated &&
				msg.LimitType == kafka.LimitTypeTurnoverFunds &&
				msg.IntervalType == kafka.IntervalTypeDaily &&
				msg.PlayerID == testData.registrationMessage.Player.ExternalID &&
				msg.Amount == testData.setTurnoverLimitReq.Body.Amount &&
				msg.CurrencyCode == s.config.Node.DefaultCurrency
			return match
		})
		testData.turnoverLimitMessage = kafka.ParseMessage[kafka.LimitMessage](sCtx, message)

		sCtx.Assert().NotEmpty(testData.turnoverLimitMessage.ID, "ID лимита на оборот средств не пустой")
		sCtx.Assert().Equal(testData.registrationMessage.Player.ExternalID, testData.turnoverLimitMessage.PlayerID, "ID игрока в лимите на оборот средств совпадает с ID игрока в регистрационном сообщении")
		sCtx.Assert().Equal(testData.setTurnoverLimitReq.Body.Amount, testData.turnoverLimitMessage.Amount, "Сумма лимита на оборот средств совпадает")
		sCtx.Assert().Equal(testData.setTurnoverLimitReq.Body.Currency, testData.turnoverLimitMessage.CurrencyCode, "Валюта лимита на оборот средств совпадает")
		sCtx.Assert().Equal(kafka.LimitTypeTurnoverFunds, testData.turnoverLimitMessage.LimitType, "Тип лимита на оборот средств совпадает")
		sCtx.Assert().True(testData.turnoverLimitMessage.Status, "Статус лимита на оборот средств установлен")
		sCtx.Assert().NotNil(testData.turnoverLimitMessage.ExpiresAt, "Время окончания лимита на оборот средств установлено")
		sCtx.Assert().True(utils.IsTimeInRange(testData.turnoverLimitMessage.StartedAt, 10), "Время начала лимита на оборот средств входит в интервал времени")

		sCtx.WithAttachments(allure.NewAttachment("Turnover Limit Kafka Message", allure.JSON, utils.CreatePrettyJSON(testData.turnoverLimitMessage)))
	})

	t.WithNewStep("Получение лимита на одиночную ставку", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
			},
		}
		testData.singleBetLimitResponse = s.publicClient.GetSingleBetLimits(sCtx, req)

		sCtx.Assert().Equal(http.StatusOK, testData.singleBetLimitResponse.StatusCode, "Лимиты получены")
		sCtx.Assert().NotEmpty(testData.singleBetLimitResponse.Body, "Список лимитов не пустой")

		limit := testData.singleBetLimitResponse.Body[0]
		sCtx.Assert().Equal(testData.setSingleBetLimitReq.Body.Amount, limit.Amount, "Сумма лимита совпадает")
		sCtx.Assert().Equal(testData.setSingleBetLimitReq.Body.Currency, limit.Currency, "Валюта лимита совпадает")
		sCtx.Assert().True(limit.Status, "Статус лимита активен")
		sCtx.Assert().True(limit.Required, "Лимит обязательный")
		sCtx.Assert().Empty(limit.UpcomingChanges, "Нет предстоящих изменений")
		sCtx.Assert().Nil(limit.DeactivatedAt, "Лимит не деактивирован")
	})

	t.WithNewStep("Получение лимита на оборот средств", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
			},
		}
		resp := s.publicClient.GetTurnoverLimits(sCtx, req)

		sCtx.Assert().Equal(http.StatusOK, resp.StatusCode, "Лимиты получены")
		sCtx.Assert().NotEmpty(resp.Body, "Список лимитов не пустой")

		limit := resp.Body[0]
		sCtx.Assert().Equal(testData.setTurnoverLimitReq.Body.Amount, limit.Amount, "Сумма лимита совпадает")
		sCtx.Assert().Equal(testData.setTurnoverLimitReq.Body.Currency, limit.Currency, "Валюта лимита совпадает")
		sCtx.Assert().Equal(testData.setTurnoverLimitReq.Body.Type, limit.Type, "Тип интервала совпадает")
		sCtx.Assert().True(limit.Status, "Статус лимита активен")
		sCtx.Assert().True(limit.Required, "Лимит обязательный")
		sCtx.Assert().Empty(limit.UpcomingChanges, "Нет предстоящих изменений")
		sCtx.Assert().Nil(limit.DeactivatedAt, "Лимит не деактивирован")
		sCtx.Assert().NotNil(limit.StartedAt, "Время начала установлено")
		sCtx.Assert().NotNil(limit.ExpiresAt, "Время окончания установлено")
		sCtx.Assert().Equal("0", limit.Spent, "Потраченная сумма равна 0")
		sCtx.Assert().Equal(limit.Amount, limit.Rest, "Остаток равен сумме лимита")
	})

	t.WithNewStep("Получение лимита на проигрыш", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
			},
		}
		testData.casinoLossLimitResponse = s.publicClient.GetCasinoLossLimits(sCtx, req)

		sCtx.Assert().Equal(http.StatusOK, testData.casinoLossLimitResponse.StatusCode, "Лимиты получены")
		sCtx.Assert().NotEmpty(testData.casinoLossLimitResponse.Body, "Список лимитов не пустой")

		limit := testData.casinoLossLimitResponse.Body[0]
		sCtx.Assert().Equal(testData.setCasinoLossLimitReq.Body.Amount, limit.Amount, "Сумма лимита совпадает")
		sCtx.Assert().Equal(testData.setCasinoLossLimitReq.Body.Currency, limit.Currency, "Валюта лимита совпадает")
		sCtx.Assert().Equal(testData.setCasinoLossLimitReq.Body.Type, limit.Type, "Тип интервала совпадает")
		sCtx.Assert().True(limit.Status, "Статус лимита активен")
		sCtx.Assert().Empty(limit.UpcomingChanges, "Нет предстоящих изменений")
		sCtx.Assert().Nil(limit.DeactivatedAt, "Лимит не деактивирован")
		sCtx.Assert().NotNil(limit.StartedAt, "Время начала установлено")
		sCtx.Assert().NotNil(limit.ExpiresAt, "Время окончания установлено")
		sCtx.Assert().Equal("0", limit.Spent, "Потраченная сумма равна 0")
		sCtx.Assert().Equal(limit.Amount, limit.Rest, "Остаток равен сумме лимита")
	})

	t.WithNewStep("Проверка сообщения в NATS о создании лимита на одиночную ставку", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, testData.registrationMessage.Player.ExternalID)
		s.natsClient.SubscribeWithDeliverAll(t, subject)

		singleBetEvent := nats.FindMessageByFilter(s.natsClient, t, func(msg nats.LimitChangedV2, msgType string) bool {
			return msg.EventType == "created" &&
				len(msg.Limits) > 0 &&
				msg.Limits[0].LimitType == nats.LimitTypeSingleBet
		})

		limit := singleBetEvent.Payload.Limits[0]
		sCtx.Assert().NotEmpty(limit.ExternalID, "ID лимита на ставку не пустой")
		sCtx.Assert().Equal(testData.setSingleBetLimitReq.Body.Amount, limit.Amount, "Сумма лимита на ставку совпадает")
		sCtx.Assert().Equal(testData.setSingleBetLimitReq.Body.Currency, limit.CurrencyCode, "Валюта лимита на ставку совпадает")
		sCtx.Assert().Equal(nats.LimitTypeSingleBet, limit.LimitType, "Тип лимита на ставку совпадает")
		sCtx.Assert().True(limit.Status, "Статус лимита на ставку установлен")
		sCtx.Assert().Zero(limit.ExpiresAt, "Время окончания лимита на ставку не установлено для single-bet")
		sCtx.Assert().True(utils.IsTimeInRange(limit.StartedAt, 10), "Время начала лимита на ставку входит в интервал времени")

		sCtx.WithAttachments(allure.NewAttachment("Single Bet Limit NATS Event", allure.JSON, utils.CreatePrettyJSON(singleBetEvent)))
	})

	t.WithNewStep("Проверка сообщения в NATS о создании лимита на проигрыш", func(sCtx provider.StepCtx) {
		casinoLossEvent := nats.FindMessageByFilter(s.natsClient, t, func(msg nats.LimitChangedV2, msgType string) bool {
			return msg.EventType == "created" &&
				len(msg.Limits) > 0 &&
				msg.Limits[0].LimitType == nats.LimitTypeCasinoLoss
		})

		limit := casinoLossEvent.Payload.Limits[0]
		sCtx.Assert().NotEmpty(limit.ExternalID, "ID лимита на проигрыш не пустой")
		sCtx.Assert().Equal(testData.setCasinoLossLimitReq.Body.Amount, limit.Amount, "Сумма лимита на проигрыш совпадает")
		sCtx.Assert().Equal(testData.setCasinoLossLimitReq.Body.Currency, limit.CurrencyCode, "Валюта лимита на проигрыш совпадает")
		sCtx.Assert().Equal(nats.LimitTypeCasinoLoss, limit.LimitType, "Тип лимита на проигрыш совпадает")
		sCtx.Assert().True(limit.Status, "Статус лимита на проигрыш установлен")
		sCtx.Assert().NotNil(limit.ExpiresAt, "Время окончания лимита на проигрыш установлено")
		sCtx.Assert().True(utils.IsTimeInRange(limit.StartedAt, 10), "Время начала лимита на проигрыш входит в интервал времени")

		sCtx.WithAttachments(allure.NewAttachment("Casino Loss Limit NATS Event", allure.JSON, utils.CreatePrettyJSON(casinoLossEvent)))
	})

	t.WithNewStep("Проверка сообщения в NATS о создании лимита на оборот средств", func(sCtx provider.StepCtx) {
		turnoverEvent := nats.FindMessageByFilter(s.natsClient, t, func(msg nats.LimitChangedV2, msgType string) bool {
			return msg.EventType == "created" &&
				len(msg.Limits) > 0 &&
				msg.Limits[0].LimitType == nats.LimitTypeTurnoverFunds
		})

		limit := turnoverEvent.Payload.Limits[0]
		sCtx.Assert().NotEmpty(limit.ExternalID, "ID лимита на оборот средств не пустой")
		sCtx.Assert().Equal(testData.setTurnoverLimitReq.Body.Amount, limit.Amount, "Сумма лимита на оборот средств совпадает")
		sCtx.Assert().Equal(testData.setTurnoverLimitReq.Body.Currency, limit.CurrencyCode, "Валюта лимита на оборот средств совпадает")
		sCtx.Assert().Equal(nats.LimitTypeTurnoverFunds, limit.LimitType, "Тип лимита на оборот средств совпадает")
		sCtx.Assert().True(limit.Status, "Статус лимита на оборот средств установлен")
		sCtx.Assert().NotZero(limit.ExpiresAt, "Время окончания лимита на оборот средств установлено")
		sCtx.Assert().True(utils.IsTimeInRange(limit.StartedAt, 10), "Время начала лимита на оборот средств входит в интервал времени")

		sCtx.WithAttachments(allure.NewAttachment("Turnover Limit NATS Event", allure.JSON, utils.CreatePrettyJSON(turnoverEvent)))
	})
}

func TestLimitsSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(LimitsSuite))
}
