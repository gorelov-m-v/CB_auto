package test

// import (
// 	"fmt"
// 	"net/http"

// 	capAPI "CB_auto/internal/client/cap"
// 	capModels "CB_auto/internal/client/cap/models"
// 	publicAPI "CB_auto/internal/client/public"
// 	publicModels "CB_auto/internal/client/public/models"
// 	clientTypes "CB_auto/internal/client/types"
// 	"CB_auto/internal/config"
// 	"CB_auto/internal/repository/wallet"
// 	"CB_auto/internal/transport/kafka"
// 	"CB_auto/internal/transport/nats"
// 	"CB_auto/internal/transport/redis"

// 	_ "github.com/go-sql-driver/mysql"
// 	"github.com/ozontech/allure-go/pkg/framework/provider"
// 	"github.com/ozontech/allure-go/pkg/framework/suite"
// )

// type SingleBetLimitSuite struct {
// 	suite.Suite
// 	config       *config.Config
// 	publicClient publicAPI.PublicAPI
// 	capClient    capAPI.CapAPI
// 	kafka        *kafka.Kafka
// 	natsClient   *nats.NatsClient
// 	walletRepo   *wallet.Repository
// 	redisClient  *redis.RedisClient
// }

// func (s *SingleBetLimitSuite) TestSingleBetLimit(t provider.T) {
// 	t.Epic("Лимиты")
// 	t.Feature("single-bet лимит")
// 	t.Title("Проверка создания single-bet лимита в Kafka, NATS, Redis, MySQL, Public API, CAP API")
// 	t.Tags("wallet", "limits")

// 	var testData struct {
// 		registrationResponse   *clientTypes.Response[publicModels.FastRegistrationResponseBody]
// 		authorizationResponse  *clientTypes.Response[publicModels.TokenCheckResponseBody]
// 		registrationMessage    kafka.PlayerMessage
// 		singleBetLimitResponse *clientTypes.Response[publicModels.GetSingleBetLimitsResponseBody]
// 		setSingleBetLimitReq   *clientTypes.Request[publicModels.SetSingleBetLimitRequestBody]
// 		limitMessage           kafka.LimitMessage
// 		dbWallet               *wallet.Wallet
// 		singleBetEvent         *nats.NatsMessage[nats.LimitChangedV2]
// 	}

// 	t.WithNewStep("Регистрация нового игрока", func(sCtx provider.StepCtx) {
// 		req := &clientTypes.Request[publicModels.FastRegistrationRequestBody]{
// 			Body: &publicModels.FastRegistrationRequestBody{
// 				Country:  s.config.Node.DefaultCountry,
// 				Currency: s.config.Node.DefaultCurrency,
// 			},
// 		}
// 		testData.registrationResponse = s.publicClient.FastRegistration(sCtx, req)
// 		sCtx.Assert().Equal(http.StatusOK, testData.registrationResponse.StatusCode, "Успешная регистрация")
// 	})

// 	t.WithNewStep("Проверка Kafka-сообщения о регистрации игрока", func(sCtx provider.StepCtx) {
// 		testData.registrationMessage = kafka.FindMessageByFilter(sCtx, s.kafka, func(msg kafka.PlayerMessage) bool {
// 			return msg.Message.EventType == string(kafka.PlayerEventSignUpFast) &&
// 				msg.Player.AccountID == testData.registrationResponse.Body.Username
// 		})
// 		sCtx.Require().NotEmpty(testData.registrationMessage.Player.ID, "ID игрока не пустой")
// 	})

// 	t.WithNewStep("Проверка базового кошелька игрока в базе данных", func(sCtx provider.StepCtx) {
// 		walletFilter := map[string]interface{}{
// 			"player_uuid": testData.registrationMessage.Player.ExternalID,
// 			"is_default":  true,
// 			"is_basic":    true,
// 			"wallet_type": wallet.WalletTypeReal,
// 			"currency":    testData.registrationMessage.Player.Currency,
// 		}
// 		testData.dbWallet = s.walletRepo.GetOneWithRetry(sCtx, walletFilter)
// 		sCtx.Require().NotNil(testData.dbWallet, "Базовый кошелек найден в базе данных")
// 	})

// 	t.WithNewStep("Получение токена авторизации", func(sCtx provider.StepCtx) {
// 		req := &clientTypes.Request[publicModels.TokenCheckRequestBody]{
// 			Body: &publicModels.TokenCheckRequestBody{
// 				Username: testData.registrationResponse.Body.Username,
// 				Password: testData.registrationResponse.Body.Password,
// 			},
// 		}
// 		testData.authorizationResponse = s.publicClient.TokenCheck(sCtx, req)
// 		sCtx.Require().Equal(http.StatusOK, testData.authorizationResponse.StatusCode, "Успешная авторизация")
// 	})

