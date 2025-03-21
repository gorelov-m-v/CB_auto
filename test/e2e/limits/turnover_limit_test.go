package test

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	capModels "CB_auto/internal/client/cap/models"
	publicModels "CB_auto/internal/client/public/models"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/repository/wallet"
	"CB_auto/internal/transport/kafka"
	"CB_auto/internal/transport/nats"
	"CB_auto/internal/transport/redis"
	"CB_auto/pkg/utils"
	defaultSteps "CB_auto/pkg/utils/default_steps"

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

func (s *TurnoverLimitSuite) TestTurnoverLimitCreate(t provider.T) {
	t.Epic("Лимиты")
	t.Feature("turnover-of-funds лимит")
	t.Title("Проверка создания лимита на оборот средств")
	t.Tags("wallet", "limits")

	var testData struct {
		authToken             string
		walletAggregate       redis.WalletFullData
		turnoverLimitRequest  *clientTypes.Request[publicModels.SetTurnoverLimitRequestBody]
		turnoverLimitResponse *clientTypes.Response[publicModels.GetTurnoverLimitsResponseBody]
		limitMessage          kafka.LimitMessage
		turnoverEvent         *nats.NatsMessage[nats.LimitChangedV2]
	}

	t.WithNewStep("Создание игрока через полную регистрацию", func(sCtx provider.StepCtx) {
		playerData := defaultSteps.CreatePlayerWithFullRegistration(
			sCtx,
			s.Shared.PublicClient,
			s.Shared.Kafka,
			s.Shared.Config,
			s.Shared.PlayerRedisClient,
			s.Shared.WalletRedisClient,
		)

		sCtx.Require().NotEmpty(playerData.Auth.Body.Token, "Токен авторизации получен")
		sCtx.Require().NotEmpty(playerData.WalletData.WalletUUID, "UUID кошелька получен")
		sCtx.Require().NotEmpty(playerData.WalletData.PlayerUUID, "UUID игрока получен")

		testData.authToken = playerData.Auth.Body.Token
		testData.walletAggregate = playerData.WalletData
	})

	t.WithNewStep("Установка лимита на оборот средств", func(sCtx provider.StepCtx) {
		testData.turnoverLimitRequest = &clientTypes.Request[publicModels.SetTurnoverLimitRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authToken),
			},
			Body: &publicModels.SetTurnoverLimitRequestBody{
				Amount:    "100",
				Currency:  s.Shared.Config.Node.DefaultCurrency,
				Type:      publicModels.LimitPeriodDaily,
				StartedAt: int(time.Now().Unix()),
			},
		}

		resp := s.Shared.PublicClient.SetTurnoverLimit(sCtx, testData.turnoverLimitRequest)
		sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Лимит на оборот средств установлен")
	})

	t.WithNewStep("Получение сообщения о создании лимита на оборот средств из топика limits.v2", func(sCtx provider.StepCtx) {
		testData.limitMessage = kafka.FindMessageByFilter(sCtx, s.Shared.Kafka, func(msg kafka.LimitMessage) bool {
			return msg.EventType == kafka.LimitEventCreated &&
				msg.LimitType == kafka.LimitTypeTurnoverFunds &&
				msg.PlayerID == testData.walletAggregate.PlayerUUID &&
				msg.Amount == testData.turnoverLimitRequest.Body.Amount &&
				msg.CurrencyCode == testData.turnoverLimitRequest.Body.Currency
		})
		sCtx.Require().NotEmpty(testData.limitMessage.ID, "Сообщение о создании лимита на проигрыш из топика limits.v2 получено")
	})

	t.WithNewStep("Проверка ивента NATS о создании лимита на оборот средств", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.Shared.Config.Nats.StreamPrefix, testData.walletAggregate.PlayerUUID)
		testData.turnoverEvent = nats.FindMessageInStream(sCtx, s.Shared.NatsClient, subject, func(msg nats.LimitChangedV2, msgType string) bool {
			return msg.EventType == nats.EventTypeCreated &&
				len(msg.Limits) > 0 &&
				msg.Limits[0].LimitType == nats.LimitTypeTurnoverFunds
		})
		sCtx.Require().NotNil(testData.turnoverEvent, "Получено NATS-сообщение о создании лимита")

		limit := testData.turnoverEvent.Payload.Limits[0]
		eventType := testData.turnoverEvent.Payload.EventType
		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ExternalID, "NATS: Проверка параметра external_id")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "NATS: Проверка параметра amount")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.CurrencyCode, "NATS: Проверка параметра currency_code")
		sCtx.Assert().Equal(string(testData.limitMessage.LimitType), string(limit.LimitType), "NATS: Проверка параметра limit_type")
		sCtx.Assert().True(limit.Status, "NATS: Проверка параметра status")
		sCtx.Assert().InDelta(testData.limitMessage.StartedAt, limit.StartedAt, 10, "NATS: Проверка параметра started_at")
		sCtx.Assert().InDelta(testData.limitMessage.ExpiresAt, limit.ExpiresAt, 10, "NATS: Проверка параметра expires_at")
		sCtx.Assert().Equal(string(testData.limitMessage.IntervalType), limit.IntervalType, "NATS: Проверка параметра interval_type")
		sCtx.Assert().Equal(nats.LimitTypeTurnoverFunds, limit.LimitType, "NATS: Проверка параметра limit_type")
		sCtx.Assert().Equal(eventType, testData.turnoverEvent.Payload.EventType, "NATS: Проверка параметра event_type")
	})

	t.WithNewAsyncStep("Получение созданного лимита через Public API", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authToken),
			},
		}

		testData.turnoverLimitResponse = s.Shared.PublicClient.GetTurnoverLimits(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, testData.turnoverLimitResponse.StatusCode, "Public API: Статус-код 200")

		limit := testData.turnoverLimitResponse.Body[0]
		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ID, "Public API: Проверка параметра id")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "Public API: Проверка параметра amount")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.Currency, "Public API: Проверка параметра currency")
		sCtx.Assert().Equal(string(testData.limitMessage.IntervalType), limit.Type, "Public API: Проверка параметра type")
		sCtx.Assert().True(limit.Status, "Public API: Проверка параметра status")
		sCtx.Assert().Empty(limit.UpcomingChanges, "Public API: Проверка параметра upcomingChanges")
		sCtx.Assert().InDelta(testData.limitMessage.StartedAt, limit.StartedAt, 10, "Public API: Проверка параметра startedAt")
		sCtx.Assert().InDelta(testData.limitMessage.ExpiresAt, limit.ExpiresAt, 10, "Public API: Проверка параметра expiresAt")
		sCtx.Assert().Equal("0", limit.Spent, "Public API: Проверка параметра spent")
		sCtx.Assert().Equal(limit.Amount, limit.Rest, "Public API: Проверка параметра rest")
	})

	t.WithNewAsyncStep("Получение созданного лимита через CAP API  ", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.Shared.CapClient.GetToken(sCtx)),
				"Platform-NodeId": s.Shared.Config.Node.ProjectID,
				"Platform-Locale": capModels.DefaultLocale,
			},
			PathParams: map[string]string{
				"playerID": testData.walletAggregate.PlayerUUID,
			},
		}

		resp := s.Shared.CapClient.GetPlayerLimits(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, resp.StatusCode, "CAP API: Статус-код 200")

		turnoverLimit := resp.Body.Data[0]
		sCtx.Assert().Equal(capModels.LimitTypeTurnover, turnoverLimit.Type, "CAP API: Проверка параметра type")
		sCtx.Assert().True(turnoverLimit.Status, "CAP API: Проверка параметра status")
		sCtx.Assert().Equal(capModels.LimitPeriodDaily, turnoverLimit.Period, "CAP API: Проверка параметра period")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, turnoverLimit.Currency, "CAP API: Проверка параметра currency")
		sCtx.Assert().Equal(testData.limitMessage.Amount, turnoverLimit.Rest, "CAP API: Проверка параметра rest")
		sCtx.Assert().Equal(testData.limitMessage.Amount, turnoverLimit.Amount, "CAP API: Проверка параметра amount")
		sCtx.Assert().InDelta(testData.limitMessage.StartedAt, turnoverLimit.StartedAt, 10, "CAP API: Проверка параметра startedAt")
		sCtx.Assert().InDelta(testData.limitMessage.ExpiresAt, turnoverLimit.ExpiresAt, 10, "CAP API: Проверка параметра expiresAt")
		sCtx.Assert().Zero(turnoverLimit.DeactivatedAt, "CAP API: Проверка параметра deactivatedAt")
	})

	t.WithNewAsyncStep("Проверка данных кошелька в Redis", func(sCtx provider.StepCtx) {
		var redisValue redis.WalletFullData
		err := s.Shared.WalletRedisClient.GetWithSeqCheck(sCtx, testData.walletAggregate.WalletUUID, &redisValue, int(testData.turnoverEvent.Sequence))
		sCtx.Assert().NoError(err, "Redis: Значение кошелька получено")

		limitData := redisValue.Limits[0]
		sCtx.Assert().Equal(testData.limitMessage.ID, limitData.ExternalID, "Redis: Проверка параметра externalID")
		sCtx.Assert().Equal(redis.LimitTypeTurnoverFunds, limitData.LimitType, "Redis: Проверка параметра limitType")
		sCtx.Assert().Equal(redis.LimitPeriodDaily, limitData.IntervalType, "Redis: Проверка параметра intervalType")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limitData.Amount, "Redis: Проверка параметра amount")
		sCtx.Assert().Equal("0", limitData.Spent, "Redis: Проверка параметра spent")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limitData.Rest, "Redis: Проверка параметра rest")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limitData.CurrencyCode, "Redis: Проверка параметра currencyCode")
		sCtx.Assert().InDelta(testData.limitMessage.StartedAt, limitData.StartedAt, 10, "Redis: Проверка параметра startedAt")
		sCtx.Assert().InDelta(testData.limitMessage.ExpiresAt, limitData.ExpiresAt, 10, "Redis: Проверка параметра expiresAt")
		sCtx.Assert().True(limitData.Status, "Redis: Проверка параметра status")
	})

	t.WithNewAsyncStep("Проверка сообщения о создании лимита в топике wallet.v8.projectionSource", func(sCtx provider.StepCtx) {
		projectionMessage := kafka.FindMessageByFilter(sCtx, s.Shared.Kafka, func(msg kafka.ProjectionSourceMessage) bool {
			return msg.Type == kafka.ProjectionEventLimitChanged &&
				msg.PlayerUUID == testData.walletAggregate.PlayerUUID &&
				msg.WalletUUID == testData.walletAggregate.WalletUUID
		})
		sCtx.Assert().NotEmpty(projectionMessage.Type, "Kafka: Сообщение projectionSource найдено")
		sCtx.Assert().Equal(string(kafka.ProjectionEventLimitChanged), string(projectionMessage.Type), "Kafka: Проверка параметра type")
		sCtx.Assert().Equal(testData.turnoverEvent.Sequence, projectionMessage.SeqNumber, "Kafka: Проверка параметра seq_number")
		sCtx.Assert().Equal(testData.walletAggregate.WalletUUID, projectionMessage.WalletUUID, "Kafka: Проверка параметра wallet_uuid")
		sCtx.Assert().Equal(testData.walletAggregate.PlayerUUID, projectionMessage.PlayerUUID, "Kafka: Проверка параметра player_uuid")
		sCtx.Assert().Equal(s.Shared.Config.Node.ProjectID, projectionMessage.NodeUUID, "Kafka: Проверка параметра node_uuid")
		sCtx.Assert().Equal(s.Shared.Config.Node.DefaultCurrency, projectionMessage.Currency, "Kafka: Проверка параметра currency")
		sCtx.Assert().NotZero(projectionMessage.Timestamp, "Kafka: Проверка параметра timestamp")
		sCtx.Assert().NotEmpty(projectionMessage.SeqNumberNodeUUID, "Kafka: Проверка параметра seq_number_node_uuid")

		var limitsPayload kafka.ProjectionPayloadLimits
		err := projectionMessage.UnmarshalPayloadTo(&limitsPayload)
		sCtx.Assert().NoError(err, "Kafka: Payload успешно распакован")

		limit := limitsPayload.Limits[0]
		sCtx.Assert().Equal(string(kafka.LimitEventCreated), string(limitsPayload.EventType), "Kafka: Проверка параметра event_type")
		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ExternalID, "Kafka: Проверка параметра external_id")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "Kafka: Проверка параметра amount")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.CurrencyCode, "Kafka: Проверка параметра currency_code")
		sCtx.Assert().Equal(string(testData.limitMessage.LimitType), string(limit.LimitType), "Kafka: Проверка параметра limit_type")
		sCtx.Assert().Equal(string(testData.limitMessage.IntervalType), string(limit.IntervalType), "Kafka: Проверка параметра interval_type")
		sCtx.Assert().Equal(testData.limitMessage.StartedAt, limit.StartedAt, "Kafka: Проверка параметра started_at")
		sCtx.Assert().Equal(testData.limitMessage.ExpiresAt, limit.ExpiresAt, "Kafka: Проверка параметра expires_at")
		sCtx.Assert().True(limit.Status, "Kafka: Проверка параметра status")
	})

	t.WithNewAsyncStep("Проверка записи в таблице limit_record_v2", func(sCtx provider.StepCtx) {
		filters := map[string]interface{}{
			"external_uuid": testData.limitMessage.ID,
			"player_uuid":   testData.walletAggregate.PlayerUUID,
		}

		limitRecord := s.Shared.LimitRecordRepo.GetLimitRecordWithRetry(sCtx, filters)
		sCtx.Assert().NotNil(limitRecord, "DB: Запись о лимите найдена в таблице limit_record_v2")
		sCtx.Assert().Equal(testData.limitMessage.ID, limitRecord.ExternalUUID, "DB: Проверка параметра external_uuid")
		sCtx.Assert().Equal(testData.walletAggregate.PlayerUUID, limitRecord.PlayerUUID, "DB: Проверка параметра player_uuid")
		sCtx.Assert().Equal(string(wallet.LimitTypeTurnoverFunds), string(limitRecord.LimitType), "DB: Проверка параметра limit_type")
		sCtx.Assert().Equal(string(wallet.IntervalTypeDaily), string(limitRecord.IntervalType), "DB: Проверка параметра interval_type")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limitRecord.Amount.String(), "DB: Проверка параметра amount")
		sCtx.Assert().Equal("0", limitRecord.Spent.String(), "DB: Проверка параметра spent")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limitRecord.Rest.String(), "DB: Проверка параметра rest")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limitRecord.CurrencyCode, "DB: Проверка параметра currency_code")
		sCtx.Assert().Equal(testData.limitMessage.StartedAt, limitRecord.StartedAt, "DB: Проверка параметра started_at")
		sCtx.Assert().Equal(testData.limitMessage.ExpiresAt, limitRecord.ExpiresAt, "DB: Проверка параметра expires_at")
		sCtx.Assert().True(limitRecord.LimitStatus, "DB: Проверка параметра limit_status")
	})
}

