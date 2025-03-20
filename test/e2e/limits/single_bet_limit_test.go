package test

import (
	"fmt"
	"net/http"
	"strconv"

	capModels "CB_auto/internal/client/cap/models"
	publicModels "CB_auto/internal/client/public/models"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/repository/wallet"
	"CB_auto/internal/transport/kafka"
	"CB_auto/internal/transport/nats"
	"CB_auto/internal/transport/redis"
	defaultSteps "CB_auto/pkg/utils/default_steps"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type SingleBetLimitSuite struct {
	suite.Suite
	Shared *SharedConnections
}

func (s *SingleBetLimitSuite) SetShared(sc *SharedConnections) {
	s.Shared = sc
}

func (s *SingleBetLimitSuite) TestSingleBetLimitCreate(t provider.T) {
	t.Epic("Лимиты")
	t.Feature("single-bet лимит")
	t.Title("Проверка создания лимита на одиночную ставку")
	t.Tags("wallet", "limits")

	var testData struct {
		authToken              string
		walletAggregate        redis.WalletFullData
		singleBetLimitResponse *clientTypes.Response[publicModels.GetSingleBetLimitsResponseBody]
		singleBetLimitRequest  *clientTypes.Request[publicModels.SetSingleBetLimitRequestBody]
		limitMessage           kafka.LimitMessage
		singleBetEvent         *nats.NatsMessage[nats.LimitChangedV2]
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

	t.WithNewStep("Установка лимита на одиночную ставку", func(sCtx provider.StepCtx) {
		testData.singleBetLimitRequest = &clientTypes.Request[publicModels.SetSingleBetLimitRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authToken),
			},
			Body: &publicModels.SetSingleBetLimitRequestBody{
				Amount:   "100",
				Currency: s.Shared.Config.Node.DefaultCurrency,
			},
		}

		resp := s.Shared.PublicClient.SetSingleBetLimit(sCtx, testData.singleBetLimitRequest)
		sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Лимит на одиночную ставку установлен")
	})

	t.WithNewStep("Получение сообщения о создании лимита на одиночную ставку из топика limits.v2", func(sCtx provider.StepCtx) {
		testData.limitMessage = kafka.FindMessageByFilter(sCtx, s.Shared.Kafka, func(msg kafka.LimitMessage) bool {
			return msg.EventType == kafka.LimitEventCreated &&
				msg.LimitType == kafka.LimitTypeSingleBet &&
				msg.PlayerID == testData.walletAggregate.PlayerUUID &&
				msg.Amount == testData.singleBetLimitRequest.Body.Amount &&
				msg.CurrencyCode == testData.singleBetLimitRequest.Body.Currency
		})
		sCtx.Require().NotEmpty(testData.limitMessage.ID, "Сообщение о создании лимита на одиночную ставку из топика limits.v2 получено")
	})

	t.WithNewStep("Проверка ивента NATS о создании лимита на одиночную ставку", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.Shared.Config.Nats.StreamPrefix, testData.walletAggregate.PlayerUUID)
		testData.singleBetEvent = nats.FindMessageInStream(sCtx, s.Shared.NatsClient, subject, func(msg nats.LimitChangedV2, msgType string) bool {
			return msg.EventType == nats.EventTypeCreated &&
				len(msg.Limits) > 0 &&
				msg.Limits[0].LimitType == nats.LimitTypeSingleBet
		})
		sCtx.Require().NotNil(testData.singleBetEvent, "Получено NATS-сообщение о создании лимита")

		limit := testData.singleBetEvent.Payload.Limits[0]
		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ExternalID, "NATS: Проверка параметра external_id")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "NATS: Проверка параметра amount")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.CurrencyCode, "NATS: Проверка параметра currency_code")
		sCtx.Assert().Equal(string(testData.limitMessage.LimitType), limit.LimitType, "NATS: Проверка параметра limit_type")
		sCtx.Assert().True(limit.Status, "NATS: Проверка параметра status")
		sCtx.Assert().Equal(testData.limitMessage.StartedAt, limit.StartedAt, "NATS: Проверка параметра started_at")
		sCtx.Assert().Zero(limit.ExpiresAt, "NATS: Проверка параметра expires_at")
		sCtx.Assert().Equal(nats.LimitTypeSingleBet, limit.LimitType, "NATS: Проверка параметра limit_type")
		sCtx.Assert().Equal(string(testData.singleBetEvent.Payload.EventType), string(testData.limitMessage.EventType), "NATS: Проверка параметра event_type")
	})

	t.WithNewAsyncStep("Получение созданного лимита через Public API", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authToken),
			},
		}

		testData.singleBetLimitResponse = s.Shared.PublicClient.GetSingleBetLimits(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, testData.singleBetLimitResponse.StatusCode, "Public API: Статус-код 200")

		limit := testData.singleBetLimitResponse.Body[0]
		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ID, "Public API: Проверка параметра id")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "Public API: Проверка параметра amount")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.Currency, "Public API: Проверка параметра currency")
		sCtx.Assert().True(limit.Status, "Public API: Проверка параметра status")
		sCtx.Assert().True(limit.Required, "Public API: Проверка параметра required")
		sCtx.Assert().Empty(limit.UpcomingChanges, "Public API: Проверка параметра upcomingChanges")
		sCtx.Assert().Nil(limit.DeactivatedAt, "Public API: Проверка параметра deactivatedAt")
	})

	t.WithNewAsyncStep("Получение созданного лимита через CAP API", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.Shared.CapClient.GetToken(sCtx)),
				"Platform-NodeId": s.Shared.Config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"playerID": testData.walletAggregate.PlayerUUID,
			},
		}

		resp := s.Shared.CapClient.GetPlayerLimits(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, resp.StatusCode, "CAP API: Статус-код 200")

		singleBetLimit := resp.Body.Data[0]
		sCtx.Assert().Equal(capModels.LimitTypeSingleBet, singleBetLimit.Type, "CAP API: Проверка параметра type")
		sCtx.Assert().True(singleBetLimit.Status, "CAP API: Проверка параметра status")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, singleBetLimit.Currency, "CAP API: Проверка параметра currency")
		sCtx.Assert().Equal(testData.limitMessage.Amount, singleBetLimit.Rest, "CAP API: Проверка параметра rest")
		sCtx.Assert().Equal(testData.limitMessage.Amount, singleBetLimit.Amount, "CAP API: Проверка параметра amount")
		sCtx.Assert().InDelta(testData.limitMessage.StartedAt, singleBetLimit.StartedAt, 10, "CAP API: Проверка параметра startedAt")
		sCtx.Assert().InDelta(testData.limitMessage.ExpiresAt, singleBetLimit.ExpiresAt, 10, "CAP API: Проверка параметра expiresAt")
		sCtx.Assert().Zero(singleBetLimit.DeactivatedAt, "CAP API: Проверка параметра deactivatedAt")
	})

	t.WithNewAsyncStep("Проверка данных кошелька в Redis", func(sCtx provider.StepCtx) {
		var redisValue redis.WalletFullData
		err := s.Shared.WalletRedisClient.GetWithSeqCheck(sCtx, testData.walletAggregate.WalletUUID, &redisValue, int(testData.singleBetEvent.Seq))
		sCtx.Assert().NoError(err, "Redis: Значение кошелька получено")

		limitData := redisValue.Limits[0]
		sCtx.Assert().Equal(testData.limitMessage.ID, limitData.ExternalID, "Redis: Проверка параметра externalID")
		sCtx.Assert().Equal(redis.LimitTypeSingleBet, limitData.LimitType, "Redis: Проверка параметра limitType")
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
		sCtx.Assert().Equal(testData.singleBetEvent.Sequence, projectionMessage.SeqNumber, "Kafka: Проверка параметра seq_number")
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
		sCtx.Assert().Equal(string(wallet.LimitTypeSingleBet), string(limitRecord.LimitType), "DB: Проверка параметра limit_type")
		sCtx.Assert().Equal("0", limitRecord.Spent.String(), "DB: Проверка параметра spent")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limitRecord.Amount.String(), "DB: Проверка параметра amount")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limitRecord.Rest.String(), "DB: Проверка параметра rest")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limitRecord.CurrencyCode, "DB: Проверка параметра currency_code")
		sCtx.Assert().Equal(testData.limitMessage.StartedAt, limitRecord.StartedAt, "DB: Проверка параметра started_at")
		sCtx.Assert().Equal(testData.limitMessage.ExpiresAt, limitRecord.ExpiresAt, "DB: Проверка параметра expires_at")
		sCtx.Assert().True(limitRecord.LimitStatus, "DB: Проверка параметра limit_status")
	})
}

