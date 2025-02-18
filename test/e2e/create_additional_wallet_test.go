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
	"CB_auto/test/e2e/constants"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type CreateWalletSuite struct {
	suite.Suite
	client        *client.Client
	config        *config.Config
	publicService publicAPI.PublicAPI
	natsClient    *nats.NatsClient
	redisClient   *redis.RedisClient
	kafka         *kafka.Kafka
	walletDB      *database.Connector
}

func (s *CreateWalletSuite) BeforeAll(t provider.T) {
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

func (s *CreateWalletSuite) TestCreateWallet(t provider.T) {
	t.Epic("Wallet")
	t.Feature("Создание дополнительного кошелька")
	t.Tags("Wallet", "Create")
	t.Title("Проверка создания дополнительного кошелька")

	type TestData struct {
		registrationResponse      *models.FastRegistrationResponseBody
		authResponse              *models.TokenCheckResponseBody
		createWalletResponse      *models.CreateWalletResponseBody
		walletCreatedEvent        *nats.NatsMessage[nats.WalletCreatedPayload]
		playerRegistrationMessage *kafka.PlayerMessage
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
		sCtx.WithAttachments(allure.NewAttachment(
			"FastRegistration Response", allure.JSON,
			utils.CreateHttpAttachResponse(createResp),
		))
	})

	t.WithNewStep("Получение сообщения о регистрации из топика player.v1.account.", func(sCtx provider.StepCtx) {
		accountID := testData.registrationResponse.Username

		message := kafka.FindMessageByFilter[kafka.PlayerMessage](s.kafka, t, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == "player.signUpFast" &&
				msg.Player.AccountID == accountID
		})
		playerRegistrationMessage := kafka.ParseMessage[kafka.PlayerMessage](t, message)
		testData.playerRegistrationMessage = &playerRegistrationMessage

		sCtx.WithAttachments(allure.NewAttachment(
			"Kafka Player Message", allure.JSON,
			utils.CreatePrettyJSON(testData.playerRegistrationMessage),
		))
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

		sCtx.WithAttachments(allure.NewAttachment(
			"TokenCheck Request", allure.JSON,
			utils.CreateHttpAttachRequest(authReq),
		))
		sCtx.WithAttachments(allure.NewAttachment(
			"TokenCheck Response", allure.JSON,
			utils.CreateHttpAttachResponse(authResp),
		))
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

	t.WithNewStep("Проверка создания кошелька в NATS.", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, testData.playerRegistrationMessage.Player.ExternalID)
		s.natsClient.Subscribe(t, subject)

		testData.walletCreatedEvent = nats.FindMessageByFilter[nats.WalletCreatedPayload](
			s.natsClient, t, func(wallet nats.WalletCreatedPayload, msgType string) bool {
				return wallet.WalletType == nats.TypeReal &&
					wallet.WalletStatus == nats.StatusEnabled &&
					wallet.Currency == "USD" &&
					!wallet.IsBasic
			},
		)

		sCtx.Assert().NotEmpty(testData.walletCreatedEvent.Payload.WalletUUID, "UUID кошелька в ивенте `wallet_created` не пустой")
		sCtx.Assert().Equal("USD", testData.walletCreatedEvent.Payload.Currency, "Валюта в ивенте `wallet_created` совпадает с ожидаемой")
		sCtx.Assert().Equal(constants.ZeroAmount, testData.walletCreatedEvent.Payload.Balance, "Баланс в ивенте `wallet_created` равен 0")
		sCtx.Assert().False(testData.walletCreatedEvent.Payload.IsDefault, "Кошелёк в ивенте `wallet_created` не помечен как дефолтный")
		sCtx.Assert().False(testData.walletCreatedEvent.Payload.IsBasic, "Кошелёк в ивенте `wallet_created` не помечен как базовый")

		sCtx.WithAttachments(allure.NewAttachment("NATS Wallet Message", allure.JSON, utils.CreatePrettyJSON(testData.walletCreatedEvent.Payload)))
	})

	t.WithNewStep("Проверка значения в Redis.", func(sCtx provider.StepCtx) {
		key := testData.playerRegistrationMessage.Player.ExternalID
		wallets := s.redisClient.GetWithRetry(t, key)

		var foundWallet redis.WalletData
		for _, w := range wallets {
			if w.WalletUUID == testData.walletCreatedEvent.Payload.WalletUUID {
				foundWallet = w
				break
			}
		}

		sCtx.Assert().Equal("USD", foundWallet.Currency, "Валюта в Redis совпадает с валютой из ивента `wallet_created`")
		sCtx.Assert().Equal(int(nats.TypeReal), foundWallet.Type, "Тип кошелька в Redis – реальный")
		sCtx.Assert().Equal(int(nats.StatusEnabled), foundWallet.Status, "Статус кошелька в Redis – включён")

		sCtx.WithAttachments(allure.NewAttachment("Redis Value", allure.JSON, utils.CreatePrettyJSON(wallets)))
	})

	t.WithNewAsyncStep("Проверка получения списка кошельков.", func(sCtx provider.StepCtx) {
		getWalletsReq := &client.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.authResponse.Token),
				"Platform-Locale": "en",
			},
		}
		getWalletsResp := s.publicService.GetWallets(getWalletsReq)

		var foundWallet *models.WalletData
		for _, wallet := range getWalletsResp.Body.Wallets {
			if wallet.ID == testData.walletCreatedEvent.Payload.WalletUUID {
				foundWallet = &wallet
				break
			}
		}

		sCtx.Assert().NotNil(foundWallet, "Кошелёк найден в списке")
		sCtx.Assert().Equal("USD", foundWallet.Currency, "Валюта кошелька совпадает с ожидаемой")
		sCtx.Assert().Equal(constants.ZeroAmount, foundWallet.Balance, "Баланс кошелька равен 0")
		sCtx.Assert().False(foundWallet.Default, "Кошелёк не помечен как \"по умолчанию\"")

		sCtx.WithAttachments(allure.NewAttachment("GetWallets Request", allure.JSON, utils.CreateHttpAttachRequest(getWalletsReq)))
		sCtx.WithAttachments(allure.NewAttachment("GetWallets Response", allure.JSON, utils.CreateHttpAttachResponse(getWalletsResp)))
	})

	t.WithNewAsyncStep("Проверка создания кошелька в БД.", func(sCtx provider.StepCtx) {
		walletRepo := wallet.NewRepository(s.walletDB.DB(), &s.config.MySQL)
		walletFromDatabase := walletRepo.GetWallet(t, map[string]interface{}{"uuid": testData.walletCreatedEvent.Payload.WalletUUID})

		sCtx.Assert().Equal(testData.walletCreatedEvent.Payload.WalletUUID, walletFromDatabase.UUID, "UUID кошелька в БД совпадает с UUID из ивента `wallet_created`")
		sCtx.Assert().Equal("USD", walletFromDatabase.Currency, "Валюта в БД совпадает с валютой из ивента `wallet_created`")
		sCtx.Assert().Equal(constants.ZeroAmount, walletFromDatabase.Balance.String(), "Баланс в БД равен 0")
		sCtx.Assert().False(walletFromDatabase.IsDefault, "Кошелёк не помечен как \"по умолчанию\" в БД")
		sCtx.Assert().False(walletFromDatabase.IsBasic, "Кошелёк не помечен как базовый в БД")

		sCtx.WithAttachments(allure.NewAttachment("Wallet DB Data", allure.JSON, utils.CreatePrettyJSON(walletFromDatabase)))
	})
}

func (s *CreateWalletSuite) AfterAll(t provider.T) {
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

func TestCreateWalletSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(CreateWalletSuite))
}