func (s *TurnoverLimitSuite) TestTurnoverLimitUpdate(t provider.T) {
	t.Epic("Лимиты")
	t.Feature("turnover-of-funds лимит")
	t.Title("Проверка обновления лимита на оборот средств")
	t.Tags("wallet", "limits")

	var testData struct {
		authToken                      string
		walletAggregate                redis.WalletFullData
		turnoverLimitResponse          *clientTypes.Response[publicModels.GetTurnoverLimitsResponseBody]
		turnoverLimitRequest           *clientTypes.Request[publicModels.SetTurnoverLimitRequestBody]
		createLimitMessage             kafka.LimitMessage
		updateLimitMessage             kafka.LimitMessage
		turnoverCreateEvent            *nats.NatsMessage[nats.LimitChangedV2]
		turnoverUpdateEvent            *nats.NatsMessage[nats.LimitChangedV2]
		updateRecalculatedLimitRequest *clientTypes.Request[publicModels.UpdateRecalculatedLimitRequestBody]
	}

	t.WithNewStep("Создание игрока через полную регистрацию", func(sCtx provider.StepCtx) {
		playerData := defaultSteps.CreatePlayerWithFullRegistration(
			sCtx,
			s.Shared.PublicClient,
			s.Shared.Kafka,
			s.Shared.Config,
			s.Shared.PlayerRedisClient,
			s.Shared.WalletRedisClient,
		)

		sCtx.Require().NotEmpty(playerData.Auth.Body.Token, "Токен авторизации получен")
		sCtx.Require().NotEmpty(playerData.WalletData.WalletUUID, "UUID кошелька получен")
		sCtx.Require().NotEmpty(playerData.WalletData.PlayerUUID, "UUID игрока получен")

		testData.authToken = playerData.Auth.Body.Token
		testData.walletAggregate = playerData.WalletData
	})

	t.WithNewStep("Установка лимита на оборот средств", func(sCtx provider.StepCtx) {
		testData.turnoverLimitRequest = &clientTypes.Request[publicModels.SetTurnoverLimitRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authToken),
			},
			Body: &publicModels.SetTurnoverLimitRequestBody{
				Amount:    "100",
				Currency:  s.Shared.Config.Node.DefaultCurrency,
				Type:      publicModels.LimitPeriodDaily,
				StartedAt: int(time.Now().Unix()),
			},
		}

		resp := s.Shared.PublicClient.SetTurnoverLimit(sCtx, testData.turnoverLimitRequest)
		sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Лимит на оборот средств установлен")
	})

	t.WithNewStep("Получение сообщения о создании лимита на оборот средств из топика limits.v2", func(sCtx provider.StepCtx) {
		testData.createLimitMessage = kafka.FindMessageByFilter(sCtx, s.Shared.Kafka, func(msg kafka.LimitMessage) bool {
			return msg.EventType == kafka.LimitEventCreated &&
				msg.LimitType == kafka.LimitTypeTurnoverFunds &&
				msg.PlayerID == testData.walletAggregate.PlayerUUID &&
				msg.Amount == testData.turnoverLimitRequest.Body.Amount &&
				msg.CurrencyCode == testData.turnoverLimitRequest.Body.Currency
		})
		sCtx.Require().NotEmpty(testData.createLimitMessage.ID, "Сообщение о создании лимита на оборот средств из топика limits.v2 получено")
	})

	t.WithNewStep("Обновление лимита на оборот средств", func(sCtx provider.StepCtx) {
		currentAmountInt, _ := strconv.Atoi(testData.createLimitMessage.Amount)
		newAmount := strconv.Itoa(currentAmountInt - 1)

		testData.updateRecalculatedLimitRequest = &clientTypes.Request[publicModels.UpdateRecalculatedLimitRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authToken),
			},
			PathParams: map[string]string{
				"limitID": testData.createLimitMessage.ID,
			},
			Body: &publicModels.UpdateRecalculatedLimitRequestBody{
				Amount: newAmount,
			},
		}

		resp := s.Shared.PublicClient.UpdateRecalculatedLimit(sCtx, testData.updateRecalculatedLimitRequest)
		sCtx.Require().Equal(http.StatusOK, resp.StatusCode, "Лимит на оборот средств обновлен")
	})

	t.WithNewStep("Получение сообщения об обновлении лимита на оборот средств из топика limits.v2", func(sCtx provider.StepCtx) {
		testData.updateLimitMessage = kafka.FindMessageByFilter(sCtx, s.Shared.Kafka, func(msg kafka.LimitMessage) bool {
			return (msg.EventType == kafka.LimitEventUpdated || msg.EventType == kafka.LimitEventAmountUpdated) &&
				msg.LimitType == kafka.LimitTypeTurnoverFunds &&
				msg.PlayerID == testData.walletAggregate.PlayerUUID &&
				msg.ID == testData.createLimitMessage.ID &&
				msg.Amount == testData.updateRecalculatedLimitRequest.Body.Amount &&
				msg.CurrencyCode == testData.turnoverLimitRequest.Body.Currency
		})

		sCtx.Require().NotEmpty(testData.updateLimitMessage.ID, "Сообщение об обновлении лимита на оборот средств из топика limits.v2 получено")
	})

	t.WithNewStep("Проверка ивента NATS об обновлении лимита на оборот средств", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.Shared.Config.Nats.StreamPrefix, testData.walletAggregate.PlayerUUID)
		testData.turnoverUpdateEvent = nats.FindMessageInStream(sCtx, s.Shared.NatsClient, subject, func(msg nats.LimitChangedV2, msgType string) bool {
			return msg.EventType == nats.EventTypeUpdated &&
				len(msg.Limits) > 0 &&
				msg.Limits[0].LimitType == nats.LimitTypeTurnoverFunds &&
				msg.Limits[0].ExternalID == testData.createLimitMessage.ID
		})
		sCtx.Require().NotNil(testData.turnoverUpdateEvent, "Получено NATS-сообщение об обновлении лимита")

		limit := testData.turnoverUpdateEvent.Payload.Limits[0]
		sCtx.Assert().Equal(testData.updateLimitMessage.ID, limit.ExternalID, "NATS: Проверка параметра external_id")
		sCtx.Assert().Equal(testData.updateRecalculatedLimitRequest.Body.Amount, limit.Amount, "NATS: Проверка параметра amount")
		sCtx.Assert().Equal(testData.updateLimitMessage.CurrencyCode, limit.CurrencyCode, "NATS: Проверка параметра currency_code")
		sCtx.Assert().Equal(string(testData.updateLimitMessage.LimitType), limit.LimitType, "NATS: Проверка параметра limit_type")
		sCtx.Assert().True(limit.Status, "NATS: Проверка параметра status")
		sCtx.Assert().Equal(testData.updateLimitMessage.StartedAt, limit.StartedAt, "NATS: Проверка параметра started_at")
		sCtx.Assert().InDelta(testData.updateLimitMessage.ExpiresAt, limit.ExpiresAt, 10, "NATS: Проверка параметра expires_at")
		sCtx.Assert().Equal(string(testData.updateLimitMessage.IntervalType), limit.IntervalType, "NATS: Проверка параметра interval_type")
		sCtx.Assert().Equal(nats.LimitTypeTurnoverFunds, limit.LimitType, "NATS: Проверка параметра limit_type")
		sCtx.Assert().Equal(string(nats.LimitEventAmountUpdated), string(testData.turnoverUpdateEvent.Payload.EventType), "NATS: Проверка параметра event_type")
	})

	t.WithNewAsyncStep("Получение обновленного лимита через Public API", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authToken),
			},
		}

		testData.turnoverLimitResponse = s.Shared.PublicClient.GetTurnoverLimits(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, testData.turnoverLimitResponse.StatusCode, "Public API: Статус-код 200")

		limit := testData.turnoverLimitResponse.Body[0]
		sCtx.Assert().Equal(testData.updateLimitMessage.ID, limit.ID, "Public API: Проверка параметра id")
		sCtx.Assert().Equal(testData.updateRecalculatedLimitRequest.Body.Amount, limit.Amount, "Public API: Проверка параметра amount")
		sCtx.Assert().Equal(testData.updateLimitMessage.CurrencyCode, limit.Currency, "Public API: Проверка параметра currency")
		sCtx.Assert().Equal(string(testData.updateLimitMessage.IntervalType), limit.Type, "Public API: Проверка параметра type")
		sCtx.Assert().True(limit.Status, "Public API: Проверка параметра status")
		sCtx.Assert().Empty(limit.UpcomingChanges, "Public API: Проверка параметра upcomingChanges")
		sCtx.Assert().InDelta(testData.updateLimitMessage.StartedAt, limit.StartedAt, 10, "Public API: Проверка параметра startedAt")
		sCtx.Assert().InDelta(testData.updateLimitMessage.ExpiresAt, limit.ExpiresAt, 10, "Public API: Проверка параметра expiresAt")
	})

	t.WithNewAsyncStep("Получение обновленного лимита через CAP API", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.Shared.CapClient.GetToken(sCtx)),
				"Platform-NodeId": s.Shared.Config.Node.ProjectID,
				"Platform-Locale": capModels.DefaultLocale,
			},
			PathParams: map[string]string{
				"playerID": testData.walletAggregate.PlayerUUID,
			},
		}

		resp := s.Shared.CapClient.GetPlayerLimits(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, resp.StatusCode, "CAP API: Статус-код 200")

		singleBetLimit := resp.Body.Data[0]
		sCtx.Assert().Equal(capModels.LimitTypeTurnover, singleBetLimit.Type, "CAP API: Проверка параметра type")
		sCtx.Assert().True(singleBetLimit.Status, "CAP API: Проверка параметра status")
		sCtx.Assert().Equal(capModels.LimitPeriodDaily, singleBetLimit.Period, "CAP API: Проверка параметра period")
		sCtx.Assert().Equal(testData.updateLimitMessage.CurrencyCode, singleBetLimit.Currency, "CAP API: Проверка параметра currency")
		sCtx.Assert().Equal(testData.updateRecalculatedLimitRequest.Body.Amount, singleBetLimit.Rest, "CAP API: Проверка параметра rest")
		//sCtx.Assert().Equal(testData.updateRecalculatedLimitRequest.Body.Amount, singleBetLimit.Amount, "CAP API: Проверка параметра amount")
		sCtx.Assert().InDelta(testData.updateLimitMessage.StartedAt, singleBetLimit.StartedAt, 10, "CAP API: Проверка параметра startedAt")
		sCtx.Assert().InDelta(testData.updateLimitMessage.ExpiresAt, singleBetLimit.ExpiresAt, 10, "CAP API: Проверка параметра expiresAt")
		sCtx.Assert().Zero(singleBetLimit.DeactivatedAt, "CAP API: Проверка параметра deactivatedAt")
	})

	t.WithNewAsyncStep("Проверка данных кошелька в Redis", func(sCtx provider.StepCtx) {
		var redisValue redis.WalletFullData
		err := s.Shared.WalletRedisClient.GetWithSeqCheck(sCtx, testData.walletAggregate.WalletUUID, &redisValue, int(testData.turnoverUpdateEvent.Seq))
		sCtx.Assert().NoError(err, "Redis: Значение кошелька получено")

		limitData := redisValue.Limits[0]
		sCtx.Assert().Equal(testData.updateLimitMessage.ID, limitData.ExternalID, "Redis: Проверка параметра externalID")
		sCtx.Assert().Equal(redis.LimitTypeTurnoverFunds, limitData.LimitType, "Redis: Проверка параметра limitType")
		sCtx.Assert().Equal(redis.LimitPeriodDaily, limitData.IntervalType, "Redis: Проверка параметра intervalType")
		sCtx.Assert().Equal(testData.updateLimitMessage.Amount, limitData.Amount, "Redis: Проверка параметра amount")
		sCtx.Assert().Equal("0", limitData.Spent, "Redis: Проверка параметра spent")
		sCtx.Assert().Equal(testData.updateLimitMessage.CurrencyCode, limitData.CurrencyCode, "Redis: Проверка параметра currencyCode")
		sCtx.Assert().InDelta(testData.updateLimitMessage.StartedAt, limitData.StartedAt, 10, "Redis: Проверка параметра startedAt")
		sCtx.Assert().InDelta(testData.updateLimitMessage.ExpiresAt, limitData.ExpiresAt, 10, "Redis: Проверка параметра expiresAt")
		sCtx.Assert().True(limitData.Status, "Redis: Проверка параметра status")
	})

	t.WithNewAsyncStep("Проверка сообщения об обновлении лимита в топике wallet.v8.projectionSource", func(sCtx provider.StepCtx) {
		projectionMessage := kafka.FindMessageByFilter(sCtx, s.Shared.Kafka, func(msg kafka.ProjectionSourceMessage) bool {
			return msg.Type == kafka.ProjectionEventLimitChanged &&
				msg.PlayerUUID == testData.walletAggregate.PlayerUUID &&
				msg.WalletUUID == testData.walletAggregate.WalletUUID &&
				msg.SeqNumber == int(testData.turnoverUpdateEvent.Seq)
		})
		sCtx.Assert().NotEmpty(projectionMessage.Type, "Kafka: Сообщение projectionSource найдено")

		sCtx.Assert().Equal(string(kafka.ProjectionEventLimitChanged), string(projectionMessage.Type), "Kafka: Проверка параметра type")
		sCtx.Assert().Equal(testData.turnoverUpdateEvent.Sequence, projectionMessage.SeqNumber, "Kafka: Проверка параметра seq_number")
		sCtx.Assert().Equal(testData.walletAggregate.WalletUUID, projectionMessage.WalletUUID, "Kafka: Проверка параметра wallet_uuid")
		sCtx.Assert().Equal(testData.walletAggregate.PlayerUUID, projectionMessage.PlayerUUID, "Kafka: Проверка параметра player_uuid")
		sCtx.Assert().Equal(s.Shared.Config.Node.ProjectID, projectionMessage.NodeUUID, "Kafka: Проверка параметра node_uuid")
		sCtx.Assert().Equal(s.Shared.Config.Node.DefaultCurrency, projectionMessage.Currency, "Kafka: Проверка параметра currency")
		sCtx.Assert().NotZero(projectionMessage.Timestamp, "Kafka: Проверка параметра timestamp")
		sCtx.Assert().NotEmpty(projectionMessage.SeqNumberNodeUUID, "Kafka: Проверка параметра seq_number_node_uuid")

		var limitsPayload kafka.ProjectionPayloadLimits
		err := projectionMessage.UnmarshalPayloadTo(&limitsPayload)
		sCtx.Assert().NoError(err, "Kafka: Payload успешно распакован")

		limit := limitsPayload.Limits[0]
		sCtx.Assert().Equal(string(kafka.LimitEventAmountUpdated), string(limitsPayload.EventType), "Kafka: Проверка параметра event_type")
		sCtx.Assert().Equal(testData.updateLimitMessage.ID, limit.ExternalID, "Kafka: Проверка параметра external_id")
		sCtx.Assert().Equal(testData.updateRecalculatedLimitRequest.Body.Amount, limit.Amount, "Kafka: Проверка параметра amount")
		sCtx.Assert().Equal(testData.updateLimitMessage.CurrencyCode, limit.CurrencyCode, "Kafka: Проверка параметра currency_code")
		sCtx.Assert().Equal(string(testData.updateLimitMessage.LimitType), string(limit.LimitType), "Kafka: Проверка параметра limit_type")
		sCtx.Assert().Equal(string(testData.updateLimitMessage.IntervalType), string(limit.IntervalType), "Kafka: Проверка параметра interval_type")
		sCtx.Assert().Equal(testData.updateLimitMessage.StartedAt, limit.StartedAt, "Kafka: Проверка параметра started_at")
		sCtx.Assert().Equal(testData.updateLimitMessage.ExpiresAt, limit.ExpiresAt, "Kafka: Проверка параметра expires_at")
		sCtx.Assert().True(limit.Status, "Kafka: Проверка параметра status")
	})
}

