package test

import (
	"CB_auto/test/config"
	httpClient "CB_auto/test/transport/http"
	publicAPI "CB_auto/test/transport/http/public"
	"CB_auto/test/transport/http/public/models"
	"CB_auto/test/utils"
	"fmt"
	"testing"

	"CB_auto/test/transport/nats"
	"CB_auto/test/transport/redis"

	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"

	"CB_auto/test/transport/kafka"
)

type FastRegistrationSuite struct {
	suite.Suite
	client        *httpClient.Client
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
		client, err := httpClient.InitClient(s.config, httpClient.Public)
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
		registrationResponse *models.FastRegistrationResponseBody
		playerMessage        *kafka.PlayerMessage
	}
	var testData TestData

	t.WithNewStep("Быстрая регистрация пользователя.", func(sCtx provider.StepCtx) {
		createReq := &httpClient.Request[models.FastRegistrationRequestBody]{
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

		t.Require().NotEmpty(createResp.Body.Username)
		t.Require().NotEmpty(createResp.Body.Password)

		sCtx.WithAttachments(allure.NewAttachment("FastRegistration Request", allure.JSON, utils.CreateHttpAttachRequest(createReq)))
		sCtx.WithAttachments(allure.NewAttachment("FastRegistration Response", allure.JSON, utils.CreateHttpAttachResponse(createResp)))
	})

	t.WithNewStep("Проверка сообщения из Kafka о регистрации.", func(sCtx provider.StepCtx) {
		accountID := testData.registrationResponse.Username

		message := kafka.FindMessageByFilter[kafka.PlayerMessage](s.kafka, t, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == "player.signUpFast" &&
				msg.Player.AccountID == accountID
		})

		playerMessage := kafka.ParseMessage[kafka.PlayerMessage](t, message)
		testData.playerMessage = &playerMessage

		t.Require().Equal(s.config.Node.ProjectID, playerMessage.Player.NodeID)
		t.Require().Equal(s.config.Node.GroupID, playerMessage.Player.ProjectGroupID)
		t.Require().Equal(s.config.Node.DefaultCountry, playerMessage.Player.Country)
		t.Require().Equal(s.config.Node.DefaultCurrency, playerMessage.Player.Currency)

		sCtx.WithAttachments(allure.NewAttachment("Kafka Message", allure.JSON, utils.CreatePrettyJSON(message.Value)))

		t.Require().NotEmpty(playerMessage.Player.ExternalID)
	})

	t.WithNewStep("Проверка значения в Redis.", func(sCtx provider.StepCtx) {
		key := testData.playerMessage.Player.ExternalID
		value := s.redisClient.GetWithRetry(t, key)

		sCtx.WithAttachments(allure.NewAttachment("Redis Value", allure.JSON, utils.CreatePrettyJSON(value)))
	})

	t.WithNewStep("Проверка создания кошелька в NATS.", func(sCtx provider.StepCtx) {
		playerUUID := testData.playerMessage.Player.ExternalID
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, playerUUID)

		s.natsClient.Subscribe(t, subject)

		message := nats.FindMessageByFilter[nats.WalletPayload](s.natsClient, t, func(wallet nats.WalletPayload) bool {
			return wallet.WalletType == nats.TypeReal &&
				wallet.WalletStatus == nats.StatusEnabled &&
				wallet.IsBasic
		})

		walletData := nats.ParseMessage[nats.WalletPayload](t, message)

		sCtx.WithAttachments(allure.NewAttachment("NATS Wallet Message", allure.JSON, utils.CreatePrettyJSON(walletData)))
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
