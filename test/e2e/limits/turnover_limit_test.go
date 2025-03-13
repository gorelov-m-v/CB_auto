package test

import (
	"fmt"
	"net/http"
	"time"

	capModels "CB_auto/internal/client/cap/models"
	publicModels "CB_auto/internal/client/public/models"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/repository/wallet"
	"CB_auto/internal/transport/kafka"
	"CB_auto/internal/transport/nats"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type TurnoverLimitSuite struct {
	suite.Suite
	Shared *SharedConnections
}

func (s *TurnoverLimitSuite) SetShared(sc *SharedConnections) {
	s.Shared = sc
}

func (s *TurnoverLimitSuite) TestTurnoverLimit(t provider.T) {
	t.Epic("Лимиты")
	t.Feature("turnover-of-funds лимит")
	t.Title("Проверка создания лимита на оборот средств в Kafka, NATS, Redis, MySQL, Public API")
	t.Tags("wallet", "limits")

	var testData struct {
		registrationResponse  *clientTypes.Response[publicModels.FastRegistrationResponseBody]
		authorizationResponse *clientTypes.Response[publicModels.TokenCheckResponseBody]
		registrationMessage   kafka.PlayerMessage
		turnoverLimitResponse *clientTypes.Response[publicModels.GetTurnoverLimitsResponseBody]
		setTurnoverLimitReq   *clientTypes.Request[publicModels.SetTurnoverLimitRequestBody]
		limitMessage          kafka.LimitMessage
		dbWallet              *wallet.Wallet
	}

	t.WithNewStep("Регистрация нового игрока", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[publicModels.FastRegistrationRequestBody]{
			Body: &publicModels.FastRegistrationRequestBody{
				Country:  s.Shared.Config.Node.DefaultCountry,
				Currency: s.Shared.Config.Node.DefaultCurrency,
			},
		}
		testData.registrationResponse = s.Shared.PublicClient.FastRegistration(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, testData.registrationResponse.StatusCode, "Успешная регистрация")
	})

	t.WithNewStep("Проверка Kafka-сообщения о регистрации игрока", func(sCtx provider.StepCtx) {
		testData.registrationMessage = kafka.FindMessageByFilter(sCtx, s.Shared.Kafka, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == string(kafka.PlayerEventSignUpFast) &&
				msg.Player.AccountID == testData.registrationResponse.Body.Username
		})
		sCtx.Require().NotEmpty(testData.registrationMessage.Player.ID, "ID игрока не пустой")
	})

	t.WithNewStep("Проверка базового кошелька игрока в базе данных", func(sCtx provider.StepCtx) {
		walletFilter := map[string]interface{}{
			"player_uuid": testData.registrationMessage.Player.ExternalID,
			"is_default":  true,
			"is_basic":    true,
			"wallet_type": wallet.WalletTypeReal,
			"currency":    testData.registrationMessage.Player.Currency,
		}
		testData.dbWallet = s.Shared.WalletRepo.GetOneWithRetry(sCtx, walletFilter)
		sCtx.Require().NotNil(testData.dbWallet, "Базовый кошелек найден в базе данных")
	})

	t.WithNewStep("Получение токена авторизации", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[publicModels.TokenCheckRequestBody]{
			Body: &publicModels.TokenCheckRequestBody{
				Username: testData.registrationResponse.Body.Username,
				Password: testData.registrationResponse.Body.Password,
			},
		}
		testData.authorizationResponse = s.Shared.PublicClient.TokenCheck(sCtx, req)
		sCtx.Require().Equal(http.StatusOK, testData.authorizationResponse.StatusCode, "Успешная авторизация")
	})

	t.WithNewStep("Установка лимита на оборот средств", func(sCtx provider.StepCtx) {
		testData.setTurnoverLimitReq = &clientTypes.Request[publicModels.SetTurnoverLimitRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
			},
			Body: &publicModels.SetTurnoverLimitRequestBody{
				Amount:    "100",
				Currency:  s.Shared.Config.Node.DefaultCurrency,
				Type:      publicModels.LimitPeriodDaily,
				StartedAt: time.Now().Unix(),
			},
		}
		resp := s.Shared.PublicClient.SetTurnoverLimit(sCtx, testData.setTurnoverLimitReq)
		sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Лимит на оборот средств установлен")
	})

	t.WithNewStep("Получение сообщения из Kafka о создании лимита на оборот средств", func(sCtx provider.StepCtx) {
		testData.limitMessage = kafka.FindMessageByFilter(sCtx, s.Shared.Kafka, func(msg kafka.LimitMessage) bool {
			return msg.EventType == string(kafka.LimitEventCreated) &&
				msg.LimitType == string(kafka.LimitTypeTurnoverFunds) &&
				msg.PlayerID == testData.registrationMessage.Player.ExternalID &&
				msg.Amount == testData.setTurnoverLimitReq.Body.Amount &&
				msg.CurrencyCode == testData.setTurnoverLimitReq.Body.Currency
		})
		sCtx.Require().NotEmpty(testData.limitMessage.ID, "ID лимита не пустой")
	})

	t.WithNewAsyncStep("Получение лимита через Public API", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
			},
		}
		testData.turnoverLimitResponse = s.Shared.PublicClient.GetTurnoverLimits(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, testData.turnoverLimitResponse.StatusCode, "Лимиты получены")
		sCtx.Assert().NotEmpty(testData.turnoverLimitResponse.Body, "Список лимитов не пустой")
		limit := testData.turnoverLimitResponse.Body[0]
		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ID, "ID лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "Сумма лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.Currency, "Валюта лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.IntervalType, limit.Type, "Тип лимита совпадает")
		sCtx.Assert().True(limit.Status, "Лимит активен")
		sCtx.Assert().Empty(limit.UpcomingChanges, "Нет предстоящих изменений")
		sCtx.Assert().InDelta(testData.limitMessage.StartedAt, limit.StartedAt, 10)
		sCtx.Assert().InDelta(testData.limitMessage.ExpiresAt, limit.ExpiresAt, 10)
		sCtx.Assert().Equal("0", limit.Spent, "Потраченная сумма равна 0")
		sCtx.Assert().Equal(limit.Amount, limit.Rest, "Остаток равен сумме лимита")
	})

	t.WithNewAsyncStep("Проверка NATS-сообщения о создании лимита на оборот средств", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.Shared.Config.Nats.StreamPrefix, testData.registrationMessage.Player.ExternalID)
		turnoverEvent := nats.FindMessageInStream(sCtx, s.Shared.NatsClient, subject, func(msg nats.LimitChangedV2, msgType string) bool {
			return msg.EventType == nats.EventTypeCreated &&
				len(msg.Limits) > 0 &&
				msg.Limits[0].LimitType == nats.LimitTypeTurnoverFunds
		})
		sCtx.Require().NotNil(turnoverEvent, "Получено NATS-сообщение о создании лимита")
		limit := turnoverEvent.Payload.Limits[0]
		eventType := turnoverEvent.Payload.EventType
		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ExternalID, "ID лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "Сумма лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.CurrencyCode, "Валюта лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.LimitType, limit.LimitType, "Тип лимита совпадает")
		sCtx.Assert().True(limit.Status, "Лимит активен")
		sCtx.Assert().InDelta(testData.limitMessage.StartedAt, limit.StartedAt, 10)
		sCtx.Assert().InDelta(testData.limitMessage.ExpiresAt, limit.ExpiresAt, 10)
		sCtx.Assert().Equal(testData.limitMessage.IntervalType, limit.IntervalType, "Тип интервала совпадает")
		sCtx.Assert().Equal(nats.LimitTypeTurnoverFunds, limit.LimitType, "Тип лимита совпадает")
		sCtx.Assert().Equal(eventType, turnoverEvent.Payload.EventType, "Тип события совпадает")
	})

	t.WithNewAsyncStep("Проверка CAP API для лимита на оборот средств", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.Shared.CapClient.GetToken(sCtx)),
				"Platform-NodeId": s.Shared.Config.Node.ProjectID,
				"Platform-Locale": capModels.DefaultLocale,
			},
			PathParams: map[string]string{
				"playerID": testData.registrationMessage.Player.ExternalID,
			},
		}
		resp := s.Shared.CapClient.GetPlayerLimits(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, resp.StatusCode, "Список лимитов получен")
		sCtx.Assert().Equal(1, resp.Body.Total, "Количество лимитов корректно")
		turnoverLimit := resp.Body.Data[0]
		sCtx.Assert().Equal(capModels.LimitTypeTurnover, turnoverLimit.Type)
		sCtx.Assert().True(turnoverLimit.Status)
		sCtx.Assert().Equal(capModels.LimitPeriodDaily, turnoverLimit.Period)
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, turnoverLimit.Currency)
		sCtx.Assert().Equal(testData.limitMessage.Amount, turnoverLimit.Rest)
		sCtx.Assert().Equal(testData.limitMessage.Amount, turnoverLimit.Amount)
		sCtx.Assert().InDelta(testData.limitMessage.StartedAt, turnoverLimit.StartedAt, 10)
		sCtx.Assert().InDelta(testData.limitMessage.ExpiresAt, turnoverLimit.ExpiresAt, 10)
		sCtx.Assert().Zero(turnoverLimit.DeactivatedAt)
	})

	t.WithNewAsyncStep("Проверка Kafka projection event для лимита на оборот средств", func(sCtx provider.StepCtx) {
		projectionMessage := kafka.FindMessageByFilter(sCtx, s.Shared.Kafka, func(msg kafka.ProjectionSourceMessage) bool {
			return msg.Type == string(kafka.ProjectionEventLimitChanged) &&
				msg.PlayerUUID == testData.registrationMessage.Player.ExternalID &&
				msg.WalletUUID == testData.dbWallet.UUID
		})
		sCtx.Require().NotEmpty(projectionMessage.Type, "Сообщение projection event найдено")
		var limitsPayload kafka.ProjectionPayloadLimits
		err := projectionMessage.UnmarshalPayloadTo(&limitsPayload)
		sCtx.Require().NoError(err, "Payload успешно распакован")
		sCtx.Require().GreaterOrEqual(len(limitsPayload.Limits), 1, "В payload есть хотя бы один лимит")
		limit := limitsPayload.Limits[0]
		sCtx.Assert().Equal(string(kafka.LimitEventCreated), limitsPayload.EventType, "Тип события корректен")
		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ExternalID, "ID лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "Сумма лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.CurrencyCode, "Валюта лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.LimitType, limit.LimitType, "Тип лимита совпадает")
		sCtx.Assert().True(limit.Status, "Лимит активен")
	})
}