// 	t.WithNewStep("Установка лимита на одиночную ставку", func(sCtx provider.StepCtx) {
// 		testData.setSingleBetLimitReq = &clientTypes.Request[publicModels.SetSingleBetLimitRequestBody]{
// 			Headers: map[string]string{
// 				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
// 			},
// 			Body: &publicModels.SetSingleBetLimitRequestBody{
// 				Amount:   "100",
// 				Currency: s.config.Node.DefaultCurrency,
// 			},
// 		}
// 		resp := s.publicClient.SetSingleBetLimit(sCtx, testData.setSingleBetLimitReq)
// 		sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Лимит на ставку установлен")
// 	})

// 	t.WithNewStep("Проверка Kafka-сообщения о создании лимита", func(sCtx provider.StepCtx) {
// 		testData.limitMessage = kafka.FindMessageByFilter(sCtx, s.kafka, func(msg kafka.LimitMessage) bool {
// 			return msg.EventType == string(kafka.LimitEventCreated) &&
// 				msg.LimitType == string(kafka.LimitTypeSingleBet) &&
// 				msg.PlayerID == testData.registrationMessage.Player.ExternalID &&
// 				msg.Amount == testData.setSingleBetLimitReq.Body.Amount &&
// 				msg.CurrencyCode == s.config.Node.DefaultCurrency
// 		})
// 		sCtx.Require().NotEmpty(testData.limitMessage.ID, "ID лимита не пустой")
// 	})

// 	t.WithNewAsyncStep("Получение лимита через Public API", func(sCtx provider.StepCtx) {
// 		req := &clientTypes.Request[any]{
// 			Headers: map[string]string{
// 				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
// 			},
// 		}
// 		testData.singleBetLimitResponse = s.publicClient.GetSingleBetLimits(sCtx, req)
// 		sCtx.Assert().Equal(http.StatusOK, testData.singleBetLimitResponse.StatusCode, "Лимиты получены")
// 		sCtx.Assert().NotEmpty(testData.singleBetLimitResponse.Body, "Список лимитов не пустой")
// 		limit := testData.singleBetLimitResponse.Body[0]
// 		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ID, "ID лимита совпадает")
// 		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "Сумма лимита совпадает")
// 		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.Currency, "Валюта лимита совпадает")
// 		sCtx.Assert().True(limit.Status, "Лимит активен")
// 		sCtx.Assert().True(limit.Required, "Лимит обязательный")
// 		sCtx.Assert().Empty(limit.UpcomingChanges, "Нет предстоящих изменений")
// 		sCtx.Assert().Nil(limit.DeactivatedAt, "Лимит не деактивирован")
// 	})

// 	t.WithNewStep("Проверка NATS-сообщения о создании лимита", func(sCtx provider.StepCtx) {
// 		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, testData.registrationMessage.Player.ExternalID)
// 		testData.singleBetEvent = nats.FindMessageInStream(sCtx, s.natsClient, subject, func(msg nats.LimitChangedV2, msgType string) bool {
// 			return msg.EventType == nats.EventTypeCreated &&
// 				len(msg.Limits) > 0 &&
// 				msg.Limits[0].LimitType == nats.LimitTypeSingleBet
// 		})
// 		sCtx.Require().NotNil(testData.singleBetEvent, "Получено NATS-сообщение о создании лимита")
// 		limit := testData.singleBetEvent.Payload.Limits[0]
// 		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ExternalID, "ID лимита совпадает")
// 		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "Сумма лимита совпадает")
// 		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.CurrencyCode, "Валюта лимита совпадает")
// 		sCtx.Assert().Equal(testData.limitMessage.LimitType, limit.LimitType, "Тип лимита совпадает")
// 		sCtx.Assert().True(limit.Status, "Лимит активен")
// 		sCtx.Assert().Equal(testData.limitMessage.StartedAt, limit.StartedAt, "Время начала установлено")
// 		sCtx.Assert().Zero(limit.ExpiresAt, "Время окончания не установлено")
// 		sCtx.Assert().Equal(nats.LimitTypeSingleBet, limit.LimitType, "Тип лимита совпадает")
// 		sCtx.Assert().Equal(testData.singleBetEvent.Payload.EventType, testData.limitMessage.EventType, "Тип события совпадает")
// 	})

