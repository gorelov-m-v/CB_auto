package test

import (
	"fmt"
	"net/http"

	capModels "CB_auto/internal/client/cap/models"
	publicModels "CB_auto/internal/client/public/models"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/repository/wallet"
	"CB_auto/internal/transport/kafka"
	"CB_auto/internal/transport/nats"
	"CB_auto/internal/transport/redis"

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
		registrationResponse   *clientTypes.Response[publicModels.FastRegistrationResponseBody]
		authorizationResponse  *clientTypes.Response[publicModels.TokenCheckResponseBody]
		registrationMessage    kafka.PlayerMessage
		singleBetLimitResponse *clientTypes.Response[publicModels.GetSingleBetLimitsResponseBody]
		singleBetLimitReq      *clientTypes.Request[publicModels.SetSingleBetLimitRequestBody]
		limitMessage           kafka.LimitMessage
		dbWallet               *wallet.Wallet
		singleBetEvent         *nats.NatsMessage[nats.LimitChangedV2]
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

	t.WithNewStep("Получение сообщения сообщения о регистрации игрока из топика player.v1.account", func(sCtx provider.StepCtx) {
		testData.registrationMessage = kafka.FindMessageByFilter(sCtx, s.Shared.Kafka, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == kafka.PlayerEventSignUpFast &&
				msg.Player.AccountID == testData.registrationResponse.Body.Username
		})
		sCtx.Require().NotEmpty(testData.registrationMessage.Player.ID, "ID игрока не пустой")
	})

	t.WithNewStep("Получение кошелька игрока из базы данных", func(sCtx provider.StepCtx) {
		walletFilter := map[string]interface{}{
			"player_uuid": testData.registrationMessage.Player.ExternalID,
			"is_default":  true,
			"is_basic":    true,
			"wallet_type": wallet.WalletTypeReal,
			"currency":    testData.registrationMessage.Player.Currency,
		}

		testData.dbWallet = s.Shared.WalletRepo.GetWalletWithRetry(sCtx, walletFilter)
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

	t.WithNewStep("Установка лимита на одиночную ставку", func(sCtx provider.StepCtx) {
		testData.singleBetLimitReq = &clientTypes.Request[publicModels.SetSingleBetLimitRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
			},
			Body: &publicModels.SetSingleBetLimitRequestBody{
				Amount:   "100",
				Currency: s.Shared.Config.Node.DefaultCurrency,
			},
		}

		resp := s.Shared.PublicClient.SetSingleBetLimit(sCtx, testData.singleBetLimitReq)
		sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Лимит на одиночную ставку установлен")
	})

	t.WithNewStep("Получение сообщения о создании лимита на одиночную ставку из топика limits.v2", func(sCtx provider.StepCtx) {
		testData.limitMessage = kafka.FindMessageByFilter(sCtx, s.Shared.Kafka, func(msg kafka.LimitMessage) bool {
			return msg.EventType == kafka.LimitEventCreated &&
				msg.LimitType == kafka.LimitTypeSingleBet &&
				msg.PlayerID == testData.registrationMessage.Player.ExternalID &&
				msg.Amount == testData.singleBetLimitReq.Body.Amount &&
				msg.CurrencyCode == testData.singleBetLimitReq.Body.Currency
		})
		sCtx.Require().NotEmpty(testData.limitMessage.ID, "Сообщение о создании лимита на одиночную ставку из топика limits.v2 получено")
	})

	t.WithNewStep("Проверка ивента NATS о создании лимита на одиночную ставку", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.Shared.Config.Nats.StreamPrefix, testData.registrationMessage.Player.ExternalID)
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
				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
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
				"playerID": testData.registrationMessage.Player.ExternalID,
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
		err := s.Shared.RedisClient.GetWithSeqCheck(sCtx, testData.dbWallet.UUID, &redisValue, int(testData.singleBetEvent.Seq))
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
				msg.PlayerUUID == testData.registrationMessage.Player.ExternalID &&
				msg.WalletUUID == testData.dbWallet.UUID
		})
		sCtx.Assert().NotEmpty(projectionMessage.Type, "Kafka: Сообщение projectionSource найдено")

		sCtx.Assert().Equal(string(kafka.ProjectionEventLimitChanged), string(projectionMessage.Type), "Kafka: Проверка параметра type")
		sCtx.Assert().Equal(testData.singleBetEvent.Sequence, projectionMessage.SeqNumber, "Kafka: Проверка параметра seq_number")
		sCtx.Assert().Equal(testData.dbWallet.UUID, projectionMessage.WalletUUID, "Kafka: Проверка параметра wallet_uuid")
		sCtx.Assert().Equal(testData.registrationMessage.Player.ExternalID, projectionMessage.PlayerUUID, "Kafka: Проверка параметра player_uuid")
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
			"player_uuid":   testData.registrationMessage.Player.ExternalID,
		}

		limitRecord := s.Shared.LimitRecordRepo.GetLimitRecordWithRetry(sCtx, filters)
		sCtx.Assert().NotNil(limitRecord, "DB: Запись о лимите найдена в таблице limit_record_v2")
		sCtx.Assert().Equal(testData.limitMessage.ID, limitRecord.ExternalUUID, "DB: Проверка параметра external_uuid")
		sCtx.Assert().Equal(testData.registrationMessage.Player.ExternalID, limitRecord.PlayerUUID, "DB: Проверка параметра player_uuid")
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
