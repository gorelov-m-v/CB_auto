package test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	capAPI "CB_auto/internal/client/cap"
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
	defaultSteps "CB_auto/pkg/utils/default_steps"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type SingleBetLimitSuite struct {
	suite.Suite
	config            *config.Config
	publicClient      publicAPI.PublicAPI
	capClient         capAPI.CapAPI
	kafka             *kafka.Kafka
	natsClient        *nats.NatsClient
	walletRepo        *wallet.WalletRepository
	redisPlayerClient *redis.RedisClient
	redisWalletClient *redis.RedisClient
}

func (s *SingleBetLimitSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация Public API клиента", func(sCtx provider.StepCtx) {
		s.publicClient = factory.InitClient[publicAPI.PublicAPI](sCtx, s.config, clientTypes.Public)
	})

	t.WithNewStep("Инициализация Kafka клиента", func(sCtx provider.StepCtx) {
		s.kafka = kafka.GetInstance(t, s.config)
	})

	t.WithNewStep("Инициализация NATS клиента", func(sCtx provider.StepCtx) {
		s.natsClient = nats.NewClient(&s.config.Nats)
	})

	t.WithNewStep("Соединение с базой данных wallet.", func(sCtx provider.StepCtx) {
		s.walletRepo = wallet.NewWalletRepository(repository.OpenConnector(t, &s.config.MySQL, repository.Wallet).DB(), &s.config.MySQL)
	})

	t.WithNewStep("Инициализация Redis клиента", func(sCtx provider.StepCtx) {
		s.redisPlayerClient = redis.NewRedisClient(t, &s.config.Redis, redis.PlayerClient)
		s.redisWalletClient = redis.NewRedisClient(t, &s.config.Redis, redis.WalletClient)
	})

	t.WithNewStep("Инициализация CAP API клиента", func(sCtx provider.StepCtx) {
		s.capClient = factory.InitClient[capAPI.CapAPI](sCtx, s.config, clientTypes.Cap)
	})
}

