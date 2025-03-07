package test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	capAPI "CB_auto/internal/client/cap"
	capModels "CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/factory"
	publicAPI "CB_auto/internal/client/public"
	publicModels "CB_auto/internal/client/public/models"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/internal/repository/wallet"
	"CB_auto/internal/transport/kafka"
	"CB_auto/internal/transport/nats"
	"CB_auto/internal/transport/redis"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type TurnoverLimitSuite struct {
	suite.Suite
	config       *config.Config
	publicClient publicAPI.PublicAPI
	kafka        *kafka.Kafka
	natsClient   *nats.NatsClient
	capClient    capAPI.CapAPI
	database     *repository.Connector
	walletRepo   *wallet.Repository
	redisClient  *redis.RedisClient
}

func (s *TurnoverLimitSuite) BeforeAll(t provider.T) {
	t.Epic("Лимиты")

	t.WithNewStep("Чтение конфигурационного файла", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация Public API клиента", func(sCtx provider.StepCtx) {
		s.publicClient = factory.InitClient[publicAPI.PublicAPI](sCtx, s.config, clientTypes.Public)
	})

	t.WithNewStep("Соединение с базой данных wallet.", func(sCtx provider.StepCtx) {
		s.walletRepo = wallet.NewRepository(repository.OpenConnector(t, &s.config.MySQL, repository.Wallet).DB(), &s.config.MySQL)
	})

	t.WithNewStep("Инициализация Redis клиента", func(sCtx provider.StepCtx) {
		s.redisClient = redis.NewRedisClient(t, &s.config.Redis, redis.WalletClient)
	})

	t.WithNewStep("Инициализация Kafka клиента", func(sCtx provider.StepCtx) {
		s.kafka = kafka.GetInstance(t, s.config)
	})

	t.WithNewStep("Инициализация NATS клиента", func(sCtx provider.StepCtx) {
		s.natsClient = nats.NewClient(&s.config.Nats)
	})

	t.WithNewStep("Инициализация CAP API клиента", func(sCtx provider.StepCtx) {
		s.capClient = factory.InitClient[capAPI.CapAPI](sCtx, s.config, clientTypes.Cap)
	})
}

