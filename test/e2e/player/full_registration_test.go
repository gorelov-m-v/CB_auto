package test

import (
	"testing"

	"CB_auto/internal/client/factory"
	publicAPI "CB_auto/internal/client/public"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/internal/transport/kafka"
	"CB_auto/internal/transport/nats"
	"CB_auto/internal/transport/redis"
	defaultSteps "CB_auto/pkg/utils/default_steps"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type FullRegistrationSuite struct {
	suite.Suite
	config            *config.Config
	publicClient      publicAPI.PublicAPI
	kafka             *kafka.Kafka
	natsClient        *nats.NatsClient
	redisPlayerClient *redis.RedisClient
	redisWalletClient *redis.RedisClient
}

func (s *FullRegistrationSuite) BeforeAll(t provider.T) {
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

	t.WithNewStep("Инициализация Redis клиента", func(sCtx provider.StepCtx) {
		s.redisPlayerClient = redis.NewRedisClient(t, &s.config.Redis, redis.PlayerClient)
		s.redisWalletClient = redis.NewRedisClient(t, &s.config.Redis, redis.WalletClient)
	})
}

func (s *FullRegistrationSuite) TestFullRegistration(t provider.T) {
	t.Epic("Player")
	t.Feature("Полная регистрация")
	t.Title("Проверка создания игрока через полную регистрацию")
	t.Tags("player", "registration", "full")

	t.WithNewStep("Создание игрока через полную регистрацию", func(sCtx provider.StepCtx) {
		playerData := defaultSteps.CreatePlayerWithFullRegistration(
			sCtx,
			s.publicClient,
			s.kafka,
			s.config,
			s.redisPlayerClient,
			s.redisWalletClient,
		)

		sCtx.Require().NotEmpty(playerData.Auth.Body.Token, "Токен авторизации получен")
		sCtx.Require().NotEmpty(playerData.WalletData.WalletUUID, "UUID кошелька получен")
		sCtx.Require().NotEmpty(playerData.WalletData.PlayerUUID, "UUID игрока получен")
	})
}

func (s *FullRegistrationSuite) AfterAll(t provider.T) {
	kafka.CloseInstance(t)
	if s.natsClient != nil {
		s.natsClient.Close()
	}
}

func TestFullRegistrationSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(FullRegistrationSuite))
}
