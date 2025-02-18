package test

import (
	"fmt"
	"testing"

	client "CB_auto/internal/client"
	publicAPI "CB_auto/internal/client/public"
	"CB_auto/internal/client/public/models"
	"CB_auto/internal/config"
	"CB_auto/internal/database"
	"CB_auto/internal/database/wallet"
	"CB_auto/internal/transport/kafka"
	"CB_auto/internal/transport/nats"
	"CB_auto/internal/transport/redis"
	"CB_auto/pkg/utils"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type SwitchWalletSuite struct {
	suite.Suite
	client        *client.Client
	config        *config.Config
	publicService publicAPI.PublicAPI
	natsClient    *nats.NatsClient
	redisClient   *redis.RedisClient
	kafka         *kafka.Kafka
	walletDB      *database.Connector
}

func (s *SwitchWalletSuite) BeforeAll(t provider.T) {
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
		natsClient, err := nats.NewClient(&s.config.Nats)
		if err != nil {
			t.Fatalf("NewClient NATS не удался: %v", err)
		}
		s.natsClient = natsClient
	})

	t.WithNewStep("Инициализация Redis клиента.", func(sCtx provider.StepCtx) {
		redisClient, err := redis.NewRedisClient(&s.config.Redis)
		if err != nil {
			t.Fatalf("Redis init failed: %v", err)
		}
		s.redisClient = redisClient
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

	t.WithNewStep("Соединение с базой данных wallet.", func(sCtx provider.StepCtx) {
		connector, err := database.OpenConnector(&s.config.MySQL, database.Wallet)
		if err != nil {
			t.Fatalf("OpenConnector для wallet не удался: %v", err)
		}
		s.walletDB = &connector
	})
}

func (s *SwitchWalletSuite) TestSwitchWallet(t provider.T) {
	t.Epic("Wallet")
	t.Feature("Переключение дефолтного кошелька")
	t.Tags("Wallet", "Switch")
	t.Title("Проверка переключения дефолтного кошелька")

	type TestData struct {
		registrationResponse         *models.FastRegistrationResponseBody
		authResponse                 *models.TokenCheckResponseBody
		createWalletResponse         *models.CreateWalletResponseBody
		mainWalletCreatedEvent       *nats.NatsMessage[nats.WalletCreatedPayload]
		additionalWalletCreatedEvent *nats.NatsMessage[nats.WalletCreatedPayload]
		setDefaultStartedEvent       *nats.NatsMessage[nats.SetDefaultStartedPayload]
		defaultUnsettedEvent         *nats.NatsMessage[nats.DefaultUnsettedPayload]
		setDefaultCommittedEvent     *nats.NatsMessage[nats.DefaultSettedPayload]
		playerRegistrationMessage    *kafka.PlayerMessage
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

		sCtx.Assert().NotEmpty(createResp.Body.Username, "Username в ответе регистрации не пустой")
		sCtx.Assert().NotEmpty(createResp.Body.Password, "Password в ответе регистрации не пустой")

		sCtx.WithAttachments(allure.NewAttachment("FastRegistration Request", allure.JSON, utils.CreateHttpAttachRequest(createReq)))
		sCtx.WithAttachments(allure.NewAttachment("FastRegistration Response", allure.JSON, utils.CreateHttpAttachResponse(createResp)))
	})

	t.WithNewStep("Получение сообщения о регистрации из топика player.v1.account.", func(sCtx provider.StepCtx) {
		accountID := testData.registrationResponse.Username

		message := kafka.FindMessageByFilter[kafka.PlayerMessage](s.kafka, t, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == "player.signUpFast" &&
				msg.Player.AccountID == accountID
		})
		playerRegistrationMessage := kafka.ParseMessage[kafka.PlayerMessage](t, message)
		testData.playerRegistrationMessage = &playerRegistrationMessage

		sCtx.WithAttachments(allure.NewAttachment("Kafka Player Message", allure.JSON, utils.CreatePrettyJSON(testData.playerRegistrationMessage)))
	})

	t.WithNewStep("Получение токена авторизации.", func(sCtx provider.StepCtx) {
		authReq := &client.Request[models.TokenCheckRequestBody]{
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: &models.TokenCheckRequestBody{
				Username: testData.registrationResponse.Username,
				Password: testData.registrationResponse.Password,
			},
		}
		authResp := s.publicService.TokenCheck(authReq)
		testData.authResponse = &authResp.Body

		sCtx.Assert().NotEmpty(authResp.Body.Token, "Токен авторизации не пустой")
		sCtx.Assert().NotEmpty(authResp.Body.RefreshToken, "Refresh токен не пустой")

		sCtx.WithAttachments(allure.NewAttachment("TokenCheck Request", allure.JSON, utils.CreateHttpAttachRequest(authReq)))
		sCtx.WithAttachments(allure.NewAttachment("TokenCheck Response", allure.JSON, utils.CreateHttpAttachResponse(authResp)))
	})

	t.WithNewStep("Получение сообщения о создании основного кошелька в NATS.", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, testData.playerRegistrationMessage.Player.ExternalID)
		s.natsClient.Subscribe(t, subject)

		testData.mainWalletCreatedEvent = nats.FindMessageByFilter[nats.WalletCreatedPayload](
			s.natsClient, t, func(wallet nats.WalletCreatedPayload, msgType string) bool {
				return wallet.WalletType == nats.TypeReal &&
					wallet.WalletStatus == nats.StatusEnabled &&
					wallet.IsBasic
			},
		)

		sCtx.WithAttachments(allure.NewAttachment("NATS Wallet Message", allure.JSON, utils.CreatePrettyJSON(testData.mainWalletCreatedEvent.Payload)))
	})

	t.WithNewStep("Создание дополнительного кошелька.", func(sCtx provider.StepCtx) {
		createReq := &client.Request[models.CreateWalletRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.authResponse.Token),
				"Platform-Locale": "en",
				"Content-Type":    "application/json",
			},
			Body: &models.CreateWalletRequestBody{
				Currency: "USD",
			},
		}
		createResp := s.publicService.CreateWallet(createReq)
		testData.createWalletResponse = &createResp.Body

		sCtx.Assert().Equal(201, createResp.StatusCode, "Статус код ответа равен 201")

		sCtx.WithAttachments(allure.NewAttachment("CreateWallet Request", allure.JSON, utils.CreateHttpAttachRequest(createReq)))
		sCtx.WithAttachments(allure.NewAttachment("CreateWallet Response", allure.JSON, utils.CreateHttpAttachResponse(createResp)))
	})

	t.WithNewStep("Получение сообщения о создании дополнительного кошелька в NATS.", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, testData.playerRegistrationMessage.Player.ExternalID)
		s.natsClient.Subscribe(t, subject)

		testData.additionalWalletCreatedEvent = nats.FindMessageByFilter[nats.WalletCreatedPayload](
			s.natsClient, t, func(wallet nats.WalletCreatedPayload, msgType string) bool {
				return wallet.WalletType == nats.TypeReal &&
					wallet.WalletStatus == nats.StatusEnabled &&
					wallet.Currency == "USD" &&
					!wallet.IsBasic
			},
		)

		sCtx.WithAttachments(allure.NewAttachment("NATS Wallet Message", allure.JSON, utils.CreatePrettyJSON(testData.additionalWalletCreatedEvent.Payload)))
	})

	t.WithNewStep("Переключение дефолтного кошелька.", func(sCtx provider.StepCtx) {
		switchReq := &client.Request[models.SwitchWalletRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.authResponse.Token),
				"Platform-Locale": "en",
				"Content-Type":    "application/json",
			},
			Body: &models.SwitchWalletRequestBody{
				Currency: "USD",
			},
		}
		switchResp := s.publicService.SwitchWallet(switchReq)

		sCtx.Assert().Equal(204, switchResp.StatusCode, "Статус код ответа равен 204")

		sCtx.WithAttachments(allure.NewAttachment("SwitchWallet Request", allure.JSON, utils.CreateHttpAttachRequest(switchReq)))
		sCtx.WithAttachments(allure.NewAttachment("SwitchWallet Response", allure.JSON, utils.CreateHttpAttachResponse(switchResp)))
	})

	t.WithNewStep("Проверка событий смены дефолтного кошелька в NATS.", func(sCtx provider.StepCtx) {
		playerSubject := fmt.Sprintf("%s.wallet.*.%s.*",
			s.config.Nats.StreamPrefix,
			testData.playerRegistrationMessage.Player.ExternalID,
		)
		s.natsClient.Subscribe(t, playerSubject)

		testData.setDefaultStartedEvent = nats.FindMessageByFilter[nats.SetDefaultStartedPayload](
			s.natsClient, t, func(msg nats.SetDefaultStartedPayload, msgType string) bool {
				return msgType == "set_default_started"
			},
		)

		testData.defaultUnsettedEvent = nats.FindMessageByFilter[nats.DefaultUnsettedPayload](
			s.natsClient, t, func(msg nats.DefaultUnsettedPayload, msgType string) bool {
				return msgType == "default_unsetted"
			},
		)

		testData.setDefaultCommittedEvent = nats.FindMessageByFilter[nats.DefaultSettedPayload](
			s.natsClient, t, func(msg nats.DefaultSettedPayload, msgType string) bool {
				return msgType == "set_default_committed"
			},
		)

		sCtx.Assert().Less(testData.setDefaultStartedEvent.Seq, testData.defaultUnsettedEvent.Seq, "set_default_started произошло раньше default_unsetted")
		sCtx.Assert().Less(testData.defaultUnsettedEvent.Seq, testData.setDefaultCommittedEvent.Seq, "default_unsetted произошло раньше set_default_committed")
		sCtx.Assert().Equal("set_default_started", testData.setDefaultStartedEvent.Type, "Тип события set_default_started")
		sCtx.Assert().Equal("default_unsetted", testData.defaultUnsettedEvent.Type, "Тип события default_unsetted")
		sCtx.Assert().Equal("set_default_committed", testData.setDefaultCommittedEvent.Type, "Тип события set_default_committed")

		sCtx.WithAttachments(allure.NewAttachment("SetDefaultStarted Event", allure.JSON, utils.CreatePrettyJSON(testData.setDefaultStartedEvent)))
		sCtx.WithAttachments(allure.NewAttachment("DefaultUnsetted Event", allure.JSON, utils.CreatePrettyJSON(testData.defaultUnsettedEvent)))
		sCtx.WithAttachments(allure.NewAttachment("SetDefaultCommitted Event", allure.JSON, utils.CreatePrettyJSON(testData.setDefaultCommittedEvent)))
	})

	t.WithNewAsyncStep("Смены дефолтного кошелька в БД.", func(sCtx provider.StepCtx) {
		walletRepo := wallet.NewRepository(s.walletDB.DB, &s.config.MySQL)

		oldDefaultWallet := walletRepo.GetWallet(t, map[string]interface{}{"uuid": testData.mainWalletCreatedEvent.Payload.WalletUUID})
		newDefaultWallet := walletRepo.GetWallet(t, map[string]interface{}{"uuid": testData.additionalWalletCreatedEvent.Payload.WalletUUID})

		sCtx.Assert().True(newDefaultWallet.IsDefault, "Новый кошелёк помечен как дефолтный в БД")
		sCtx.Assert().False(oldDefaultWallet.IsDefault, "Старый кошелёк больше не помечен как дефолтный в БД")

		sCtx.WithAttachments(allure.NewAttachment("Old Wallet DB Data", allure.JSON, utils.CreatePrettyJSON(oldDefaultWallet)))
		sCtx.WithAttachments(allure.NewAttachment("New Wallet DB Data", allure.JSON, utils.CreatePrettyJSON(newDefaultWallet)))
	})
}

func (s *SwitchWalletSuite) AfterAll(t provider.T) {
	if s.natsClient != nil {
		s.natsClient.Close()
	}
	if s.redisClient != nil {
		s.redisClient.Close()
	}
	if s.walletDB != nil {
		if err := s.walletDB.Close(); err != nil {
			t.Errorf("Ошибка при закрытии соединения с wallet DB: %v", err)
		}
	}
}

func TestSwitchWalletSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(SwitchWalletSuite))
}