func (s *TurnoverLimitSuite) TestTurnoverLimit(t provider.T) {
	t.Feature("turnover-of-funds лимит")
	t.Title("Проверка создания лимита на оборот средств в Kafka, NATS, Redis, MySQL, Public API")

	var testData struct {
		registrationResponse  *clientTypes.Response[publicModels.FastRegistrationResponseBody]
		authorizationResponse *clientTypes.Response[publicModels.TokenCheckResponseBody]
		registrationMessage   kafka.PlayerMessage
		turnoverLimitResponse *clientTypes.Response[publicModels.GetTurnoverLimitsResponseBody] // изменить тип
		setTurnoverLimitReq   *clientTypes.Request[publicModels.SetTurnoverLimitRequestBody]
		limitMessage          kafka.LimitMessage
		dbWallet              *wallet.Wallet
	}

	t.WithNewStep("Регистрация нового игрока", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[publicModels.FastRegistrationRequestBody]{
			Body: &publicModels.FastRegistrationRequestBody{
				Country:  s.config.Node.DefaultCountry,
				Currency: s.config.Node.DefaultCurrency,
			},
		}

		testData.registrationResponse = s.publicClient.FastRegistration(sCtx, req)

		sCtx.Assert().Equal(http.StatusOK, testData.registrationResponse.StatusCode, "Успешная регистрация")
	})

	t.WithNewStep("Проверка Kafka-сообщения о регистрации игрока", func(sCtx provider.StepCtx) {
		testData.registrationMessage = kafka.FindMessageByFilter[kafka.PlayerMessage](sCtx, s.kafka, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == kafka.PlayerEventSignUpFast &&
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

		testData.dbWallet = s.walletRepo.GetWallet(sCtx, walletFilter)

		sCtx.Require().NotNil(testData.dbWallet, "Базовый кошелек найден в базе данных")
	})

	t.WithNewStep("Получение токена авторизации", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[publicModels.TokenCheckRequestBody]{
			Body: &publicModels.TokenCheckRequestBody{
				Username: testData.registrationResponse.Body.Username,
				Password: testData.registrationResponse.Body.Password,
			},
		}

		testData.authorizationResponse = s.publicClient.TokenCheck(sCtx, req)

		sCtx.Require().Equal(http.StatusOK, testData.authorizationResponse.StatusCode, "Успешная авторизация")
	})

	t.WithNewStep("Установка лимита на оборот средств", func(sCtx provider.StepCtx) {
		testData.setTurnoverLimitReq = &clientTypes.Request[publicModels.SetTurnoverLimitRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
			},
			Body: &publicModels.SetTurnoverLimitRequestBody{
				Amount:    "100",
				Currency:  s.config.Node.DefaultCurrency,
				Type:      publicModels.LimitPeriodDaily,
				StartedAt: time.Now().Unix(),
			},
		}

		resp := s.publicClient.SetTurnoverLimit(sCtx, testData.setTurnoverLimitReq)

		sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Лимит на оборот средств установлен")
	})

	t.WithNewStep("Получение сообщения из Kafka о создании лимита на оборот средств", func(sCtx provider.StepCtx) {
		testData.limitMessage = kafka.FindMessageByFilter[kafka.LimitMessage](sCtx, s.kafka, func(msg kafka.LimitMessage) bool {
			return msg.EventType == kafka.LimitEventCreated &&
				msg.LimitType == kafka.LimitTypeTurnoverFunds &&
				msg.PlayerID == testData.registrationMessage.Player.ExternalID &&
				msg.Amount == testData.setTurnoverLimitReq.Body.Amount &&
				msg.CurrencyCode == testData.setTurnoverLimitReq.Body.Currency
		})

		sCtx.Require().NotEmpty(testData.limitMessage.ID, "ID лимита не пустой")
	})

	t.WithNewAsyncStep("Получение лимита на оборот средств через Public API", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
			},
		}

		testData.turnoverLimitResponse = s.publicClient.GetTurnoverLimits(sCtx, req)

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

	t.WithNewAsyncStep("Проверка сообщения в NATS о создании лимита на оборот средств", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, testData.registrationMessage.Player.ExternalID)

		turnoverEvent := nats.FindMessageInStream(sCtx, s.natsClient, subject, func(msg nats.LimitChangedV2, msgType string) bool {
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

	t.WithNewStep("Получение списка лимитов через CAP API", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capClient.GetToken(sCtx)),
				"Platform-NodeId": s.config.Node.ProjectID,
				"Platform-Locale": capModels.DefaultLocale,
			},
			PathParams: map[string]string{
				"playerID": testData.registrationMessage.Player.ExternalID,
			},
		}

		resp := s.capClient.GetPlayerLimits(sCtx, req)

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

	t.WithNewAsyncStep("Проверка данных кошелька в Redis", func(sCtx provider.StepCtx) {
		var redisValue redis.WalletFullData
		err := s.redisClient.GetWithRetry(sCtx, testData.dbWallet.UUID, &redisValue)

		sCtx.Require().NoError(err, "Значение кошелька получено из Redis")
		sCtx.Require().NotEmpty(redisValue.Limits, "Должен быть хотя бы один лимит")

		limitData := redisValue.Limits[0]

		sCtx.Assert().Equal(testData.limitMessage.ID, limitData.ExternalID, "ID лимита совпадает")
		sCtx.Assert().Equal(redis.LimitTypeTurnoverFunds, limitData.LimitType, "Тип лимита совпадает")
		sCtx.Assert().Equal(redis.LimitPeriodDaily, limitData.IntervalType, "Интервал лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limitData.Amount, "Сумма лимита совпадает")
		sCtx.Assert().Equal("0", limitData.Spent, "Потраченная сумма равна 0")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limitData.Rest, "Остаток равен сумме лимита")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limitData.CurrencyCode, "Валюта лимита совпадает")
		sCtx.Assert().InDelta(testData.limitMessage.StartedAt, limitData.StartedAt, 10, "Время начала лимита")
		sCtx.Assert().InDelta(testData.limitMessage.ExpiresAt, limitData.ExpiresAt, 10, "Время окончания лимита")
		sCtx.Assert().True(limitData.Status, "Лимит активен")
	})

	t.WithNewStep("Проверка отправки события лимита в Kafka projection source", func(sCtx provider.StepCtx) {
		projectionMessage := kafka.FindMessageByFilter[kafka.ProjectionSourceMessage](sCtx, s.kafka, func(msg kafka.ProjectionSourceMessage) bool {
			return msg.Type == kafka.ProjectionEventLimitChanged &&
				msg.PlayerUUID == testData.registrationMessage.Player.ExternalID &&
				msg.WalletUUID == testData.dbWallet.UUID
		})

		sCtx.Require().NotEmpty(projectionMessage.Type, "Сообщение limit_changed_v2 найдено в топике projection source")

		var limitsPayload kafka.ProjectionPayloadLimits
		err := projectionMessage.UnmarshalPayloadTo(&limitsPayload)
		sCtx.Require().NoError(err, "Payload успешно распакован")

		sCtx.Require().GreaterOrEqual(len(limitsPayload.Limits), 1, "В payload есть хотя бы один лимит")

		limit := limitsPayload.Limits[0]
		sCtx.Assert().Equal(kafka.LimitEventCreated, limitsPayload.EventType, "Тип события в payload корректен")
		sCtx.Assert().Equal(testData.limitMessage.ID, limit.ExternalID, "ID лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.Amount, limit.Amount, "Сумма лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.CurrencyCode, limit.CurrencyCode, "Валюта лимита совпадает")
		sCtx.Assert().Equal(testData.limitMessage.LimitType, limit.LimitType, "Тип лимита совпадает")
		sCtx.Assert().True(limit.Status, "Лимит активен")
	})
}

func (s *TurnoverLimitSuite) AfterAll(t provider.T) {
	kafka.CloseInstance(t)
	if s.natsClient != nil {
		s.natsClient.Close()
	}
}

func TestTurnoverLimitSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(TurnoverLimitSuite))
}