func (s *TurnoverLimitSuite) TestTurnoverLimitReset(t provider.T) {
	t.Epic("Лимиты")
	t.Feature("turnover-of-funds лимит")
	t.Title("Проверка сброса лимита на оборот средств")
	t.Tags("wallet", "limits")

	var testData struct {
		authToken             string
		walletAggregate       redis.WalletFullData
		turnoverLimitResponse *clientTypes.Response[publicModels.GetTurnoverLimitsResponseBody]
		turnoverLimitRequest  *clientTypes.Request[publicModels.SetTurnoverLimitRequestBody]
		limitMessage          kafka.LimitMessage
		turnoverEvent         *nats.NatsMessage[nats.LimitChangedV2]
		turnoverEventReset    *nats.NatsMessage[nats.LimitChangedV2]
	}

	t.WithNewStep("Создание игрока через полную регистрацию", func(sCtx provider.StepCtx) {
		playerData := defaultSteps.CreatePlayerWithFullRegistration(
			sCtx,
			s.Shared.PublicClient,
			s.Shared.Kafka,
			s.Shared.Config,
			s.Shared.PlayerRedisClient,
			s.Shared.WalletRedisClient,
		)

		sCtx.Require().NotEmpty(playerData.Auth.Body.Token, "Токен авторизации получен")
		sCtx.Require().NotEmpty(playerData.WalletData.WalletUUID, "UUID кошелька получен")
		sCtx.Require().NotEmpty(playerData.WalletData.PlayerUUID, "UUID игрока получен")

		testData.authToken = playerData.Auth.Body.Token
		testData.walletAggregate = playerData.WalletData
	})

	t.WithNewStep("Установка лимита на оборот средств", func(sCtx provider.StepCtx) {
		testData.turnoverLimitRequest = &clientTypes.Request[publicModels.SetTurnoverLimitRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authToken),
			},
			Body: &publicModels.SetTurnoverLimitRequestBody{
				Amount:    "100",
				Currency:  s.Shared.Config.Node.DefaultCurrency,
				Type:      publicModels.LimitPeriodDaily,
				StartedAt: int(time.Now().Unix() - 86395),
			},
		}

		resp := s.Shared.PublicClient.SetTurnoverLimit(sCtx, testData.turnoverLimitRequest)
		sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Лимит на оборот средств установлен")
	})

	t.WithNewStep("Получение сообщения о создании лимита на оборот средств из топика limits.v2", func(sCtx provider.StepCtx) {
		testData.limitMessage = kafka.FindMessageByFilter(sCtx, s.Shared.Kafka, func(msg kafka.LimitMessage) bool {
			return msg.EventType == kafka.LimitEventCreated &&
				msg.LimitType == kafka.LimitTypeTurnoverFunds &&
				msg.IntervalType == kafka.IntervalTypeDaily &&
				msg.PlayerID == testData.walletAggregate.PlayerUUID &&
				msg.Amount == testData.turnoverLimitRequest.Body.Amount &&
				msg.CurrencyCode == testData.turnoverLimitRequest.Body.Currency
		})
		sCtx.Require().NotEmpty(testData.limitMessage.ID, "Сообщение о создании лимита на оборот средств из топика limits.v2 получено")
	})

	t.WithNewStep("Получение ивента NATS о создании лимита на оборот средств", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.Shared.Config.Nats.StreamPrefix, testData.walletAggregate.PlayerUUID)
		testData.turnoverEvent = nats.FindMessageInStream(sCtx, s.Shared.NatsClient, subject, func(data nats.LimitChangedV2, msgType string) bool {
			return data.EventType == nats.EventTypeCreated &&
				len(data.Limits) > 0 &&
				data.Limits[0].LimitType == nats.LimitTypeTurnoverFunds
		})
		sCtx.Require().NotNil(testData.turnoverEvent, "Должно получиться получить сообщение из NATS")
	})

	t.WithNewStep("Выполнение корректировки баланса в положительную сторону", func(sCtx provider.StepCtx) {
		time.Sleep(time.Second * 5)
		req := &clientTypes.Request[capModels.CreateBalanceAdjustmentRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.Shared.CapClient.GetToken(sCtx)),
				"Platform-Locale": capModels.DefaultLocale,
				"Platform-NodeID": s.Shared.Config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"player_uuid": testData.walletAggregate.PlayerUUID,
			},
			Body: &capModels.CreateBalanceAdjustmentRequestBody{
				Currency:      s.Shared.Config.Node.DefaultCurrency,
				Amount:        100.0,
				Reason:        capModels.ReasonOperationalMistake,
				OperationType: capModels.OperationTypeDeposit,
				Direction:     capModels.DirectionIncrease,
				Comment:       utils.Get(utils.LETTERS, 25),
			},
		}

		resp := s.Shared.CapClient.CreateBalanceAdjustment(sCtx, req)
		sCtx.Require().Equal(http.StatusOK, resp.StatusCode, "Статус код ответа равен 200")
	})

	t.WithNewStep("Проверка ивента NATS о сбросе лимита на оборот средств", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.Shared.Config.Nats.StreamPrefix, testData.walletAggregate.PlayerUUID)
		testData.turnoverEventReset = nats.FindMessageInStream(sCtx, s.Shared.NatsClient, subject, func(data nats.LimitChangedV2, msgType string) bool {
			return data.EventType == nats.EventTypeSpentResetted &&
				len(data.Limits) > 0 &&
				data.Limits[0].LimitType == nats.LimitTypeTurnoverFunds
		})
		sCtx.Require().NotNil(testData.turnoverEventReset, "Должно получиться получить сообщение из NATS")

		limit := testData.turnoverEventReset.Payload.Limits[0]
		eventType := testData.turnoverEventReset.Payload.EventType
		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ExternalID, "NATS: Проверка параметра external_id")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "NATS: Проверка параметра amount")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.CurrencyCode, "NATS: Проверка параметра currency_code")
		sCtx.Assert().Equal(string(testData.limitMessage.LimitType), limit.LimitType, "NATS: Проверка параметра limit_type")
		sCtx.Assert().True(limit.Status, "NATS: Проверка параметра status")
		sCtx.Assert().Equal(testData.turnoverEvent.Payload.Limits[0].ExpiresAt, limit.StartedAt, "NATS: Проверка параметра started_at")
		sCtx.Assert().Equal(testData.turnoverEvent.Payload.Limits[0].ExpiresAt+86400, limit.ExpiresAt, "NATS: Проверка параметра expires_at")
		sCtx.Assert().Equal(string(testData.limitMessage.IntervalType), limit.IntervalType, "NATS: Проверка параметра interval_type")
		sCtx.Assert().Equal(nats.LimitTypeTurnoverFunds, limit.LimitType, "NATS: Проверка параметра limit_type")
		sCtx.Assert().Equal(eventType, testData.turnoverEventReset.Payload.EventType, "NATS: Проверка параметра event_type")
	})

	t.WithNewAsyncStep("Получение сброшенного лимита через Public API", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authToken),
			},
		}

		testData.turnoverLimitResponse = s.Shared.PublicClient.GetTurnoverLimits(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, testData.turnoverLimitResponse.StatusCode, "Public API: Статус-код 200")

		limit := testData.turnoverLimitResponse.Body[0]
		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ID, "Public API: Проверка параметра id")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "Public API: Проверка параметра amount")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.Currency, "Public API: Проверка параметра currency")
		sCtx.Assert().Equal(string(testData.limitMessage.IntervalType), string(limit.Type), "Public API: Проверка параметра type")
		sCtx.Assert().True(limit.Status, "Public API: Проверка параметра status")
		sCtx.Assert().Empty(limit.UpcomingChanges, "Public API: Проверка параметра upcomingChanges")
		sCtx.Assert().Nil(limit.DeactivatedAt, "Public API: Проверка параметра deactivatedAt")
		sCtx.Assert().Equal(testData.turnoverEventReset.Payload.Limits[0].StartedAt, limit.StartedAt, "Public API: Проверка параметра startedAt")
		sCtx.Assert().Equal(testData.turnoverEventReset.Payload.Limits[0].ExpiresAt, limit.ExpiresAt, "Public API: Проверка параметра expiresAt")
		sCtx.Assert().Equal("0", limit.Spent, "Public API: Проверка параметра spent")
		sCtx.Assert().Equal(limit.Amount, limit.Rest, "Public API: Проверка параметра rest")
	})

	t.WithNewAsyncStep("Получение сброшенного лимита через CAP API", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.Shared.CapClient.GetToken(sCtx)),
				"Platform-NodeId": s.Shared.Config.Node.ProjectID,
				"Platform-Locale": capModels.DefaultLocale,
			},
			PathParams: map[string]string{
				"playerID": testData.walletAggregate.PlayerUUID,
			},
		}

		resp := s.Shared.CapClient.GetPlayerLimits(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, resp.StatusCode, "CAP API: Статус-код 200")

		turnoverLimit := resp.Body.Data[0]
		sCtx.Assert().Equal(capModels.LimitTypeTurnover, turnoverLimit.Type, "CAP API: Проверка параметра type")
		sCtx.Assert().True(turnoverLimit.Status, "CAP API: Проверка параметра status")
		sCtx.Assert().Equal(capModels.LimitPeriodDaily, turnoverLimit.Period, "CAP API: Проверка параметра period")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, turnoverLimit.Currency, "CAP API: Проверка параметра currency")
		sCtx.Assert().Equal(testData.limitMessage.Amount, turnoverLimit.Rest, "CAP API: Проверка параметра rest")
		sCtx.Assert().Equal(testData.limitMessage.Amount, turnoverLimit.Amount, "CAP API: Проверка параметра amount")
		sCtx.Assert().InDelta(testData.turnoverEventReset.Payload.Limits[0].StartedAt, turnoverLimit.StartedAt, 10, "CAP API: Проверка параметра startedAt")
		sCtx.Assert().InDelta(testData.turnoverEventReset.Payload.Limits[0].ExpiresAt, turnoverLimit.ExpiresAt, 10, "CAP API: Проверка параметра expiresAt")
		sCtx.Assert().Zero(turnoverLimit.DeactivatedAt, "CAP API: Проверка параметра deactivatedAt")
	})

	t.WithNewAsyncStep("Проверка данных кошелька в Redis", func(sCtx provider.StepCtx) {
		var redisValue redis.WalletFullData
		err := s.Shared.WalletRedisClient.GetWithRetry(sCtx, testData.walletAggregate.WalletUUID, &redisValue)
		sCtx.Assert().NoError(err, "Redis: Получение данных кошелька")

		limitData := redisValue.Limits[0]
		sCtx.Assert().Equal(testData.limitMessage.ID, limitData.ExternalID, "Redis: Проверка параметра externalID")
		sCtx.Assert().Equal(redis.LimitTypeTurnoverFunds, limitData.LimitType, "Redis: Проверка параметра limitType")
		sCtx.Assert().Equal(redis.LimitPeriodDaily, limitData.IntervalType, "Redis: Проверка параметра intervalType")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limitData.Amount, "Redis: Проверка параметра amount")
		sCtx.Assert().Equal("0", limitData.Spent, "Redis: Проверка параметра spent")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limitData.Rest, "Redis: Проверка параметра rest")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limitData.CurrencyCode, "Redis: Проверка параметра currencyCode")
		sCtx.Assert().InDelta(testData.turnoverEventReset.Payload.Limits[0].StartedAt, limitData.StartedAt, 10, "Redis: Проверка параметра startedAt")
		sCtx.Assert().InDelta(testData.turnoverEventReset.Payload.Limits[0].ExpiresAt, limitData.ExpiresAt, 10, "Redis: Проверка параметра expiresAt")
		sCtx.Assert().True(limitData.Status, "Redis: Проверка параметра status")
	})

	t.WithNewAsyncStep("Проверка сообщения о ресете лимита в топике wallet.v8.projectionSource", func(sCtx provider.StepCtx) {
		projectionMessage := kafka.FindMessageByFilter(sCtx, s.Shared.Kafka, func(msg kafka.ProjectionSourceMessage) bool {
			return msg.Type == kafka.ProjectionEventLimitChanged &&
				msg.PlayerUUID == testData.walletAggregate.PlayerUUID &&
				msg.WalletUUID == testData.walletAggregate.WalletUUID &&
				msg.SeqNumber == int(testData.turnoverEventReset.Seq)
		})
		sCtx.Assert().NotEmpty(projectionMessage.Type, "Kafka: Сообщение projectionSource найдено")
		sCtx.Assert().Equal(string(kafka.ProjectionEventLimitChanged), string(projectionMessage.Type), "Kafka: Проверка параметра type")
		sCtx.Assert().Equal(testData.turnoverEventReset.Sequence, projectionMessage.SeqNumber, "Kafka: Проверка параметра seq_number")
		sCtx.Assert().Equal(testData.walletAggregate.WalletUUID, projectionMessage.WalletUUID, "Kafka: Проверка параметра wallet_uuid")
		sCtx.Assert().Equal(testData.walletAggregate.PlayerUUID, projectionMessage.PlayerUUID, "Kafka: Проверка параметра player_uuid")
		sCtx.Assert().Equal(s.Shared.Config.Node.ProjectID, projectionMessage.NodeUUID, "Kafka: Проверка параметра node_uuid")
		sCtx.Assert().Equal(s.Shared.Config.Node.DefaultCurrency, projectionMessage.Currency, "Kafka: Проверка параметра currency")
		sCtx.Assert().Equal(projectionMessage.Timestamp, int(testData.turnoverEventReset.Timestamp.Unix()), "Kafka: Проверка параметра timestamp")
		sCtx.Assert().NotEmpty(projectionMessage.SeqNumberNodeUUID, "Kafka: Проверка параметра seq_number_node_uuid")

		var limitsPayload kafka.ProjectionPayloadLimits
		err := projectionMessage.UnmarshalPayloadTo(&limitsPayload)
		sCtx.Assert().NoError(err, "Kafka: Payload успешно распакован")

		limit := limitsPayload.Limits[0]
		sCtx.Assert().Equal(string(kafka.LimitEventSpentResetted), string(limitsPayload.EventType), "Kafka: Проверка параметра event_type")
		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ExternalID, "Kafka: Проверка параметра external_id")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "Kafka: Проверка параметра amount")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.CurrencyCode, "Kafka: Проверка параметра currency_code")
		sCtx.Assert().Equal(string(testData.limitMessage.LimitType), string(limit.LimitType), "Kafka: Проверка параметра limit_type")
		sCtx.Assert().Equal(string(testData.limitMessage.IntervalType), string(limit.IntervalType), "Kafka: Проверка параметра interval_type")
		sCtx.Assert().Equal(testData.turnoverEventReset.Payload.Limits[0].StartedAt, limit.StartedAt, "Kafka: Проверка параметра started_at")
		sCtx.Assert().Equal(testData.turnoverEventReset.Payload.Limits[0].ExpiresAt, limit.ExpiresAt, "Kafka: Проверка параметра expires_at")
		sCtx.Assert().True(limit.Status, "Kafka: Проверка параметра status")
	})
}
