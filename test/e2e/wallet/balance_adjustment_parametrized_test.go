package test

import (
	"fmt"
	"net/http"
	"testing"

	capAPI "CB_auto/internal/client/cap"
	capModels "CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/factory"
	publicAPI "CB_auto/internal/client/public"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/internal/repository/wallet"
	"CB_auto/internal/transport/kafka"
	"CB_auto/internal/transport/nats"
	"CB_auto/internal/transport/redis"
	"CB_auto/pkg/mappers"
	"CB_auto/pkg/utils"
	defaultSteps "CB_auto/pkg/utils/default_steps"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type BalanceAdjustmentParam struct {
	Direction     capModels.DirectionType
	OperationType capModels.OperationType
	ReasonType    capModels.ReasonType
	Description   string
}

type ParametrizedBalanceAdjustmentSuite struct {
	suite.Suite
	config                 *config.Config
	publicClient           publicAPI.PublicAPI
	capClient              capAPI.CapAPI
	kafka                  *kafka.Kafka
	natsClient             *nats.NatsClient
	database               *repository.Connector
	walletRepo             *wallet.WalletRepository
	redisWalletClient      *redis.RedisClient
	redisPlayerClient      *redis.RedisClient
	ParamBalanceAdjustment []BalanceAdjustmentParam
}

func (s *ParametrizedBalanceAdjustmentSuite) BeforeAll(t provider.T) {
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

	t.WithNewStep("Инициализация Redis клиентов", func(sCtx provider.StepCtx) {
		s.redisWalletClient = redis.NewRedisClient(t, &s.config.Redis, redis.WalletClient)
		s.redisPlayerClient = redis.NewRedisClient(t, &s.config.Redis, redis.PlayerClient)
	})

	t.WithNewStep("Инициализация NATS клиента", func(sCtx provider.StepCtx) {
		s.natsClient = nats.NewClient(&s.config.Nats)
	})

	t.WithNewStep("Соединение с базой данных", func(sCtx provider.StepCtx) {
		s.walletRepo = wallet.NewWalletRepository(repository.OpenConnector(t, &s.config.MySQL, repository.Wallet).DB(), &s.config.MySQL)
	})

	s.ParamBalanceAdjustment = []BalanceAdjustmentParam{
		{
			Direction:     capModels.DirectionIncrease,
			OperationType: capModels.OperationTypeCorrection,
			ReasonType:    capModels.ReasonMalfunction,
			Description:   "Корректировка из-за технического сбоя",
		},
		{
			Direction:     capModels.DirectionIncrease,
			OperationType: capModels.OperationTypeDeposit,
			ReasonType:    capModels.ReasonOperationalMistake,
			Description:   "Депозит из-за операционной ошибки",
		},
		{
			Direction:     capModels.DirectionIncrease,
			OperationType: capModels.OperationTypeGift,
			ReasonType:    capModels.ReasonBalanceCorrection,
			Description:   "Подарок для корректировки баланса",
		},
		{
			Direction:     capModels.DirectionIncrease,
			OperationType: capModels.OperationTypeCashback,
			ReasonType:    capModels.ReasonOperationalMistake,
			Description:   "Кэшбэк из-за операционной ошибки",
		},
		{
			Direction:     capModels.DirectionIncrease,
			OperationType: capModels.OperationTypeTournamentPrize,
			ReasonType:    capModels.ReasonMalfunction,
			Description:   "Приз турнира из-за технического сбоя",
		},
		{
			Direction:     capModels.DirectionIncrease,
			OperationType: capModels.OperationTypeJackpot,
			ReasonType:    capModels.ReasonBalanceCorrection,
			Description:   "Джекпот для корректировки баланса",
		},
		{
			Direction:     capModels.DirectionDecrease,
			OperationType: capModels.OperationTypeCorrection,
			ReasonType:    capModels.ReasonBalanceCorrection,
			Description:   "Уменьшение для корректировки баланса",
		},
		{
			Direction:     capModels.DirectionDecrease,
			OperationType: capModels.OperationTypeWithdrawal,
			ReasonType:    capModels.ReasonOperationalMistake,
			Description:   "Вывод из-за операционной ошибки",
		},
		{
			Direction:     capModels.DirectionDecrease,
			OperationType: capModels.OperationTypeGift,
			ReasonType:    capModels.ReasonMalfunction,
			Description:   "Отмена подарка из-за технического сбоя",
		},
		{
			Direction:     capModels.DirectionDecrease,
			OperationType: capModels.OperationTypeReferralCommission,
			ReasonType:    capModels.ReasonOperationalMistake,
			Description:   "Отмена реферальной комиссии из-за ошибки",
		},
		{
			Direction:     capModels.DirectionDecrease,
			OperationType: capModels.OperationTypeTournamentPrize,
			ReasonType:    capModels.ReasonBalanceCorrection,
			Description:   "Корректировка выигрыша в турнире",
		},
		{
			Direction:     capModels.DirectionDecrease,
			OperationType: capModels.OperationTypeJackpot,
			ReasonType:    capModels.ReasonMalfunction,
			Description:   "Отмена джекпота из-за технического сбоя",
		},
	}
}

func (s *ParametrizedBalanceAdjustmentSuite) TableTestBalanceAdjustment(t provider.T, param BalanceAdjustmentParam) {
	t.Epic("Wallet")
	t.Feature("Корректировка баланса")
	t.Title(fmt.Sprintf("Проверка корректировки баланса игрока: %s", param.Description))
	t.Tags("wallet", "cap")

	var testData struct {
		authToken             string
		walletAggregate       redis.WalletFullData
		adjustmentRequest     *clientTypes.Request[capModels.CreateBalanceAdjustmentRequestBody]
		adjustmentResponse    *clientTypes.Response[struct{}]
		balanceAdjustedEvent  *nats.NatsMessage[nats.BalanceAdjustedPayload]
		projectionAdjustEvent kafka.ProjectionSourceMessage
		expectedBalance       float64
	}

	t.WithNewStep("Создание верифицированного игрока с балансом", func(sCtx provider.StepCtx) {
		depositAmount := 150.0
		playerData := defaultSteps.CreateVerifiedPlayer(
			sCtx,
			s.publicClient,
			s.capClient,
			s.kafka,
			s.config,
			s.redisPlayerClient,
			s.redisWalletClient,
			s.natsClient,
			depositAmount,
		)
		testData.authToken = playerData.Auth.Body.Token
		testData.walletAggregate = playerData.WalletData
		testData.expectedBalance = depositAmount
	})

	t.WithNewStep("CAP API: Выполнение корректировки баланса", func(sCtx provider.StepCtx) {
		testData.adjustmentRequest = &clientTypes.Request[capModels.CreateBalanceAdjustmentRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capClient.GetToken(sCtx)),
				"Platform-NodeID": s.config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"player_uuid": testData.walletAggregate.PlayerUUID,
			},
			Body: &capModels.CreateBalanceAdjustmentRequestBody{
				Currency:      s.config.Node.DefaultCurrency,
				Amount:        100.0,
				Reason:        param.ReasonType,
				OperationType: param.OperationType,
				Direction:     param.Direction,
				Comment:       utils.Get(utils.LETTERS, 25),
			},
		}

		testData.adjustmentResponse = s.capClient.CreateBalanceAdjustment(sCtx, testData.adjustmentRequest)
		sCtx.Require().Equal(http.StatusOK, testData.adjustmentResponse.StatusCode, "CAP API: Статус-код 200")

		if param.Direction == capModels.DirectionDecrease {
			testData.expectedBalance = testData.expectedBalance - testData.adjustmentRequest.Body.Amount
		} else {
			testData.expectedBalance = testData.expectedBalance + testData.adjustmentRequest.Body.Amount
		}
	})

	t.WithNewStep("Проверка события корректировки баланса в NATS", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.%s", s.config.Nats.StreamPrefix,
			testData.walletAggregate.PlayerUUID,
			testData.walletAggregate.WalletUUID)

		testData.balanceAdjustedEvent = nats.FindMessageInStream(
			sCtx, s.natsClient, subject, func(payload nats.BalanceAdjustedPayload, msgType string) bool {
				return msgType == string(nats.BalanceAdjustedType)
			})

		sCtx.Require().NotNil(testData.balanceAdjustedEvent, "NATS: Событие balance_adjusted получено")

		expectedAmount := testData.adjustmentRequest.Body.Amount
		actualAmount := mappers.StringToAmount(testData.balanceAdjustedEvent.Payload.Amount)

		if param.Direction == capModels.DirectionDecrease {
			expectedAmount = -expectedAmount
		}

		sCtx.Assert().Equal(expectedAmount, actualAmount, "NATS: Сумма корректировки совпадает с учетом направления")
		sCtx.Assert().Equal(
			mappers.MapDirectionToNats(testData.adjustmentRequest.Body.Direction),
			testData.balanceAdjustedEvent.Payload.Direction,
			"NATS: Проверка параметра direction")
		sCtx.Assert().Equal(
			mappers.MapOperationTypeToNats(testData.adjustmentRequest.Body.OperationType),
			testData.balanceAdjustedEvent.Payload.OperationType,
			"NATS: Проверка параметра operation_type")
		sCtx.Assert().Equal(
			mappers.MapReasonToNats(testData.adjustmentRequest.Body.Reason),
			testData.balanceAdjustedEvent.Payload.Reason,
			"NATS: Проверка параметра reason")
		sCtx.Assert().Equal(testData.adjustmentRequest.Body.Comment, testData.balanceAdjustedEvent.Payload.Comment, "NATS: Проверка параметра comment")
		sCtx.Assert().Equal(testData.adjustmentRequest.Body.Currency, testData.balanceAdjustedEvent.Payload.Currenc, "NATS: Проверка параметра currency")
		sCtx.Assert().NotEmpty(testData.balanceAdjustedEvent.Payload.UserUUID, "NATS: Проверка параметра user_uuid")
		sCtx.Assert().Equal(s.config.HTTP.CapUsername, testData.balanceAdjustedEvent.Payload.UserName, "NATS: Проверка параметра user_name")
	})

	t.WithNewAsyncStep("Проверка сообщения о корректировке баланса в топике wallet.v8.projectionSource", func(sCtx provider.StepCtx) {
		testData.projectionAdjustEvent = kafka.FindMessageByFilter(
			sCtx, s.kafka, func(msg kafka.ProjectionSourceMessage) bool {
				return msg.Type == kafka.ProjectionEventBalanceAdjusted &&
					msg.PlayerUUID == testData.walletAggregate.PlayerUUID &&
					msg.WalletUUID == testData.walletAggregate.WalletUUID
			})
		sCtx.Assert().NotEmpty(testData.projectionAdjustEvent.Type, "Kafka: Сообщение balance_adjusted найдено в топике wallet.v8.projectionSource")
		sCtx.Assert().Equal(testData.walletAggregate.PlayerUUID, testData.projectionAdjustEvent.PlayerUUID, "Kafka: Проверка параметра player_uuid")
		sCtx.Assert().Equal(testData.walletAggregate.WalletUUID, testData.projectionAdjustEvent.WalletUUID, "Kafka: Проверка параметра wallet_uuid")
		sCtx.Assert().Equal(s.config.Node.DefaultCurrency, testData.projectionAdjustEvent.Currency, "Kafka: Проверка параметра currency")
		sCtx.Assert().Equal(testData.balanceAdjustedEvent.Sequence, testData.projectionAdjustEvent.SeqNumber, "Kafka: Проверка параметра seq_number")
		sCtx.Assert().Equal(s.config.Node.ProjectID, testData.projectionAdjustEvent.NodeUUID, "Kafka: Проверка параметра node_uuid")
		sCtx.Assert().Equal(s.config.Node.DefaultCurrency, testData.projectionAdjustEvent.Currency, "Kafka: Проверка параметра currency")
		sCtx.Assert().NotEmpty(testData.projectionAdjustEvent.SeqNumberNodeUUID, "Kafka: Проверка параметра seq_number_node_uuid")

		var adjustmentPayload kafka.ProjectionPayloadAdjustment
		err := testData.projectionAdjustEvent.UnmarshalPayloadTo(&adjustmentPayload)
		sCtx.Assert().NoError(err, "Kafka: Payload успешно распакован")

		expectedAmount := testData.adjustmentRequest.Body.Amount
		actualAmount := mappers.StringToAmount(adjustmentPayload.Amount)
		if param.Direction == capModels.DirectionDecrease {
			expectedAmount = -expectedAmount
		}
		sCtx.Assert().Equal(expectedAmount, actualAmount, "Kafka: Проверка параметра amount")
		sCtx.Assert().Equal(
			mappers.MapDirectionToNats(testData.adjustmentRequest.Body.Direction),
			adjustmentPayload.Direction,
			"Kafka: Проверка параметра direction")
		sCtx.Assert().Equal(
			mappers.MapOperationTypeToNats(testData.adjustmentRequest.Body.OperationType),
			adjustmentPayload.OperationType,
			"Kafka: Проверка параметра operation_type")
		sCtx.Assert().Equal(
			mappers.MapReasonToNats(testData.adjustmentRequest.Body.Reason),
			adjustmentPayload.Reason,
			"Kafka: Проверка параметра reason")
		sCtx.Assert().Equal(testData.adjustmentRequest.Body.Comment, adjustmentPayload.Comment, "Kafka: Проверка параметра comment")
		sCtx.Assert().Equal(s.config.Node.DefaultCurrency, adjustmentPayload.Currenc, "Kafka: Проверка параметра currency")
		sCtx.Assert().Equal(s.config.HTTP.CapUsername, adjustmentPayload.UserName, "Kafka: Проверка параметра user_name")
		sCtx.Assert().NotEmpty(adjustmentPayload.UserUUID, "Kafka: Проверка параметра user_uuid")
	})

	t.WithNewAsyncStep("Проверка данных кошелька в Redis", func(sCtx provider.StepCtx) {
		var redisValue redis.WalletFullData
		err := s.redisWalletClient.GetWithSeqCheck(
			sCtx,
			testData.walletAggregate.WalletUUID,
			&redisValue,
			int(testData.balanceAdjustedEvent.Sequence))
		sCtx.Assert().NoError(err, "Redis: Значение кошелька получено")

		expectedBalance := fmt.Sprintf("%.0f", testData.expectedBalance)
		sCtx.Assert().Equal(expectedBalance, redisValue.Balance, "Redis: Проверка параметра balance")
		sCtx.Assert().Equal(int(testData.balanceAdjustedEvent.Sequence), redisValue.LastSeqNumber, "Redis: Проверка параметра last_seq_number")
	})
}

func (s *ParametrizedBalanceAdjustmentSuite) AfterAll(t provider.T) {
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

func TestParametrizedBalanceAdjustmentSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(ParametrizedBalanceAdjustmentSuite))
}
