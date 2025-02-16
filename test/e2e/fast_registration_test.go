package test

import (
	client "CB_auto/internal/client"
	publicAPI "CB_auto/internal/client/public"
	"CB_auto/internal/client/public/models"
	"CB_auto/internal/config"
	"CB_auto/pkg/utils"
	"fmt"
	"testing"

	"CB_auto/internal/transport/nats"
	"CB_auto/internal/transport/redis"

	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"

	"CB_auto/internal/transport/kafka"
)

type FastRegistrationSuite struct {
	suite.Suite
	client        *client.Client
	config        *config.Config
	publicService publicAPI.PublicAPI
	natsClient    *nats.NatsClient
	redisClient   *redis.RedisClient
	kafka         *kafka.Kafka
}

func (s *FastRegistrationSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла.", func(sCtx provider.StepCtx) {
		cfg, err := config.ReadConfig()
		if err != nil {
			t.Fatalf("Ошибка при чтении конфигурации: %v", err)
		}
		s.config = cfg
	})

	t.WithNewStep("Инициализация http-клиента и Public API сервиса.", func(sCtx provider.StepCtx) {
		client, err := client.InitClient(s.config, client.Public)
		if err != nil {
			t.Fatalf("InitClient не удался: %v", err)
		}
		s.client = client
		s.publicService = publicAPI.NewPublicClient(s.client)
	})

	t.WithNewStep("Инициализация NATS клиента.", func(sCtx provider.StepCtx) {
		client, err := nats.NewClient(&s.config.Nats)
		if err != nil {
			t.Fatalf("NewClient не удался: %v", err)
		}
		s.natsClient = client
	})

	t.WithNewStep("Инициализация Redis клиента.", func(sCtx provider.StepCtx) {
		client, err := redis.NewRedisClient(&s.config.Redis)
		if err != nil {
			t.Fatalf("Redis init failed: %v", err)
		}
		s.redisClient = client
	})

	t.WithNewStep("Инициализация Kafka.", func(sCtx provider.StepCtx) {
		fmt.Printf("Initializing Kafka consumer for topic: %s\n", s.config.Kafka.PlayerTopic)
		s.kafka = kafka.NewConsumer(
			[]string{s.config.Kafka.Brokers},
			s.config.Kafka.PlayerTopic,
			s.config.Node.GroupID,
			s.config.Kafka.GetTimeout(),
		)
		s.kafka.StartReading(t)
	})
}

