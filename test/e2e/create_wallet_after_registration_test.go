package test

import (
	"context"
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

type FastRegistrationSuite struct {
	suite.Suite
	client        *client.Client
	config        *config.Config
	publicService publicAPI.PublicAPI
	natsClient    *nats.NatsClient
	redisClient   *redis.RedisClient
	kafka         *kafka.Kafka
	walletDB      *database.Connector
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
		connector, err := database.OpenConnector(context.Background(), database.Config{
			DriverName:      s.config.MySQL.DriverName,
			DSN:             s.config.MySQL.DSNWallet,
			PingTimeout:     s.config.MySQL.PingTimeout,
			ConnMaxLifetime: s.config.MySQL.ConnMaxLifetime,
			ConnMaxIdleTime: s.config.MySQL.ConnMaxIdleTime,
			MaxOpenConns:    s.config.MySQL.MaxOpenConns,
			MaxIdleConns:    s.config.MySQL.MaxIdleConns,
		})
		if err != nil {
			t.Fatalf("OpenConnector для wallet не удался: %v", err)
		}
		s.walletDB = &connector
	})
}

func (s *FastRegistrationSuite) TestFastRegistration(t provider.T) {
	t.Epic("Wallet")
	t.Feature("Создание кошелька при регистрации пользователя")
	t.Tags("Wallet", "Registration")
	t.Title("Проверка создания кошелька при регистрации пользователя")

	type TestData struct {
		authResponse              *models.TokenCheckResponseBody
		registrationResponse      *models.FastRegistrationResponseBody
		playerRegistrationMessage *kafka.PlayerMessage
		walletCreatedEvent        *nats.NatsMessage[nats.WalletCreatedPayload]
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

	t.WithNewStep("Получение сообщения о регистрации из топика player.v1.account.", func(sCtx provider.StepCtx) {
		message := kafka.FindMessageByFilter(s.kafka, t, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == "player.signUpFast" &&
				msg.Player.AccountID == testData.registrationResponse.Username
		})
		playerRegistrationMessage := kafka.ParseMessage[kafka.PlayerMessage](t, message)
		testData.playerRegistrationMessage = &playerRegistrationMessage

		sCtx.Assert().NotEmpty(playerRegistrationMessage.Player.ExternalID, "External ID игрока в регистрации не пустой")

		sCtx.WithAttachments(allure.NewAttachment("Kafka Message", allure.JSON, utils.CreatePrettyJSON(playerRegistrationMessage)))
	})

	t.WithNewStep("Проверка создания кошелька в NATS.", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*",
			s.config.Nats.StreamPrefix,
			testData.playerRegistrationMessage.Player.ExternalID,
		)
		s.natsClient.Subscribe(t, subject)

		testData.walletCreatedEvent = nats.FindMessageByFilter[nats.WalletCreatedPayload](
			s.natsClient, t, func(wallet nats.WalletCreatedPayload, msgType string) bool {
				return wallet.WalletType == nats.TypeReal &&
					wallet.WalletStatus == nats.StatusEnabled &&
					wallet.IsBasic
			},
		)

		sCtx.Assert().NotEmpty(testData.walletCreatedEvent.Payload.WalletUUID, "UUID кошелька в ивенте `wallet_created` не пустой")
		sCtx.Assert().Equal(testData.playerRegistrationMessage.Player.ExternalID, testData.walletCreatedEvent.Payload.PlayerUUID, "UUID игрока в ивенте `wallet_created` совпадает с ожидаемым")
		sCtx.Assert().Equal(constants.EmptyBonusUUID, testData.walletCreatedEvent.Payload.PlayerBonusUUID, "Bonus UUID в ивенте `wallet_created` равен пустому значению")
		sCtx.Assert().Equal(nats.TypeReal, testData.walletCreatedEvent.Payload.WalletType, "Тип кошелька в ивенте `wallet_created` – реальный")
		sCtx.Assert().Equal(nats.StatusEnabled, testData.walletCreatedEvent.Payload.WalletStatus, "Статус кошелька в ивенте `wallet_created` – включён")
		sCtx.Assert().Equal(s.config.Node.DefaultCurrency, testData.walletCreatedEvent.Payload.Currency, "Валюта в ивенте `wallet_created` совпадает с ожидаемой")
		sCtx.Assert().Equal(constants.ZeroAmount, testData.walletCreatedEvent.Payload.Balance, "Баланс в ивенте `wallet_created` равен 0")
		sCtx.Assert().True(testData.walletCreatedEvent.Payload.IsDefault, "Кошелёк в ивенте `wallet_created` помечен как \"по умолчанию\"")
		sCtx.Assert().True(testData.walletCreatedEvent.Payload.IsBasic, "Кошелёк в ивенте `wallet_created` помечен как базовый")
		sCtx.Assert().NotEmpty(testData.walletCreatedEvent.Payload.CreatedAt, "Дата создания в ивенте `wallet_created` не пустая")
		sCtx.Assert().NotEmpty(testData.walletCreatedEvent.Payload.UpdatedAt, "Дата обновления в ивенте `wallet_created` не пустая")

		sCtx.WithAttachments(allure.NewAttachment("NATS Wallet Message", allure.JSON, utils.CreatePrettyJSON(testData.walletCreatedEvent.Payload)))
	})

	t.WithNewAsyncStep("Проверка значения в Redis.", func(sCtx provider.StepCtx) {
		key := testData.playerRegistrationMessage.Player.ExternalID
		wallets := s.redisClient.GetWithRetry(t, key)
		var wallet redis.WalletData
		for _, w := range wallets {
			wallet = w
			break
		}

		sCtx.Assert().Equal(int(nats.TypeReal), wallet.Type, "Тип кошелька в Redis – реальный")
		sCtx.Assert().Equal(int(nats.StatusEnabled), wallet.Status, "Статус кошелька в Redis – включён")
		sCtx.Assert().Equal(testData.walletCreatedEvent.Payload.WalletUUID, wallet.WalletUUID, "UUID кошелька в Redis совпадает с UUID из ивента `wallet_created`")

		sCtx.WithAttachments(allure.NewAttachment(
			"Redis Value", allure.JSON,
			utils.CreatePrettyJSON(wallets),
		))
	})

	t.WithNewAsyncStep("Проверка создания кошелька в БД.", func(sCtx provider.StepCtx) {
		walletRepo := wallet.NewRepository(s.walletDB.DB)
		walletFromDatabase := walletRepo.GetWallet(t, map[string]interface{}{"uuid": testData.walletCreatedEvent.Payload.WalletUUID})

		sCtx.Assert().Equal(testData.walletCreatedEvent.Payload.WalletUUID, walletFromDatabase.UUID, "UUID кошелька в БД совпадает с UUID из ивента `wallet_created`")
		sCtx.Assert().Equal(testData.walletCreatedEvent.Payload.PlayerUUID, walletFromDatabase.PlayerUUID, "UUID игрока в БД совпадает с UUID из ивента `wallet_created`")
		sCtx.Assert().Equal(testData.walletCreatedEvent.Payload.Currency, walletFromDatabase.Currency, "Валюта в БД совпадает с валютой из ивента `wallet_created`")
		sCtx.Assert().Equal(int(nats.StatusEnabled), walletFromDatabase.WalletStatus, "Статус кошелька в БД – включён")
		sCtx.Assert().Equal(constants.ZeroAmount, walletFromDatabase.Balance.String(), "Баланс в БД равен 0")
		sCtx.Assert().NotZero(walletFromDatabase.CreatedAt, "Дата создания в БД не равна 0")
		sCtx.Assert().NotZero(walletFromDatabase.UpdatedAt.Int64, "Дата обновления в БД должна быть не нулевой")
		sCtx.Assert().True(walletFromDatabase.IsDefault, "Кошелёк помечен как \"по умолчанию\" в БД")
		sCtx.Assert().True(walletFromDatabase.IsBasic, "Кошелёк помечен как базовый в БД")
		sCtx.Assert().False(walletFromDatabase.IsBlocked, "Кошелёк не заблокирован в БД")
		sCtx.Assert().Equal(int(nats.TypeReal), walletFromDatabase.WalletType, "Тип кошелька в БД – реальный")
		sCtx.Assert().Equal(int(testData.walletCreatedEvent.Sequence), walletFromDatabase.Seq, "Номер последовательности в БД совпадает с номером из ивента")
		sCtx.Assert().True(walletFromDatabase.IsGamblingActive, "Гэмблинг активен в БД")
		sCtx.Assert().True(walletFromDatabase.IsBettingActive, "Беттинг активен в БД")
		sCtx.Assert().Equal(constants.ZeroAmount, walletFromDatabase.DepositAmount.String(), "Сумма депозитов в БД равна 0")
		sCtx.Assert().Equal(constants.ZeroAmount, walletFromDatabase.ProfitAmount.String(), "Сумма прибыли в БД равна 0")
		sCtx.Assert().Equal(testData.walletCreatedEvent.Payload.NodeUUID, walletFromDatabase.NodeUUID.String, "Node UUID в БД совпадает с Node UUID из ивента")
		sCtx.Assert().False(walletFromDatabase.IsSumsubVerified, "Кошелёк не верифицирован через Sumsub в БД")
		sCtx.Assert().Equal(constants.ZeroAmount, walletFromDatabase.AvailableWithdrawal.String(), "Доступная сумма для вывода в БД равна 0")
		sCtx.Assert().True(walletFromDatabase.IsKycVerified, "Кошелёк прошёл KYC-верификацию в БД")

		sCtx.WithAttachments(allure.NewAttachment(
			"Wallet DB Data", allure.JSON,
			utils.CreatePrettyJSON(walletFromDatabase),
		))
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
		sCtx.Assert().Equal(testData.walletCreatedEvent.Payload.Currency, foundWallet.Currency, "Валюта кошелька совпадает с ожидаемой")
		sCtx.Assert().Equal(constants.ZeroAmount, foundWallet.Balance, "Баланс кошелька равен 0")
		sCtx.Assert().True(foundWallet.Default, "Кошелёк помечен как \"по умолчанию\"")

		sCtx.WithAttachments(allure.NewAttachment("GetWallets Request", allure.JSON, utils.CreateHttpAttachRequest(getWalletsReq)))
		sCtx.WithAttachments(allure.NewAttachment("GetWallets Response", allure.JSON, utils.CreateHttpAttachResponse(getWalletsResp)))
	})
}

func (s *FastRegistrationSuite) AfterAll(t provider.T) {
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

func TestFastRegistrationSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(FastRegistrationSuite))
}
