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
	walletRepo        *wallet.Repository
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
		s.walletRepo = wallet.NewRepository(repository.OpenConnector(t, &s.config.MySQL, repository.Wallet).DB(), &s.config.MySQL)
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
		)
		testData.authorizationResponse = playerData.Auth
	})

	t.WithNewStep("Создание депозита", func(sCtx provider.StepCtx) {
		time.Sleep(15 * time.Second)
		depositRequest := &clientTypes.Request[publicModels.DepositRequestBody]{
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

		depositResponse := s.publicClient.CreateDeposit(sCtx, depositRequest)
		sCtx.Require().Equal(http.StatusCreated, depositResponse.StatusCode, "Статус код ответа равен 201")
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
