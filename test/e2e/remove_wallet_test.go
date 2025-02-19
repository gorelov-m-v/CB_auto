package test

import (
	"fmt"
	"testing"

	client "CB_auto/internal/client"
	publicAPI "CB_auto/internal/client/public"
	"CB_auto/internal/client/public/models"
	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/internal/repository/wallet"
	"CB_auto/internal/transport/kafka"
	"CB_auto/internal/transport/nats"
	"CB_auto/internal/transport/redis"
	"CB_auto/pkg/utils"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type RemoveWalletSuite struct {
	suite.Suite
	client        *client.Client
	config        *config.Config
	publicService publicAPI.PublicAPI
	natsClient    *nats.NatsClient
	redisClient   *redis.RedisClient
	kafka         *kafka.Kafka
	walletDB      *repository.Connector
}

func (s *RemoveWalletSuite) BeforeAll(t provider.T) {
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
		connector, err := repository.OpenConnector(&s.config.MySQL, repository.Wallet)
		if err != nil {
			t.Fatalf("OpenConnector для wallet не удался: %v", err)
		}
		s.walletDB = &connector
	})
}

func (s *RemoveWalletSuite) TestRemoveWallet(t provider.T) {
	t.Epic("Wallet")
	t.Feature("Удаление дополнительного кошелька")
	t.Tags("Wallet", "Remove")
	t.Title("Проверка удаления дополнительного кошелька")

	type TestData struct {
		registrationResponse         *models.FastRegistrationResponseBody
		authResponse                 *models.TokenCheckResponseBody
		createWalletResponse         *models.CreateWalletResponseBody
		playerRegistrationMessage    *kafka.PlayerMessage
		additionalWalletCreatedEvent *nats.NatsMessage[nats.WalletCreatedPayload]
		walletDisabledEvent          *nats.NatsMessage[nats.WalletDisabledPayload]
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

	t.WithNewStep("Получение сообщения о регистрации игрока из Kafka.", func(sCtx provider.StepCtx) {
		message := kafka.FindMessageByFilter[kafka.PlayerMessage](s.kafka, t, func(msg kafka.PlayerMessage) bool {
			return msg.Player.AccountID == testData.registrationResponse.Username
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

	t.WithNewStep("Получение сообщения о создании дополнительного кошелька из NATS.", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, testData.playerRegistrationMessage.Player.ExternalID)
		s.natsClient.Subscribe(t, subject)

		testData.additionalWalletCreatedEvent = nats.FindMessageByFilter[nats.WalletCreatedPayload](
			s.natsClient, t, func(wallet nats.WalletCreatedPayload, msgType string) bool {
				return wallet.WalletType == nats.TypeReal &&
					wallet.WalletStatus == nats.StatusEnabled &&
					!wallet.IsBasic &&
					wallet.Currency == "USD"
			},
		)

		sCtx.Assert().NotEmpty(testData.additionalWalletCreatedEvent.Payload.WalletUUID, "UUID кошелька в ивенте `wallet_created` не пустой")
		sCtx.Assert().Equal(testData.playerRegistrationMessage.Player.ExternalID, testData.additionalWalletCreatedEvent.Payload.PlayerUUID, "UUID игрока в ивенте `wallet_created` совпадает с ожидаемым")
		sCtx.Assert().Equal("USD", testData.additionalWalletCreatedEvent.Payload.Currency, "Валюта в ивенте `wallet_created` совпадает с ожидаемой")
		sCtx.Assert().Equal(nats.TypeReal, testData.additionalWalletCreatedEvent.Payload.WalletType, "Тип кошелька в ивенте `wallet_created` – реальный")
		sCtx.Assert().Equal(nats.StatusEnabled, testData.additionalWalletCreatedEvent.Payload.WalletStatus, "Статус кошелька в ивенте `wallet_created` – включён")
		sCtx.Assert().False(testData.additionalWalletCreatedEvent.Payload.IsBasic, "Кошелёк не помечен как базовый")

		sCtx.WithAttachments(allure.NewAttachment("NATS Additional Wallet Message", allure.JSON, utils.CreatePrettyJSON(testData.additionalWalletCreatedEvent.Payload)))
	})

	t.WithNewStep("Удаление дополнительного кошелька.", func(sCtx provider.StepCtx) {
		removeReq := &client.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.authResponse.Token),
				"Platform-Locale": "en",
			},
			QueryParams: map[string]string{
				"currency": "USD",
			},
		}
		removeResp := s.publicService.RemoveWallet(removeReq)

		sCtx.Assert().Equal(200, removeResp.StatusCode, "Статус код ответа равен 200")

		sCtx.WithAttachments(allure.NewAttachment("RemoveWallet Request", allure.JSON, utils.CreateHttpAttachRequest(removeReq)))
		sCtx.WithAttachments(allure.NewAttachment("RemoveWallet Response", allure.JSON, utils.CreateHttpAttachResponse(removeResp)))
	})

	t.WithNewStep("Проверка события отключения кошелька в NATS.", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.%s",
			s.config.Nats.StreamPrefix,
			testData.playerRegistrationMessage.Player.ExternalID,
			testData.additionalWalletCreatedEvent.Payload.WalletUUID)
		s.natsClient.Subscribe(t, subject)

		testData.walletDisabledEvent = nats.FindMessageByFilter[nats.WalletDisabledPayload](
			s.natsClient, t, func(msg nats.WalletDisabledPayload, msgType string) bool {
				return msgType == "wallet_disabled"
			},
		)

		sCtx.Assert().NotNil(testData.walletDisabledEvent)

		sCtx.WithAttachments(allure.NewAttachment("WalletDisabled Event", allure.JSON, utils.CreatePrettyJSON(testData.walletDisabledEvent)))
	})

	t.WithNewAsyncStep("Проверка отсутствия кошелька в списке.", func(sCtx provider.StepCtx) {
		getWalletsReq := &client.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.authResponse.Token),
				"Platform-Locale": "en",
			},
		}
		getWalletsResp := s.publicService.GetWallets(getWalletsReq)

		var foundWallet *models.WalletData
		for _, wallet := range getWalletsResp.Body.Wallets {
			if wallet.Currency == "USD" {
				foundWallet = &wallet
				break
			}
		}

		sCtx.Assert().Nil(foundWallet, "Удаленный кошелек отсутствует в списке")

		sCtx.WithAttachments(allure.NewAttachment("GetWallets Request", allure.JSON, utils.CreateHttpAttachRequest(getWalletsReq)))
		sCtx.WithAttachments(allure.NewAttachment("GetWallets Response", allure.JSON, utils.CreateHttpAttachResponse(getWalletsResp)))
	})

	t.WithNewAsyncStep("Проверка отключения кошелька в Redis.", func(sCtx provider.StepCtx) {
		key := testData.playerRegistrationMessage.Player.ExternalID
		wallets := s.redisClient.GetWithRetry(t, key)

		var disabledWallet redis.WalletData
		for _, w := range wallets {
			if w.WalletUUID == testData.additionalWalletCreatedEvent.Payload.WalletUUID {
				disabledWallet = w
				break
			}
		}

		sCtx.Assert().Equal(int(nats.StatusDisabled), disabledWallet.Status, "Статус кошелька в Redis – отключен")

		sCtx.WithAttachments(allure.NewAttachment("Redis Value", allure.JSON, utils.CreatePrettyJSON(wallets)))
	})

	t.WithNewAsyncStep("Проверка отключения кошелька в БД.", func(sCtx provider.StepCtx) {
		walletRepo := wallet.NewRepository(s.walletDB.DB(), &s.config.MySQL)
		walletFromDatabase := walletRepo.GetWallet(t, map[string]interface{}{
			"uuid":          testData.additionalWalletCreatedEvent.Payload.WalletUUID,
			"wallet_status": int(nats.StatusDisabled),
		})

		sCtx.Assert().NotNil(walletFromDatabase, "Кошелек найден в БД со статусом отключен")

		sCtx.WithAttachments(allure.NewAttachment("Wallet DB Data", allure.JSON, utils.CreatePrettyJSON(walletFromDatabase)))
	})
}

func (s *RemoveWalletSuite) AfterAll(t provider.T) {
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

func TestRemoveWalletSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(RemoveWalletSuite))
}