func (s *SingleBetLimitSuite) TestSingleBetLimitUpdate(t provider.T) {
	t.Epic("Лимиты")
	t.Feature("single-bet лимит")
	t.Title("Проверка обновления лимита на одиночную ставку")
	t.Tags("wallet", "limits")

	var testData struct {
		authToken                  string
		walletAggregate            redis.WalletFullData
		singleBetLimitResponse     *clientTypes.Response[publicModels.GetSingleBetLimitsResponseBody]
		singleBetLimitRequest      *clientTypes.Request[publicModels.SetSingleBetLimitRequestBody]
		createLimitMessage         kafka.LimitMessage
		updateLimitMessage         kafka.LimitMessage
		singleBetCreateEvent       *nats.NatsMessage[nats.LimitChangedV2]
		singleBetUpdateEvent       *nats.NatsMessage[nats.LimitChangedV2]
		updateSingeBetLimitRequest *clientTypes.Request[publicModels.UpdateSingleBetLimitRequestBody]
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

	t.WithNewStep("Установка лимита на одиночную ставку", func(sCtx provider.StepCtx) {
		testData.singleBetLimitRequest = &clientTypes.Request[publicModels.SetSingleBetLimitRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authToken),
			},
			Body: &publicModels.SetSingleBetLimitRequestBody{
				Amount:   "100",
				Currency: s.Shared.Config.Node.DefaultCurrency,
			},
		}

		resp := s.Shared.PublicClient.SetSingleBetLimit(sCtx, testData.singleBetLimitRequest)
		sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Лимит на одиночную ставку установлен")
	})

	t.WithNewStep("Получение сообщения о создании лимита на одиночную ставку из топика limits.v2", func(sCtx provider.StepCtx) {
		testData.createLimitMessage = kafka.FindMessageByFilter(sCtx, s.Shared.Kafka, func(msg kafka.LimitMessage) bool {
			return msg.EventType == kafka.LimitEventCreated &&
				msg.LimitType == kafka.LimitTypeSingleBet &&
				msg.PlayerID == testData.walletAggregate.PlayerUUID &&
				msg.Amount == testData.singleBetLimitRequest.Body.Amount &&
				msg.CurrencyCode == testData.singleBetLimitRequest.Body.Currency
		})

		sCtx.Require().NotEmpty(testData.createLimitMessage.ID, "Сообщение о создании лимита на одиночную ставку из топика limits.v2 получено")
	})

	t.WithNewStep("Обновление лимита на одиночную ставку", func(sCtx provider.StepCtx) {
		currentAmountInt, _ := strconv.Atoi(testData.createLimitMessage.Amount)
		newAmount := strconv.Itoa(currentAmountInt - 1)

		testData.updateSingeBetLimitRequest = &clientTypes.Request[publicModels.UpdateSingleBetLimitRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authToken),
			},
			PathParams: map[string]string{
				"limitID": testData.createLimitMessage.ID,
			},
			Body: &publicModels.UpdateSingleBetLimitRequestBody{
				Amount: newAmount,
			},
		}

		resp := s.Shared.PublicClient.UpdateSingleBetLimit(sCtx, testData.updateSingeBetLimitRequest)
		sCtx.Require().Equal(http.StatusOK, resp.StatusCode, "Лимит на одиночную ставку обновлен")
	})

	t.WithNewStep("Получение сообщения об обновлении лимита на одиночную ставку из топика limits.v2", func(sCtx provider.StepCtx) {
		testData.updateLimitMessage = kafka.FindMessageByFilter(sCtx, s.Shared.Kafka, func(msg kafka.LimitMessage) bool {
			return msg.EventType == kafka.LimitEventUpdated &&
				msg.LimitType == kafka.LimitTypeSingleBet &&
				msg.PlayerID == testData.walletAggregate.PlayerUUID &&
				msg.ID == testData.createLimitMessage.ID &&
				msg.Amount == testData.updateSingeBetLimitRequest.Body.Amount &&
				msg.CurrencyCode == testData.singleBetLimitRequest.Body.Currency
		})

		sCtx.Require().NotEmpty(testData.updateLimitMessage.ID, "Сообщение об обновлении лимита на одиночную ставку из топика limits.v2 получено")
	})

	t.WithNewStep("Проверка ивента NATS об обновлении лимита на одиночную ставку", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.Shared.Config.Nats.StreamPrefix, testData.walletAggregate.PlayerUUID)
		testData.singleBetUpdateEvent = nats.FindMessageInStream(sCtx, s.Shared.NatsClient, subject, func(msg nats.LimitChangedV2, msgType string) bool {
			return msg.EventType == nats.EventTypeUpdated &&
				len(msg.Limits) > 0 &&
				msg.Limits[0].LimitType == nats.LimitTypeSingleBet &&
				msg.Limits[0].ExternalID == testData.createLimitMessage.ID
		})
		sCtx.Require().NotNil(testData.singleBetUpdateEvent, "Получено NATS-сообщение об обновлении лимита")

		limit := testData.singleBetUpdateEvent.Payload.Limits[0]
		sCtx.Assert().Equal(testData.updateLimitMessage.ID, limit.ExternalID, "NATS: Проверка параметра external_id")
		sCtx.Assert().Equal(testData.updateSingeBetLimitRequest.Body.Amount, limit.Amount, "NATS: Проверка параметра amount")
		sCtx.Assert().Equal(testData.updateLimitMessage.CurrencyCode, limit.CurrencyCode, "NATS: Проверка параметра currency_code")
		sCtx.Assert().Equal(string(testData.updateLimitMessage.LimitType), limit.LimitType, "NATS: Проверка параметра limit_type")
		sCtx.Assert().True(limit.Status, "NATS: Проверка параметра status")
		sCtx.Assert().Equal(testData.updateLimitMessage.StartedAt, limit.StartedAt, "NATS: Проверка параметра started_at")
		sCtx.Assert().Zero(limit.ExpiresAt, "NATS: Проверка параметра expires_at")
		sCtx.Assert().Equal(nats.LimitTypeSingleBet, limit.LimitType, "NATS: Проверка параметра limit_type")
		sCtx.Assert().Equal(string(nats.LimitEventAmountUpdated), string(testData.singleBetUpdateEvent.Payload.EventType), "NATS: Проверка параметра event_type")
	})

	t.WithNewAsyncStep("Получение созданного лимита через Public API", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authToken),
			},
		}

		testData.singleBetLimitResponse = s.Shared.PublicClient.GetSingleBetLimits(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, testData.singleBetLimitResponse.StatusCode, "Public API: Статус-код 200")

		limit := testData.singleBetLimitResponse.Body[0]
		sCtx.Assert().Equal(testData.updateLimitMessage.ID, limit.ID, "Public API: Проверка параметра id")
		sCtx.Assert().Equal(testData.updateSingeBetLimitRequest.Body.Amount, limit.Amount, "Public API: Проверка параметра amount")
		sCtx.Assert().Equal(testData.updateLimitMessage.CurrencyCode, limit.Currency, "Public API: Проверка параметра currency")
		sCtx.Assert().True(limit.Status, "Public API: Проверка параметра status")
		sCtx.Assert().True(limit.Required, "Public API: Проверка параметра required")
		sCtx.Assert().Empty(limit.UpcomingChanges, "Public API: Проверка параметра upcomingChanges")
		sCtx.Assert().Nil(limit.DeactivatedAt, "Public API: Проверка параметра deactivatedAt")
	})

	t.WithNewAsyncStep("Получение созданного лимита через CAP API", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.Shared.CapClient.GetToken(sCtx)),
				"Platform-NodeId": s.Shared.Config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"playerID": testData.walletAggregate.PlayerUUID,
			},
		}

		resp := s.Shared.CapClient.GetPlayerLimits(sCtx, req)
		sCtx.Assert().Equal(http.StatusOK, resp.StatusCode, "CAP API: Статус-код 200")

		singleBetLimit := resp.Body.Data[0]
		sCtx.Assert().Equal(capModels.LimitTypeSingleBet, singleBetLimit.Type, "CAP API: Проверка параметра type")
		sCtx.Assert().True(singleBetLimit.Status, "CAP API: Проверка параметра status")
		sCtx.Assert().Equal(testData.updateLimitMessage.CurrencyCode, singleBetLimit.Currency, "CAP API: Проверка параметра currency")
		sCtx.Assert().Equal("0", singleBetLimit.Rest, "CAP API: Проверка параметра rest")
		sCtx.Assert().Equal(testData.updateSingeBetLimitRequest.Body.Amount, singleBetLimit.Amount, "CAP API: Проверка параметра amount")
		sCtx.Assert().InDelta(testData.updateLimitMessage.StartedAt, singleBetLimit.StartedAt, 10, "CAP API: Проверка параметра startedAt")
		sCtx.Assert().InDelta(testData.updateLimitMessage.ExpiresAt, singleBetLimit.ExpiresAt, 10, "CAP API: Проверка параметра expiresAt")
		sCtx.Assert().Zero(singleBetLimit.DeactivatedAt, "CAP API: Проверка параметра deactivatedAt")
	})

	t.WithNewAsyncStep("Проверка данных кошелька в Redis", func(sCtx provider.StepCtx) {
		var redisValue redis.WalletFullData
		err := s.Shared.WalletRedisClient.GetWithSeqCheck(sCtx, testData.walletAggregate.WalletUUID, &redisValue, int(testData.singleBetUpdateEvent.Seq))
		sCtx.Assert().NoError(err, "Redis: Значение кошелька получено")

		limitData := redisValue.Limits[0]
		sCtx.Assert().Equal(testData.updateLimitMessage.ID, limitData.ExternalID, "Redis: Проверка параметра externalID")
		sCtx.Assert().Equal(redis.LimitTypeSingleBet, limitData.LimitType, "Redis: Проверка параметра limitType")
		sCtx.Assert().Equal(testData.updateLimitMessage.Amount, limitData.Amount, "Redis: Проверка параметра amount")
		sCtx.Assert().Equal("0", limitData.Spent, "Redis: Проверка параметра spent")
		//sCtx.Assert().Equal("0", limitData.Rest, "Redis: Проверка параметра rest")
		sCtx.Assert().Equal(testData.updateLimitMessage.CurrencyCode, limitData.CurrencyCode, "Redis: Проверка параметра currencyCode")
		sCtx.Assert().InDelta(testData.updateLimitMessage.StartedAt, limitData.StartedAt, 10, "Redis: Проверка параметра startedAt")
		sCtx.Assert().InDelta(testData.updateLimitMessage.ExpiresAt, limitData.ExpiresAt, 10, "Redis: Проверка параметра expiresAt")
		sCtx.Assert().True(limitData.Status, "Redis: Проверка параметра status")
	})

	t.WithNewAsyncStep("Проверка сообщения о создании лимита в топике wallet.v8.projectionSource", func(sCtx provider.StepCtx) {
		projectionMessage := kafka.FindMessageByFilter(sCtx, s.Shared.Kafka, func(msg kafka.ProjectionSourceMessage) bool {
			return msg.Type == kafka.ProjectionEventLimitChanged &&
				msg.PlayerUUID == testData.walletAggregate.PlayerUUID &&
				msg.WalletUUID == testData.walletAggregate.WalletUUID &&
				msg.SeqNumber == int(testData.singleBetUpdateEvent.Seq)
		})
		sCtx.Assert().NotEmpty(projectionMessage.Type, "Kafka: Сообщение projectionSource найдено")

		sCtx.Assert().Equal(string(kafka.ProjectionEventLimitChanged), string(projectionMessage.Type), "Kafka: Проверка параметра type")
		sCtx.Assert().Equal(testData.singleBetUpdateEvent.Sequence, projectionMessage.SeqNumber, "Kafka: Проверка параметра seq_number")
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
		sCtx.Assert().Equal(testData.updateSingeBetLimitRequest.Body.Amount, limit.Amount, "Kafka: Проверка параметра amount")
		sCtx.Assert().Equal(testData.updateLimitMessage.CurrencyCode, limit.CurrencyCode, "Kafka: Проверка параметра currency_code")
		sCtx.Assert().Equal(string(testData.updateLimitMessage.LimitType), string(limit.LimitType), "Kafka: Проверка параметра limit_type")
		sCtx.Assert().Equal(string(testData.updateLimitMessage.IntervalType), string(limit.IntervalType), "Kafka: Проверка параметра interval_type")
		sCtx.Assert().Equal(testData.updateLimitMessage.StartedAt, limit.StartedAt, "Kafka: Проверка параметра started_at")
		sCtx.Assert().Equal(testData.updateLimitMessage.ExpiresAt, limit.ExpiresAt, "Kafka: Проверка параметра expires_at")
		sCtx.Assert().True(limit.Status, "Kafka: Проверка параметра status")
	})

	t.WithNewAsyncStep("Проверка записи в таблице limit_record_v2", func(sCtx provider.StepCtx) {
		filters := map[string]interface{}{
			"external_uuid": testData.createLimitMessage.ID,
			"player_uuid":   testData.walletAggregate.PlayerUUID,
		}

		limitRecord := s.Shared.LimitRecordRepo.GetLimitRecordWithRetry(sCtx, filters)
		sCtx.Assert().NotNil(limitRecord, "DB: Запись о лимите найдена в таблице limit_record_v2")
		sCtx.Assert().Equal(testData.updateLimitMessage.ID, limitRecord.ExternalUUID, "DB: Проверка параметра external_uuid")
		sCtx.Assert().Equal(testData.walletAggregate.PlayerUUID, limitRecord.PlayerUUID, "DB: Проверка параметра player_uuid")
		sCtx.Assert().Equal(string(wallet.LimitTypeSingleBet), string(limitRecord.LimitType), "DB: Проверка параметра limit_type")
		sCtx.Assert().Equal("0", limitRecord.Spent.String(), "DB: Проверка параметра spent")
		sCtx.Assert().Equal(testData.updateSingeBetLimitRequest.Body.Amount, limitRecord.Amount.String(), "DB: Проверка параметра amount")
		sCtx.Assert().Equal("0", limitRecord.Rest.String(), "DB: Проверка параметра rest")
		sCtx.Assert().Equal(testData.updateLimitMessage.CurrencyCode, limitRecord.CurrencyCode, "DB: Проверка параметра currency_code")
		sCtx.Assert().Equal(testData.updateLimitMessage.StartedAt, limitRecord.StartedAt, "DB: Проверка параметра started_at")
		sCtx.Assert().Equal(testData.updateLimitMessage.ExpiresAt, limitRecord.ExpiresAt, "DB: Проверка параметра expires_at")
		sCtx.Assert().True(limitRecord.LimitStatus, "DB: Проверка параметра limit_status")
	})
}