func (s *SingleBetLimitSuite) TestSingleBetLimit(t provider.T) {
	t.Epic("Payment")
	t.Feature("Депозит")
	t.Title("Проверка создания депозита в Kafka, NATS, Redis, MySQL")
	t.Tags("wallet", "payment")

	var testData struct {
		authorizationResponse *clientTypes.Response[publicModels.TokenCheckResponseBody]
		walletAggregate       redis.WalletFullData
		depositRequest        *clientTypes.Request[publicModels.DepositRequestBody]
		depositEvent          *nats.NatsMessage[nats.DepositedMoneyPayload]
		transactionMessage    kafka.TransactionMessage
		projectionMessage     kafka.ProjectionSourceMessage
	}

	t.WithNewStep("Создание и верификация игрока", func(sCtx provider.StepCtx) {
		playerData := defaultSteps.CreateVerifiedPlayer(
			sCtx,
			s.publicClient,
			s.capClient,
			s.kafka,
			s.config,
			s.redisPlayerClient,
			s.redisWalletClient,
			s.natsClient,
			0,
		)
		testData.authorizationResponse = playerData.Auth
		testData.walletAggregate = playerData.WalletData
	})

	t.WithNewStep("Public API: Создание депозита", func(sCtx provider.StepCtx) {
		time.Sleep(5 * time.Second)
		testData.depositRequest = &clientTypes.Request[publicModels.DepositRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
			},
			Body: &publicModels.DepositRequestBody{
				Amount:          "10",
				PaymentMethodID: int(publicModels.Fake),
				Currency:        s.config.Node.DefaultCurrency,
				Country:         s.config.Node.DefaultCountry,
				Redirect: publicModels.DepositRedirectURLs{
					Failed:  publicModels.DepositRedirectURLFailed,
					Success: publicModels.DepositRedirectURLSuccess,
					Pending: publicModels.DepositRedirectURLPending,
				},
			},
		}

		resp := s.publicClient.CreateDeposit(sCtx, testData.depositRequest)
		sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Public API: Статус-код 201")
	})

	t.WithNewStep("Проверка сообщения о транзакции в Kafka", func(sCtx provider.StepCtx) {
		testData.transactionMessage = kafka.FindMessageByFilter(sCtx, s.kafka, func(msg kafka.TransactionMessage) bool {
			return msg.PlayerID == testData.walletAggregate.PlayerUUID &&
				msg.Transaction.Direction == "deposit" &&
				msg.Transaction.Amount == testData.depositRequest.Body.Amount &&
				msg.Transaction.CurrencyCode == testData.depositRequest.Body.Currency &&
				msg.Transaction.Status == kafka.TransactionStatusSuccess
		})
		sCtx.Require().NotEmpty(testData.transactionMessage.Transaction.TransactionID, "Kafka: Сообщение о транзакции найдено")
		sCtx.Assert().Equal(testData.walletAggregate.PlayerUUID, testData.transactionMessage.PlayerID, "Kafka: Проверка параметра playerId")
		sCtx.Assert().Equal(s.config.Node.ProjectID, testData.transactionMessage.NodeID, "Kafka: Проверка параметра nodeId")
		sCtx.Assert().Equal(kafka.TransactionDirectionDeposit, testData.transactionMessage.Transaction.Direction, "Kafka: Проверка параметра direction")
		sCtx.Assert().Equal(testData.depositRequest.Body.Amount, testData.transactionMessage.Transaction.Amount, "Kafka: Проверка параметра amount")
		sCtx.Assert().Equal(testData.depositRequest.Body.Currency, testData.transactionMessage.Transaction.CurrencyCode, "Kafka: Проверка параметра currencyCode")
		sCtx.Assert().Equal(kafka.TransactionStatusSuccess, testData.transactionMessage.Transaction.Status, "Kafka: Проверка параметра status")
		sCtx.Assert().NotZero(testData.transactionMessage.Transaction.CreatedAt, "Kafka: Проверка параметра createdAt")
		sCtx.Assert().NotZero(testData.transactionMessage.Transaction.UpdatedAt, "Kafka: Проверка параметра updatedAt")
		sCtx.Assert().True(testData.transactionMessage.Meta.FirstDep, "Kafka: Проверка параметра firstDep")
		sCtx.Assert().Equal(testData.depositRequest.Body.Amount, testData.transactionMessage.Meta.FirstDepAmount, "Kafka: Проверка параметра firstDepAmount")
		sCtx.Assert().Equal(kafka.PaymentMethodFake, testData.transactionMessage.Transaction.PaymentMethod, "Kafka: Проверка параметра paymentMethod")
		sCtx.Assert().Equal(kafka.PaymentMethodAliasFake, testData.transactionMessage.Transaction.PaymentMethodAlias, "Kafka: Проверка параметра paymentMethodAlias")
	})

	t.WithNewStep("Проверка ивента NATS о зачислении депозита", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, testData.walletAggregate.PlayerUUID)
		testData.depositEvent = nats.FindMessageInStream(sCtx, s.natsClient, subject, func(msg nats.DepositedMoneyPayload, msgType string) bool {
			return string(nats.DepositedMoney) == msgType &&
				msg.Amount == testData.depositRequest.Body.Amount &&
				msg.CurrencyCode == testData.depositRequest.Body.Currency
		})
		sCtx.Require().NotNil(testData.depositEvent, "NATS: Сообщение deposited_money найдено")
		sCtx.Assert().Equal(testData.depositRequest.Body.Amount, testData.depositEvent.Payload.Amount, "NATS: Проверка параметра amount")
		sCtx.Assert().Equal(testData.depositRequest.Body.Currency, testData.depositEvent.Payload.CurrencyCode, "NATS: Проверка параметра currency_code")
		sCtx.Assert().NotEmpty(testData.depositEvent.Payload.UUID, "NATS: Проверка параметра uuid")
		sCtx.Assert().Equal(s.config.Node.ProjectID, testData.depositEvent.Payload.NodeUUID, "NATS: Проверка параметра node_uuid")
		sCtx.Assert().Equal(testData.depositEvent.Payload.Status, nats.TransactionStatusSuccess, "NATS: Проверка параметра status")
	})

	t.WithNewAsyncStep("Проверка сообщения о депозите в топике wallet.v8.projectionSource", func(sCtx provider.StepCtx) {
		testData.projectionMessage = kafka.FindMessageByFilter(sCtx, s.kafka, func(msg kafka.ProjectionSourceMessage) bool {
			return msg.Type == kafka.ProjectionEventDepositedMoney &&
				msg.PlayerUUID == testData.walletAggregate.PlayerUUID &&
				msg.WalletUUID == testData.walletAggregate.WalletUUID
		})
		sCtx.Require().NotEmpty(testData.projectionMessage.Type, "Kafka: Сообщение о депозите в projectionSource найдено")

		sCtx.Assert().Equal(string(kafka.ProjectionEventDepositedMoney), string(testData.projectionMessage.Type), "Kafka: Проверка параметра type")
		sCtx.Assert().Equal(testData.walletAggregate.WalletUUID, testData.projectionMessage.WalletUUID, "Kafka: Проверка параметра wallet_uuid")
		sCtx.Assert().Equal(testData.walletAggregate.PlayerUUID, testData.projectionMessage.PlayerUUID, "Kafka: Проверка параметра player_uuid")
		sCtx.Assert().Equal(s.config.Node.ProjectID, testData.projectionMessage.NodeUUID, "Kafka: Проверка параметра node_uuid")
		sCtx.Assert().Equal(testData.depositRequest.Body.Currency, testData.projectionMessage.Currency, "Kafka: Проверка параметра currency")
		sCtx.Assert().Equal(int(testData.projectionMessage.Timestamp), testData.projectionMessage.Timestamp, "Kafka: Проверка параметра timestamp")
		sCtx.Assert().NotEmpty(testData.projectionMessage.SeqNumberNodeUUID, "Kafka: Проверка параметра seq_number_node_uuid")

		var payloadDepositedMoney kafka.ProjectionPayloadDepositedMoney
		err := testData.projectionMessage.UnmarshalPayloadTo(&payloadDepositedMoney)
		sCtx.Require().NoError(err, "Kafka: Payload успешно распакован")

		sCtx.Assert().Equal(testData.depositEvent.Payload.UUID, payloadDepositedMoney.UUID, "Kafka: Проверка параметра uuid в payload")
		sCtx.Assert().Equal(testData.depositEvent.Payload.CurrencyCode, payloadDepositedMoney.CurrencyCode, "Kafka: Проверка параметра currency_code в payload")
		sCtx.Assert().Equal(testData.depositEvent.Payload.Amount, payloadDepositedMoney.Amount, "Kafka: Проверка параметра amount в payload")
		sCtx.Assert().Equal(int(testData.depositEvent.Payload.Status), payloadDepositedMoney.Status, "Kafka: Проверка параметра status в payload")
		sCtx.Assert().Equal(testData.depositEvent.Payload.NodeUUID, payloadDepositedMoney.NodeUUID, "Kafka: Проверка параметра node_uuid в payload")
	})

	t.WithNewAsyncStep("Проверка данных кошелька в Redis после депозита", func(sCtx provider.StepCtx) {
		var walletAggregate redis.WalletFullData
		err := s.redisWalletClient.GetWithSeqCheck(sCtx, testData.walletAggregate.WalletUUID, &walletAggregate, testData.depositEvent.Sequence)
		sCtx.Assert().NoError(err, "Redis: Данные кошелька успешно получены")

		sCtx.Assert().Equal(int(testData.depositEvent.Sequence), walletAggregate.LastSeqNumber, "Redis: Проверка параметра LastSeqNumber")
		sCtx.Assert().Equal(testData.depositRequest.Body.Amount, walletAggregate.Balance, "Redis: Проверка параметра Balance")
		sCtx.Assert().Equal("0", walletAggregate.AvailableWithdrawalBalance, "Redis: Проверка параметра AvailableWithdrawalBalance")
		sCtx.Assert().NotEmpty(walletAggregate.Deposits, "Redis: Массив депозитов не пустой")

		depositData := walletAggregate.Deposits[0]
		sCtx.Assert().Equal(testData.depositEvent.Payload.UUID, depositData.UUID, "Redis: Проверка параметра UUID")
		sCtx.Assert().Equal(testData.depositEvent.Payload.NodeUUID, depositData.NodeUUID, "Redis: Проверка параметра NodeUUID")
		sCtx.Assert().Equal(testData.depositRequest.Body.Currency, depositData.CurrencyCode, "Redis: Проверка параметра CurrencyCode")
		sCtx.Assert().Equal(int(testData.depositEvent.Payload.Status), int(depositData.Status), "Redis: Проверка параметра Status")
		sCtx.Assert().Equal(testData.depositRequest.Body.Amount, depositData.Amount, "Redis: Проверка параметра Amount")
		sCtx.Assert().Equal(testData.depositEvent.Payload.BonusID, depositData.BonusID, "Redis: Проверка параметра BonusID")
		sCtx.Assert().Equal("0", depositData.WageringAmount, "Redis: Проверка параметра WageringAmount")
	})
}

func (s *SingleBetLimitSuite) AfterAll(t provider.T) {
	kafka.CloseInstance(t)
	if s.natsClient != nil {
		s.natsClient.Close()
	}
}

func TestSingleBetLimitSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(SingleBetLimitSuite))
}