// 	t.WithNewStep("Получение списка лимитов через CAP API", func(sCtx provider.StepCtx) {
// 		req := &clientTypes.Request[any]{
// 			Headers: map[string]string{
// 				"Authorization":   fmt.Sprintf("Bearer %s", s.capClient.GetToken(sCtx)),
// 				"Platform-NodeId": s.config.Node.ProjectID,
// 				"Platform-Locale": capModels.DefaultLocale,
// 			},
// 			PathParams: map[string]string{
// 				"playerID": testData.registrationMessage.Player.ExternalID,
// 			},
// 		}
// 		resp := s.capClient.GetPlayerLimits(sCtx, req)
// 		sCtx.Assert().Equal(http.StatusOK, resp.StatusCode, "Список лимитов получен")
// 		sCtx.Assert().Equal(1, resp.Body.Total, "Количество лимитов корректно")
// 		singleBetLimit := resp.Body.Data[0]
// 		sCtx.Assert().Equal(capModels.LimitTypeSingleBet, singleBetLimit.Type)
// 		sCtx.Assert().True(singleBetLimit.Status)
// 		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, singleBetLimit.Currency)
// 		sCtx.Assert().Equal(testData.limitMessage.Amount, singleBetLimit.Rest)
// 		sCtx.Assert().Equal(testData.limitMessage.Amount, singleBetLimit.Amount)
// 		sCtx.Assert().InDelta(testData.limitMessage.StartedAt, singleBetLimit.StartedAt, 10)
// 		sCtx.Assert().InDelta(testData.limitMessage.ExpiresAt, singleBetLimit.ExpiresAt, 10)
// 		sCtx.Assert().Zero(singleBetLimit.DeactivatedAt)
// 	})

// 	t.WithNewStep("Проверка данных кошелька в Redis", func(sCtx provider.StepCtx) {
// 		var redisValue redis.WalletFullData
// 		err := s.redisClient.GetWithRetry(sCtx, testData.dbWallet.UUID, &redisValue)
// 		sCtx.Require().NoError(err, "Значение кошелька получено из Redis")
// 		sCtx.Require().NotEmpty(redisValue.Limits, "Должен быть хотя бы один лимит")
// 		limitData := redisValue.Limits[0]
// 		sCtx.Assert().Equal(testData.limitMessage.ID, limitData.ExternalID, "ID лимита совпадает")
// 		sCtx.Assert().Equal(redis.LimitTypeSingleBet, limitData.LimitType, "Тип лимита совпадает")
// 		sCtx.Assert().Equal(testData.limitMessage.Amount, limitData.Amount, "Сумма лимита совпадает")
// 		sCtx.Assert().Equal("0", limitData.Spent, "Потраченная сумма равна 0")
// 		sCtx.Assert().Equal(testData.limitMessage.Amount, limitData.Rest, "Остаток равен сумме лимита")
// 		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limitData.CurrencyCode, "Валюта лимита совпадает")
// 		sCtx.Assert().InDelta(testData.limitMessage.StartedAt, limitData.StartedAt, 10, "Время начала лимита")
// 		sCtx.Assert().InDelta(testData.limitMessage.ExpiresAt, limitData.ExpiresAt, 10, "Время окончания лимита")
// 		sCtx.Assert().True(limitData.Status, "Лимит активен")
// 	})

// 	t.WithNewStep("Проверка отправки события лимита в Kafka projection source", func(sCtx provider.StepCtx) {
// 		projectionMessage := kafka.FindMessageByFilter(sCtx, s.kafka, func(msg kafka.ProjectionSourceMessage) bool {
// 			return msg.Type == string(kafka.ProjectionEventLimitChanged) &&
// 				msg.PlayerUUID == testData.registrationMessage.Player.ExternalID &&
// 				msg.WalletUUID == testData.dbWallet.UUID
// 		})
// 		sCtx.Require().NotEmpty(projectionMessage.Type, "Сообщение limit_changed_v2 найдено в топике projection source")
// 		var limitsPayload kafka.ProjectionPayloadLimits
// 		err := projectionMessage.UnmarshalPayloadTo(&limitsPayload)
// 		sCtx.Require().NoError(err, "Payload успешно распакован")
// 		sCtx.Require().GreaterOrEqual(len(limitsPayload.Limits), 1, "В payload есть хотя бы один лимит")
// 		limit := limitsPayload.Limits[0]
// 		sCtx.Assert().Equal(string(kafka.LimitEventCreated), limitsPayload.EventType, "Тип события в payload корректен")
// 		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ExternalID, "ID лимита совпадает")
// 		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "Сумма лимита совпадает")
// 		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.CurrencyCode, "Валюта лимита совпадает")
// 		sCtx.Assert().Equal(testData.limitMessage.LimitType, limit.LimitType, "Тип лимита совпадает")
// 		sCtx.Assert().True(limit.Status, "Лимит активен")
// 	})
// }
