package test

import (
	"fmt"
	"net/http"
	"testing"

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
	"CB_auto/pkg/mappers"
	"CB_auto/pkg/utils"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type BalanceAdjustmentSuite struct {
	suite.Suite
	config       *config.Config
	publicClient publicAPI.PublicAPI
	capClient    capAPI.CapAPI
	kafka        *kafka.Kafka
	natsClient   *nats.NatsClient
	database     *repository.Connector
	walletRepo   *wallet.Repository
	redisClient  *redis.RedisClient
}

func (s *BalanceAdjustmentSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация http-клиентов", func(sCtx provider.StepCtx) {
		s.publicClient = factory.InitClient[publicAPI.PublicAPI](sCtx, s.config, clientTypes.Public)
		s.capClient = factory.InitClient[capAPI.CapAPI](sCtx, s.config, clientTypes.Cap)
	})

	t.WithNewStep("Инициализация Kafka", func(sCtx provider.StepCtx) {
		s.kafka = kafka.GetInstance(t, s.config)
	})

	t.WithNewStep("Инициализация Redis клиента", func(sCtx provider.StepCtx) {
		s.redisClient = redis.NewRedisClient(t, &s.config.Redis, redis.WalletClient)
	})

	t.WithNewStep("Инициализация NATS клиента", func(sCtx provider.StepCtx) {
		s.natsClient = nats.NewClient(&s.config.Nats)
	})

	t.WithNewStep("Соединение с базой данных", func(sCtx provider.StepCtx) {
		s.walletRepo = wallet.NewRepository(repository.OpenConnector(t, &s.config.MySQL, repository.Wallet).DB(), &s.config.MySQL)
	})
}