func (s *FastRegistrationSuite) TestFastRegistration(t provider.T) {
	t.Epic("Users")
	t.Feature("Регистрация пользователя")
	t.Tags("Public", "Users")
	t.Title("Проверка быстрой регистрации пользователя")

	type TestData struct {
		registrationResponse      *models.FastRegistrationResponseBody
		playerRegistrationMessage *kafka.PlayerMessage
		walletCreatedPayload      *nats.WalletCreatedPayload
	}
	var testData TestData

	t.WithNewStep("Регистрация пользователя.", func(sCtx provider.StepCtx) {
		createReq := &client.Request[models.FastRegistrationRequestBody]{
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: &models.FastRegistrationRequestBody{
				Country:  s.config.Node.DefaultCountry,
				Currency: s.config.Node.DefaultCurrency,
			},
		}

		createResp := s.publicService.FastRegistration(createReq)
		testData.registrationResponse = &createResp.Body

		sCtx.Require().NotEmpty(createResp.Body.Username, "Username is empty")
		sCtx.Require().NotEmpty(createResp.Body.Password, "Password is empty")

		sCtx.WithAttachments(allure.NewAttachment("FastRegistration Request", allure.JSON, utils.CreateHttpAttachRequest[models.FastRegistrationRequestBody](createReq)))
		sCtx.WithAttachments(allure.NewAttachment("FastRegistration Response", allure.JSON, utils.CreateHttpAttachResponse[models.FastRegistrationResponseBody](createResp)))
	})

	t.WithNewStep("Получение сообщения о регистрации из топика player.v1.account.", func(sCtx provider.StepCtx) {
		accountID := testData.registrationResponse.Username

		message := kafka.FindMessageByFilter[kafka.PlayerMessage](s.kafka, t, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == "player.signUpFast" &&
				msg.Player.AccountID == accountID
		})

		playerRegistrationMessage := kafka.ParseMessage[kafka.PlayerMessage](t, message)
		testData.playerRegistrationMessage = &playerRegistrationMessage

		sCtx.Require().NotEmpty(playerRegistrationMessage.Player.ExternalID, "Player External ID is not empty")
		sCtx.WithAttachments(allure.NewAttachment("Kafka Message", allure.JSON, utils.CreatePrettyJSON(playerRegistrationMessage)))
	})

	t.WithNewStep("Проверка создания кошелька в NATS.", func(sCtx provider.StepCtx) {
		playerUUID := testData.playerRegistrationMessage.Player.ExternalID
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, playerUUID)

		s.natsClient.Subscribe(t, subject)

		message := nats.FindMessageByFilter[nats.WalletCreatedPayload](s.natsClient, t, func(wallet nats.WalletCreatedPayload) bool {
			return wallet.WalletType == nats.TypeReal &&
				wallet.WalletStatus == nats.StatusEnabled &&
				wallet.IsBasic
		})

		walletData := nats.ParseMessage[nats.WalletCreatedPayload](t, message)
		testData.walletCreatedPayload = &walletData

		sCtx.Require().NotEmpty(walletData.WalletUUID, "Expected non-empty wallet UUID")
		sCtx.Require().Equal(playerUUID, walletData.PlayerUUID, "Wrong player UUID")
		sCtx.Require().Equal("00000000-0000-0000-0000-000000000000", walletData.PlayerBonusUUID, "Expected nil bonus UUID")
		sCtx.Require().Equal(nats.TypeReal, walletData.WalletType, "Expected real wallet type")
		sCtx.Require().Equal(nats.StatusEnabled, walletData.WalletStatus, "Expected enabled status")
		sCtx.Require().Equal(s.config.Node.DefaultCurrency, walletData.Currency, "Wrong currency")
		sCtx.Require().Equal("0", walletData.Balance, "Non-zero initial balance")
		sCtx.Require().True(walletData.IsDefault, "Not marked as default")
		sCtx.Require().True(walletData.IsBasic, "Not marked as basic")
		sCtx.Require().NotEmpty(walletData.CreatedAt, "Missing creation timestamp")
		sCtx.Require().NotEmpty(walletData.UpdatedAt, "Missing update timestamp")

		sCtx.WithAttachments(allure.NewAttachment("NATS Wallet Message", allure.JSON, utils.CreatePrettyJSON(walletData)))
	})

	t.WithNewStep("Проверка значения в Redis.", func(sCtx provider.StepCtx) {
		key := testData.playerRegistrationMessage.Player.ExternalID
		wallets := s.redisClient.GetWithRetry(t, key)

		var wallet redis.WalletData
		for _, w := range wallets {
			wallet = w
			break
		}

		sCtx.Require().Equal(testData.walletCreatedPayload.Currency, wallet.Currency, "Wrong currency in Redis")
		sCtx.Require().Equal(1, wallet.Type, "Wrong wallet type in Redis")
		sCtx.Require().Equal(1, wallet.Status, "Wrong wallet status in Redis")
		sCtx.Require().Equal(testData.walletCreatedPayload.WalletUUID, wallet.WalletUUID, "Wrong wallet UUID in Redis")

		sCtx.WithAttachments(allure.NewAttachment("Redis Value", allure.JSON, utils.CreatePrettyJSON(wallets)))
	})

}

func (s *FastRegistrationSuite) AfterAll(t provider.T) {
	if s.natsClient != nil {
		s.natsClient.Close()
	}
	if s.redisClient != nil {
		s.redisClient.Close()
	}
}

func TestFastRegistrationSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(FastRegistrationSuite))
}
