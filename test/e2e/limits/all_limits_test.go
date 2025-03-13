package test

import (
	"testing"

	"CB_auto/internal/client/cap"
	"CB_auto/internal/client/factory"
	"CB_auto/internal/client/public"
	"CB_auto/internal/client/types"
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

// SharedConnections хранит общий набор соединений для всех тестов.
type SharedConnections struct {
	Config       *config.Config
	PublicClient public.PublicAPI
	CapClient    cap.CapAPI
	WalletRepo   *wallet.Repository
	RedisClient  *redis.RedisClient
	Kafka        *kafka.Kafka
	NatsClient   *nats.NatsClient
}

// SharedSuite — интерфейс для дочерних сьютов, которым можно передать общие соединения.
// Он расширяет suite.TestSuite, чтобы удовлетворять требованиям s.RunSuite.
type SharedSuite interface {
	suite.TestSuite
	SetShared(*SharedConnections)
}

// AllLimitsSuite – родительский сьют, в котором создаются и закрываются все общие соединения.
type AllLimitsSuite struct {
	suite.Suite
	shared *SharedConnections
}

func (s *AllLimitsSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Инициализация общего набора соединений", func(sCtx provider.StepCtx) {
		cfg := config.ReadConfig(t)
		publicClient := factory.InitClient[public.PublicAPI](sCtx, cfg, types.Public)
		capClient := factory.InitClient[cap.CapAPI](sCtx, cfg, types.Cap)
		walletRepo := wallet.NewRepository(repository.OpenConnector(t, &cfg.MySQL, repository.Wallet).DB(), &cfg.MySQL)
		redisClient := redis.NewRedisClient(t, &cfg.Redis, redis.WalletClient)
		kafkaClient := kafka.GetInstance(t, cfg)
		natsClient := nats.NewClient(&cfg.Nats)

		s.shared = &SharedConnections{
			Config:       cfg,
			PublicClient: publicClient,
			CapClient:    capClient,
			WalletRepo:   walletRepo,
			RedisClient:  redisClient,
			Kafka:        kafkaClient,
			NatsClient:   natsClient,
		}
	})
}

func (s *AllLimitsSuite) AfterAll(t provider.T) {
	t.WithNewStep("Закрытие общего набора соединений", func(sCtx provider.StepCtx) {
		kafka.CloseInstance(t)
		if s.shared.NatsClient != nil {
			s.shared.NatsClient.Close()
		}
		// При необходимости можно закрыть и другие соединения (Redis, БД и т.д.)
	})
}

// runSharedTest передаёт общие соединения дочернему сьюту и запускает его.
func (s *AllLimitsSuite) runSharedTest(t provider.T, child SharedSuite) {
	child.SetShared(s.shared)
	s.RunSuite(t, child)
}

func (s *AllLimitsSuite) TestCasinoLossLimit(t provider.T) {
	t.Parallel()
	var suite CasinoLossLimitSuite
	s.runSharedTest(t, &suite)
}

// func (s *AllLimitsSuite) TestSingleBetLimit(t provider.T) {
// 	t.Parallel()
// 	var suite SingleBetLimitSuite
// 	s.runSharedTest(t, &suite)
// }

func (s *AllLimitsSuite) TestTurnoverLimit(t provider.T) {
	t.Parallel()
	var suite TurnoverLimitSuite
	s.runSharedTest(t, &suite)
}

func TestAllLimits(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(AllLimitsSuite))
}
