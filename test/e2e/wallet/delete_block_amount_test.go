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
	"CB_auto/pkg/utils"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type DeleteBlockAmountSuite struct {
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

func (s *DeleteBlockAmountSuite) BeforeAll(t provider.T) {
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

func (s *DeleteBlockAmountSuite) TestBlockAmount(t provider.T) {
	t.Epic("Wallet")
	t.Feature("Блокировка средств")
	t.Title("Проверка блокировки средств кошелька")
	t.Tags("Wallet", "BlockAmount")

	var testData struct {
		registrationResponse *clientTypes.Response[publicModels.FastRegistrationResponseBody]
		registrationMessage  kafka.PlayerMessage
		walletCreatedEvent   *nats.NatsMessage[nats.WalletCreatedPayload]
		adjustmentRequest    *clientTypes.Request[capModels.CreateBalanceAdjustmentRequestBody]
		adjustmentResponse   *clientTypes.Response[struct{}]
		balanceAdjustedEvent *nats.NatsMessage[nats.BalanceAdjustedPayload]
		blockRequest         *clientTypes.Request[capModels.CreateBlockAmountRequestBody]
		blockResponse        *clientTypes.Response[capModels.CreateBlockAmountResponseBody]
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
			return msg.Message.EventType == kafka.PlayerEventSignUpFast &&
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
				"Platform-Locale": "ru",
				"Platform-NodeID": s.config.Node.ProjectID,
				"Content-Type":    "application/json",
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

	t.WithNewStep("Создание блокировки средств", func(sCtx provider.StepCtx) {
		reason := utils.Get(utils.LETTERS, 25)
		testData.blockRequest = &clientTypes.Request[capModels.CreateBlockAmountRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capClient.GetToken(sCtx)),
				"Platform-Locale": "ru",
				"Platform-NodeID": s.config.Node.ProjectID,
				"Content-Type":    "application/json",
			},
			PathParams: map[string]string{
				"player_uuid": testData.walletCreatedEvent.Payload.PlayerUUID,
			},
			Body: &capModels.CreateBlockAmountRequestBody{
				Currency: s.config.Node.DefaultCurrency,
				Amount:   "50",
				Reason:   reason,
			},
		}

		testData.blockResponse = s.capClient.CreateBlockAmount(sCtx, testData.blockRequest)
		sCtx.Require().Equal(http.StatusOK, testData.blockResponse.StatusCode, "Статус код ответа равен 200")
		sCtx.Assert().Equal("50", testData.blockResponse.Body.Amount, "Сумма блокировки верна")
		sCtx.Assert().Equal(s.config.Node.DefaultCurrency, testData.blockResponse.Body.Currency, "Валюта блокировки верна")
		sCtx.Assert().Equal(reason, testData.blockResponse.Body.Reason, "Причина блокировки верна")
		sCtx.Assert().NotEmpty(testData.blockResponse.Body.TransactionID, "ID транзакции не пустой")
	})

	t.WithNewStep("Удаление блокировки средств", func(sCtx provider.StepCtx) {
		deleteBlockRequest := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capClient.GetToken(sCtx)),
				"Platform-Locale": "ru",
				"Platform-NodeID": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"block_uuid": testData.blockResponse.Body.TransactionID,
			},
			QueryParams: map[string]string{
				"walletId": testData.walletCreatedEvent.Payload.WalletUUID,
				"playerId": testData.registrationMessage.Player.ExternalID,
			},
		}

		deleteResponse := s.capClient.DeleteBlockAmount(sCtx, deleteBlockRequest)
		sCtx.Require().Equal(http.StatusNoContent, deleteResponse.StatusCode, "Статус код ответа равен 204")
	})

	t.WithNewStep("Проверка события отмены блокировки средств в NATS", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.%s", s.config.Nats.StreamPrefix,
			testData.registrationMessage.Player.ExternalID,
			testData.walletCreatedEvent.Payload.WalletUUID)

		revokeEvent := nats.FindMessageInStream[nats.BlockAmountRevokedPayload](sCtx, s.natsClient, subject, func(payload nats.BlockAmountRevokedPayload, msgType string) bool {
			return msgType == string(nats.BlockAmountRevoked) &&
				payload.UUID == testData.blockResponse.Body.TransactionID
		})

		sCtx.Require().NotNil(revokeEvent, "Событие block_amount_revoked получено")
		sCtx.Assert().Equal(s.config.HTTP.CapUsername, revokeEvent.Payload.UserName, "Имя пользователя верно")
		// sCtx.Assert().Equal(s.config.HTTP.CapUserUUID, revokeEvent.Payload.UserUUID, "UUID пользователя верный")
		sCtx.Assert().Equal(s.config.Node.ProjectID, revokeEvent.Payload.NodeUUID, "UUID ноды верный")
	})

	t.WithNewAsyncStep("Проверка отправки события отмены блокировки в Kafka projection source", func(sCtx provider.StepCtx) {
		projectionRevokeEvent := kafka.FindMessageByFilter[kafka.ProjectionSourceMessage](sCtx, s.kafka, func(msg kafka.ProjectionSourceMessage) bool {
			return msg.Type == kafka.ProjectionEventBlockAmountRevoked &&
				msg.PlayerUUID == testData.registrationMessage.Player.ExternalID &&
				msg.WalletUUID == testData.walletCreatedEvent.Payload.WalletUUID
		})

		sCtx.Require().NotEmpty(projectionRevokeEvent.Type, "Сообщение block_amount_revoked найдено в топике projection source")

		sCtx.Assert().Equal(testData.registrationMessage.Player.ExternalID, projectionRevokeEvent.PlayerUUID, "UUID игрока совпадает")
		sCtx.Assert().Equal(testData.walletCreatedEvent.Payload.WalletUUID, projectionRevokeEvent.WalletUUID, "UUID кошелька совпадает")
		sCtx.Assert().Equal(s.config.Node.DefaultCurrency, projectionRevokeEvent.Currency, "Валюта совпадает")
		sCtx.Assert().Equal(s.config.Node.ProjectID, projectionRevokeEvent.NodeUUID, "UUID ноды совпадает")

		var revokePayload kafka.ProjectionPayloadBlockAmountRevoked
		err := projectionRevokeEvent.UnmarshalPayloadTo(&revokePayload)
		sCtx.Require().NoError(err, "Payload успешно распарсен")

		sCtx.Assert().Equal(testData.blockResponse.Body.TransactionID, revokePayload.UUID, "UUID блокировки совпадает")
		sCtx.Assert().Equal(s.config.Node.ProjectID, revokePayload.NodeUUID, "UUID ноды совпадает")
		sCtx.Assert().Equal(testData.blockRequest.Body.Amount, revokePayload.Amount, "Сумма разблокировки верна")
	})

}

func (s *DeleteBlockAmountSuite) AfterAll(t provider.T) {
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

func TestDeleteBlockAmountSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(DeleteBlockAmountSuite))
}
