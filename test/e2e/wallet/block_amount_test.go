package test

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	capAPI "CB_auto/internal/client/cap"
	"CB_auto/internal/client/cap/models"
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

type BlockAmountSuite struct {
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

func (s *BlockAmountSuite) BeforeAll(t provider.T) {
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

func (s *BlockAmountSuite) TestBlockAmount(t provider.T) {
	t.Epic("Wallet")
	t.Feature("Блокировка средств")
	t.Title("Проверка блокировки средств кошелька")
	t.Tags("Wallet", "BlockAmount")

	var testData struct {
		registrationResponse *clientTypes.Response[publicModels.FastRegistrationResponseBody]
		registrationMessage  kafka.PlayerMessage
		walletCreatedEvent   *nats.NatsMessage[nats.WalletCreatedPayload]
		adjustmentRequest    *clientTypes.Request[capModels.CreateBalanceAdjustmentRequestBody]
		adjustmentResponse   *clientTypes.Response[capModels.CreateBalanceAdjustmentResponseBody]
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
				"Platform-Locale": models.DefaultLocale,
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

	t.WithNewStep("Создание блокировки средств", func(sCtx provider.StepCtx) {
		reason := utils.Get(utils.LETTERS, 25)
		testData.blockRequest = &clientTypes.Request[capModels.CreateBlockAmountRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capClient.GetToken(sCtx)),
				"Platform-Locale": models.DefaultLocale,
				"Platform-NodeID": s.config.Node.ProjectID,
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

	t.WithNewStep("Проверка события блокировки средств в NATS", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.%s", s.config.Nats.StreamPrefix,
			testData.registrationMessage.Player.ExternalID,
			testData.walletCreatedEvent.Payload.WalletUUID)

		blockEvent := nats.FindMessageInStream[nats.BlockAmountStartedPayload](sCtx, s.natsClient, subject, func(payload nats.BlockAmountStartedPayload, msgType string) bool {
			return msgType == string(nats.BlockAmountStarted) &&
				payload.Reason == testData.blockRequest.Body.Reason
		})

		sCtx.Require().NotNil(blockEvent, "Событие block_amount_started получено")
		sCtx.Assert().Equal("-"+testData.blockRequest.Body.Amount, blockEvent.Payload.Amount, "Сумма блокировки верна")
		sCtx.Assert().Equal(s.config.HTTP.CapUsername, blockEvent.Payload.UserName, "Имя пользователя верно")
		sCtx.Assert().NotEmpty(blockEvent.Payload.UUID, "UUID блокировки не пустой")
	})

	t.WithNewAsyncStep("Проверка отправки события блокировки в Kafka projection source", func(sCtx provider.StepCtx) {
		projectionBlockEvent := kafka.FindMessageByFilter[kafka.ProjectionSourceMessage](sCtx, s.kafka, func(msg kafka.ProjectionSourceMessage) bool {
			return msg.Type == kafka.ProjectionEventBlockAmountStarted &&
				msg.PlayerUUID == testData.registrationMessage.Player.ExternalID &&
				msg.WalletUUID == testData.walletCreatedEvent.Payload.WalletUUID
		})

		sCtx.Require().NotEmpty(projectionBlockEvent.Type, "Сообщение block_amount_started найдено в топике projection source")

		sCtx.Assert().Equal(testData.registrationMessage.Player.ExternalID, projectionBlockEvent.PlayerUUID, "UUID игрока совпадает")
		sCtx.Assert().Equal(testData.walletCreatedEvent.Payload.WalletUUID, projectionBlockEvent.WalletUUID, "UUID кошелька совпадает")
		sCtx.Assert().Equal(s.config.Node.DefaultCurrency, projectionBlockEvent.Currency, "Валюта совпадает")

		var blockPayload kafka.ProjectionPayloadBlockAmount
		err := projectionBlockEvent.UnmarshalPayloadTo(&blockPayload)
		sCtx.Require().NoError(err, "Payload успешно распарсен")

		sCtx.Assert().Equal("-"+testData.blockRequest.Body.Amount, blockPayload.Amount, "Сумма блокировки верна")
		sCtx.Assert().Equal(testData.blockRequest.Body.Reason, blockPayload.Reason, "Причина блокировки верна")
		sCtx.Assert().Equal(2, blockPayload.Status, "Статус блокировки верный")
		sCtx.Assert().Equal(3, blockPayload.Type, "Тип блокировки верный")
		sCtx.Assert().Equal(s.config.HTTP.CapUsername, blockPayload.UserName, "Имя пользователя верно")
		sCtx.Assert().NotEmpty(blockPayload.UUID, "UUID блокировки не пустой")
		sCtx.Assert().NotEmpty(blockPayload.UserUUID, "UUID пользователя не пустой")
		sCtx.Assert().NotZero(blockPayload.CreatedAt, "Время создания не пустое")
	})

	t.WithNewStep("Проверка данных кошелька в Redis после блокировки", func(sCtx provider.StepCtx) {
		blockAmount, _ := strconv.ParseFloat(testData.blockRequest.Body.Amount, 64)
		expectedBalance := fmt.Sprintf("%.0f", testData.adjustmentRequest.Body.Amount-blockAmount)
		var redisValue redis.WalletFullData

		err := s.redisClient.GetWithRetry(sCtx, testData.walletCreatedEvent.Payload.WalletUUID, &redisValue)

		sCtx.Require().NoError(err, "Значение кошелька получено из Redis")
		sCtx.Assert().Equal(expectedBalance, redisValue.Balance, "Общий баланс кошелька не изменился после блокировки")
		sCtx.Require().NotEmpty(redisValue.BlockedAmounts, "В кошельке есть блокировки")

		lastBlock := redisValue.BlockedAmounts[0]

		sCtx.Assert().NotEmpty(lastBlock.UUID, "UUID блокировки не пустой")
		//sCtx.Assert().Equal(s.config.HTTP.CapUserUUID, lastBlock.UserUUID, "UUID пользователя верный")
		sCtx.Assert().Equal(3, lastBlock.Type, "Тип блокировки - manual (3)")
		sCtx.Assert().Equal(2, lastBlock.Status, "Статус блокировки - активная (2)")
		sCtx.Assert().Equal("-"+testData.blockRequest.Body.Amount, lastBlock.Amount, "Сумма блокировки верна")
		sCtx.Assert().Equal(expectedBalance, lastBlock.DeltaAvailableWithdrawalBalance, "Дельта доступного баланса верна")
		sCtx.Assert().Equal(testData.blockRequest.Body.Reason, lastBlock.Reason, "Причина блокировки верна")
		sCtx.Assert().Equal(s.config.HTTP.CapUsername, lastBlock.UserName, "Имя пользователя верно")
		sCtx.Assert().Equal(testData.blockResponse.Body.CreatedAt, lastBlock.CreatedAt, "Время создания не пустое")
		sCtx.Assert().Equal(testData.blockResponse.Body.TransactionID, lastBlock.UUID, "ID транзакции верный")
		sCtx.Assert().Zero(lastBlock.ExpiredAt, "Нет срока истечения блокировки")

		expectedAvailable := fmt.Sprintf("%.0f", testData.adjustmentRequest.Body.Amount-blockAmount)
		sCtx.Assert().Equal(expectedAvailable, redisValue.AvailableWithdrawalBalance, "Доступный для вывода баланс уменьшился на сумму блокировки")
	})

	t.WithNewStep("Получение списка блокировок", func(sCtx provider.StepCtx) {
		blockListRequest := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capClient.GetToken(sCtx)),
				"Platform-Locale": models.DefaultLocale,
				"Platform-NodeID": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"player_uuid": testData.walletCreatedEvent.Payload.PlayerUUID,
			},
		}

		blockListResponse := s.capClient.GetBlockAmountList(sCtx, blockListRequest)
		sCtx.Require().Equal(http.StatusOK, blockListResponse.StatusCode, "Статус код ответа равен 200")
		sCtx.Require().NotEmpty(blockListResponse.Body.Items, "Список блокировок не пустой")

		lastBlock := blockListResponse.Body.Items[0]
		sCtx.Assert().Equal("-"+testData.blockRequest.Body.Amount, lastBlock.Amount, "Сумма блокировки верна")
		sCtx.Assert().Equal(testData.blockRequest.Body.Currency, lastBlock.Currency, "Валюта блокировки верна")
		sCtx.Assert().Equal(testData.blockRequest.Body.Reason, lastBlock.Reason, "Причина блокировки верна")
	})

	t.WithNewStep("Получение списка кошельков пользователя", func(sCtx provider.StepCtx) {
		walletListRequest := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capClient.GetToken(sCtx)),
				"Platform-Locale": models.DefaultLocale,
				"Platform-NodeID": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"player_uuid": testData.registrationMessage.Player.ExternalID,
			},
		}

		walletListResponse := s.capClient.GetWalletList(sCtx, walletListRequest)
		sCtx.Require().Equal(http.StatusOK, walletListResponse.StatusCode, "Статус код ответа равен 200")
		sCtx.Require().NotEmpty(walletListResponse.Body.Wallets, "Список кошельков не пустой")

		wallet := walletListResponse.Body.Wallets[0]
		sCtx.Assert().Equal(testData.blockRequest.Body.Currency, wallet.Currency, "Валюта кошелька верна")
		sCtx.Assert().Equal("50", wallet.Balance, "Баланс кошелька верен (100-50 заблокировано)")
		sCtx.Assert().Equal("-50", wallet.BlockAmount, "Сумма блокировки верна")
		sCtx.Assert().Equal("100", wallet.ActualBalance, "Актуальный баланс кошелька верен")
	})

}

func (s *BlockAmountSuite) AfterAll(t provider.T) {
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

func TestBlockAmountSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(BlockAmountSuite))
}