func (s *BalanceAdjustmentSuite) TestBalanceAdjustment(t provider.T) {
	t.Epic("Wallet")
	t.Feature("Корректировка баланса")
	t.Title("Проверка корректировки баланса кошелька")
	t.Tags("Wallet", "BalanceAdjustment")

	var testData struct {
		registrationResponse  *clientTypes.Response[publicModels.FastRegistrationResponseBody]
		registrationMessage   kafka.PlayerMessage
		walletCreatedEvent    *nats.NatsMessage[nats.WalletCreatedPayload]
		adjustmentRequest     *clientTypes.Request[capModels.CreateBalanceAdjustmentRequestBody]
		adjustmentResponse    *clientTypes.Response[struct{}]
		balanceAdjustedEvent  *nats.NatsMessage[nats.BalanceAdjustedPayload]
		projectionAdjustEvent kafka.ProjectionSourceMessage
	}

	t.WithNewStep("Регистрация пользователя.", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[publicModels.FastRegistrationRequestBody]{
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: &publicModels.FastRegistrationRequestBody{
				Country:  s.config.Node.DefaultCountry,
				Currency: s.config.Node.DefaultCurrency,
			},
		}
		testData.registrationResponse = s.publicClient.FastRegistration(sCtx, req)

		sCtx.Require().NotEmpty(testData.registrationResponse.Body.Username, "Username в ответе регистрации не пустой")
		sCtx.Require().NotEmpty(testData.registrationResponse.Body.Password, "Password в ответе регистрации не пустой")
	})

	t.WithNewStep("Получение сообщения о регистрации из топика player.v1.account.", func(sCtx provider.StepCtx) {
		testData.registrationMessage = kafka.FindMessageByFilter(sCtx, s.kafka, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == string(kafka.PlayerEventSignUpFast) &&
				msg.Player.AccountID == testData.registrationResponse.Body.Username
		})

		sCtx.Require().NotEmpty(testData.registrationMessage.Player.ExternalID, "External ID игрока в сообщении регистрации не пустой")
	})

	t.WithNewStep("Проверка создания кошелька в NATS.", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, testData.registrationMessage.Player.ExternalID)

		testData.walletCreatedEvent = nats.FindMessageInStream(sCtx, s.natsClient, subject, func(payload nats.WalletCreatedPayload, msgType string) bool {
			return msgType == string(nats.WalletCreated) &&
				payload.WalletType == nats.TypeReal &&
				payload.WalletStatus == nats.StatusEnabled &&
				payload.IsBasic
		})

		sCtx.Require().NotEmpty(testData.walletCreatedEvent.Payload.WalletUUID, "UUID кошелька в ивенте wallet_created не пустой")
	})

	t.WithNewStep("Выполнение корректировки баланса в положительную сторону", func(sCtx provider.StepCtx) {
		testData.adjustmentRequest = &clientTypes.Request[capModels.CreateBalanceAdjustmentRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capClient.GetToken(sCtx)),
				"Platform-Locale": capModels.DefaultLocale,
				"Platform-NodeID": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"player_uuid": testData.registrationMessage.Player.ExternalID,
			},
			Body: &capModels.CreateBalanceAdjustmentRequestBody{
				Currency:      s.config.Node.DefaultCurrency,
				Amount:        100.0,
				Reason:        capModels.ReasonOperationalMistake,
				OperationType: capModels.OperationTypeDeposit,
				Direction:     capModels.DirectionIncrease,
				Comment:       utils.Get(utils.LETTERS, 25),
			},
		}

		testData.adjustmentResponse = s.capClient.CreateBalanceAdjustment(sCtx, testData.adjustmentRequest)
		sCtx.Require().Equal(http.StatusOK, testData.adjustmentResponse.StatusCode, "Статус код ответа равен 200")
	})

	t.WithNewStep("Проверка события корректировки баланса в NATS", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.%s", s.config.Nats.StreamPrefix,
			testData.registrationMessage.Player.ExternalID,
			testData.walletCreatedEvent.Payload.WalletUUID)

		testData.balanceAdjustedEvent = nats.FindMessageInStream(
			sCtx, s.natsClient, subject,
			func(payload nats.BalanceAdjustedPayload, msgType string) bool {
				return msgType == string(nats.BalanceAdjusted)
			})

		sCtx.Assert().NotNil(testData.balanceAdjustedEvent, "Событие balance_adjusted получено")

		expectedAmount := testData.adjustmentRequest.Body.Amount
		actualAmount := mappers.StringToAmount(testData.balanceAdjustedEvent.Payload.Amount)
		sCtx.Assert().Equal(expectedAmount, actualAmount, "Сумма корректировки совпадает")

		sCtx.Assert().Equal(
			mappers.MapDirectionToNats(testData.adjustmentRequest.Body.Direction),
			testData.balanceAdjustedEvent.Payload.Direction,
			"Направление корректировки совпадает")

		sCtx.Assert().Equal(
			mappers.MapOperationTypeToNats(testData.adjustmentRequest.Body.OperationType),
			testData.balanceAdjustedEvent.Payload.OperationType,
			"Тип операции совпадает")

		sCtx.Assert().Equal(
			mappers.MapReasonToNats(testData.adjustmentRequest.Body.Reason),
			testData.balanceAdjustedEvent.Payload.Reason,
			"Причина корректировки совпадает")

		sCtx.Assert().Equal(
			testData.adjustmentRequest.Body.Comment,
			testData.balanceAdjustedEvent.Payload.Comment,
			"Комментарий совпадает")

		sCtx.Assert().Equal(
			testData.adjustmentRequest.Body.Currency,
			testData.balanceAdjustedEvent.Payload.Currenc,
			"Валюта совпадает")

		sCtx.Assert().NotEmpty(
			testData.balanceAdjustedEvent.Payload.UserUUID,
			"UUID пользователя не пустой")

		sCtx.Assert().Equal(
			s.config.HTTP.CapUsername,
			testData.balanceAdjustedEvent.Payload.UserName,
			"Имя пользователя - admin")
	})

	t.WithNewAsyncStep("Проверка отправки события корректировки баланса в Kafka projection source", func(sCtx provider.StepCtx) {
		testData.projectionAdjustEvent = kafka.FindMessageByFilter(sCtx, s.kafka, func(msg kafka.ProjectionSourceMessage) bool {
			return msg.Type == string(kafka.ProjectionEventBalanceAdjusted) &&
				msg.PlayerUUID == testData.registrationMessage.Player.ExternalID &&
				msg.WalletUUID == testData.walletCreatedEvent.Payload.WalletUUID
		})

		sCtx.Require().NotEmpty(testData.projectionAdjustEvent.Type, "Сообщение balance_adjusted найдено в топике projection source")

		sCtx.Assert().Equal(testData.registrationMessage.Player.ExternalID, testData.projectionAdjustEvent.PlayerUUID, "UUID игрока совпадает")
		sCtx.Assert().Equal(testData.walletCreatedEvent.Payload.WalletUUID, testData.projectionAdjustEvent.WalletUUID, "UUID кошелька совпадает")
		sCtx.Assert().Equal(s.config.Node.DefaultCurrency, testData.projectionAdjustEvent.Currency, "Валюта совпадает")

		var adjustmentPayload kafka.ProjectionPayloadAdjustment
		err := testData.projectionAdjustEvent.UnmarshalPayloadTo(&adjustmentPayload)
		sCtx.Require().NoError(err, "Payload успешно распарсен")

		expectedAmount := testData.adjustmentRequest.Body.Amount
		actualAmount := mappers.StringToAmount(adjustmentPayload.Amount)
		sCtx.Assert().Equal(expectedAmount, actualAmount, "Сумма корректировки равна запрошенной")

		sCtx.Assert().Equal(
			mappers.MapDirectionToNats(testData.adjustmentRequest.Body.Direction),
			adjustmentPayload.Direction,
			"Направление корректировки совпадает")

		sCtx.Assert().Equal(
			mappers.MapOperationTypeToNats(testData.adjustmentRequest.Body.OperationType),
			adjustmentPayload.OperationType,
			"Тип операции совпадает")

		sCtx.Assert().Equal(
			mappers.MapReasonToNats(testData.adjustmentRequest.Body.Reason),
			adjustmentPayload.Reason,
			"Причина корректировки совпадает")

		sCtx.Assert().Equal(testData.adjustmentRequest.Body.Comment, adjustmentPayload.Comment, "Комментарий верный")
		sCtx.Assert().Equal(s.config.Node.DefaultCurrency, adjustmentPayload.Currenc, "Валюта верная")
		sCtx.Assert().Equal(s.config.HTTP.CapUsername, adjustmentPayload.UserName, "Имя пользователя - admin")
		sCtx.Assert().NotEmpty(adjustmentPayload.UserUUID, "UUID пользователя не пустой")
	})

	t.WithNewStep("Проверка данных кошелька в Redis", func(sCtx provider.StepCtx) {
		var redisValue redis.WalletFullData
		err := s.redisClient.GetWithRetry(sCtx, testData.walletCreatedEvent.Payload.WalletUUID, &redisValue)
		sCtx.Require().NoError(err, "Значение кошелька получено из Redis")

		expectedBalance := fmt.Sprintf("%.0f", testData.adjustmentRequest.Body.Amount)
		sCtx.Assert().Equal(expectedBalance, redisValue.Balance, "Баланс кошелька соответствует сумме корректировки")
		sCtx.Assert().Equal(int(testData.balanceAdjustedEvent.Sequence), redisValue.LastSeqNumber, "Номер последовательности совпадает")
	})
}

func (s *BalanceAdjustmentSuite) AfterAll(t provider.T) {
	if s.natsClient != nil {
		s.natsClient.Close()
	}

	if s.database != nil {
		if err := s.database.Close(); err != nil {
			t.Errorf("Ошибка при закрытии соединения с DB: %v", err)
		}
	}

	kafka.CloseInstance(t)
}

func TestBalanceAdjustmentSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(BalanceAdjustmentSuite))
}
