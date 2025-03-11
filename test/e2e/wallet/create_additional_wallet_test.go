package test

import (
	"fmt"
	"net/http"
	"testing"

	"CB_auto/internal/client/factory"
	publicAPI "CB_auto/internal/client/public"
	"CB_auto/internal/client/public/models"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/internal/repository/wallet"
	"CB_auto/internal/transport/kafka"
	"CB_auto/internal/transport/nats"
	"CB_auto/internal/transport/redis"
	"CB_auto/test/e2e/constants"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type CreateWalletSuite struct {
	suite.Suite
	config        *config.Config
	publicService publicAPI.PublicAPI
	natsClient    *nats.NatsClient
	redisClient   *redis.RedisClient
	kafka         *kafka.Kafka
	walletDB      *repository.Connector
	walletRepo    *wallet.Repository
}

func (s *CreateWalletSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла.", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация http-клиента и Public API сервиса.", func(sCtx provider.StepCtx) {
		s.publicService = factory.InitClient[publicAPI.PublicAPI](sCtx, s.config, clientTypes.Public)
	})

	t.WithNewStep("Инициализация NATS клиента.", func(sCtx provider.StepCtx) {
		s.natsClient = nats.NewClient(&s.config.Nats)
	})

	t.WithNewStep("Инициализация Redis клиента.", func(sCtx provider.StepCtx) {
		s.redisClient = redis.NewRedisClient(t, &s.config.Redis, redis.PlayerClient)
	})

	t.WithNewStep("Инициализация Kafka.", func(sCtx provider.StepCtx) {
		s.kafka = kafka.GetInstance(t, s.config)
	})

	t.WithNewStep("Соединение с базой данных wallet.", func(sCtx provider.StepCtx) {
		s.walletRepo = wallet.NewRepository(repository.OpenConnector(t, &s.config.MySQL, repository.Wallet).DB(), &s.config.MySQL)
	})
}

func (s *CreateWalletSuite) TestCreateWallet(t provider.T) {
	t.Epic("Wallet")
	t.Feature("Создание дополнительного кошелька")
	t.Tags("Wallet", "Create")
	t.Title("Проверка создания дополнительного кошелька")

	type TestData struct {
		registrationResponse *clientTypes.Response[models.FastRegistrationResponseBody]
		authResponse         *clientTypes.Response[models.TokenCheckResponseBody]
		createWalletResponse *clientTypes.Response[models.CreateWalletResponseBody]
		walletCreatedEvent   *nats.NatsMessage[nats.WalletCreatedPayload]
		registrationMessage  kafka.PlayerMessage
	}
	var testData TestData

	t.WithNewStep("Регистрация пользователя.", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[models.FastRegistrationRequestBody]{
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: &models.FastRegistrationRequestBody{
				Country:  s.config.Node.DefaultCountry,
				Currency: s.config.Node.DefaultCurrency,
			},
		}
		testData.registrationResponse = s.publicService.FastRegistration(sCtx, req)

		sCtx.Assert().NotEmpty(testData.registrationResponse.Body.Username, "Username в ответе регистрации не пустой")
		sCtx.Assert().NotEmpty(testData.registrationResponse.Body.Password, "Password в ответе регистрации не пустой")
	})

	t.WithNewStep("Получение сообщения о регистрации из топика player.v1.account.", func(sCtx provider.StepCtx) {
		testData.registrationMessage = kafka.FindMessageByFilter[kafka.PlayerMessage](sCtx, s.kafka, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == string(kafka.PlayerEventSignUpFast) &&
				msg.Player.AccountID == testData.registrationResponse.Body.Username
		})

		sCtx.Require().NotEmpty(testData.registrationMessage.Player.ID, "ID игрока не пустой")
	})

	t.WithNewStep("Получение токена авторизации.", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[models.TokenCheckRequestBody]{
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: &models.TokenCheckRequestBody{
				Username: testData.registrationResponse.Body.Username,
				Password: testData.registrationResponse.Body.Password,
			},
		}

		testData.authResponse = s.publicService.TokenCheck(sCtx, req)

		sCtx.Assert().NotEmpty(testData.authResponse.Body.Token, "Токен авторизации не пустой")
		sCtx.Assert().NotEmpty(testData.authResponse.Body.RefreshToken, "Refresh токен не пустой")
	})

	t.WithNewStep("Создание дополнительного кошелька.", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[models.CreateWalletRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.authResponse.Body.Token),
				"Platform-Locale": "en",
				"Content-Type":    "application/json",
			},
			Body: &models.CreateWalletRequestBody{
				Currency: "USD",
			},
		}
		testData.createWalletResponse = s.publicService.CreateWallet(sCtx, req)

		sCtx.Assert().Equal(http.StatusCreated, testData.createWalletResponse.StatusCode, "Статус код ответа равен 201")
	})

	t.WithNewStep("Проверка создания кошелька в NATS.", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, testData.registrationMessage.Player.ExternalID)

		testData.walletCreatedEvent = nats.FindMessageInStream(sCtx, s.natsClient, subject, func(wallet nats.WalletCreatedPayload, msgType string) bool {
			return wallet.WalletType == nats.TypeReal &&
				wallet.WalletStatus == nats.StatusEnabled &&
				wallet.Currency == "USD" &&
				!wallet.IsBasic
		})

		sCtx.Assert().NotEmpty(testData.walletCreatedEvent.Payload.WalletUUID, "UUID кошелька в ивенте `wallet_created` не пустой")
		sCtx.Assert().Equal("USD", testData.walletCreatedEvent.Payload.Currency, "Валюта в ивенте `wallet_created` совпадает с ожидаемой")
		sCtx.Assert().Equal(constants.ZeroAmount, testData.walletCreatedEvent.Payload.Balance, "Баланс в ивенте `wallet_created` равен 0")
		sCtx.Assert().False(testData.walletCreatedEvent.Payload.IsDefault, "Кошелёк в ивенте `wallet_created` не помечен как дефолтный")
		sCtx.Assert().False(testData.walletCreatedEvent.Payload.IsBasic, "Кошелёк в ивенте `wallet_created` не помечен как базовый")
	})

	t.WithNewStep("Проверка значения в Redis.", func(sCtx provider.StepCtx) {
		var wallets redis.WalletsMap
		err := s.redisClient.GetWithRetry(sCtx, testData.registrationMessage.Player.ExternalID, &wallets)

		sCtx.Require().NoError(err, "Значение кошелька получено из Redis")

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
	})

	t.WithNewStep("Проверка создания кошелька в БД.", func(sCtx provider.StepCtx) {
		walletFromDatabase := s.walletRepo.GetOneWithRetry(sCtx, map[string]interface{}{"uuid": testData.walletCreatedEvent.Payload.WalletUUID})

		sCtx.Assert().Equal(testData.walletCreatedEvent.Payload.WalletUUID, walletFromDatabase.UUID, "UUID кошелька в БД совпадает с UUID из ивента `wallet_created`")
		sCtx.Assert().Equal("USD", walletFromDatabase.Currency, "Валюта в БД совпадает с валютой из ивента `wallet_created`")
		sCtx.Assert().Equal(constants.ZeroAmount, walletFromDatabase.Balance.String(), "Баланс в БД равен 0")
		sCtx.Assert().False(walletFromDatabase.IsDefault, "Кошелёк не помечен как \"по умолчанию\" в БД")
		sCtx.Assert().False(walletFromDatabase.IsBasic, "Кошелёк не помечен как базовый в БД")
	})

	t.WithNewStep("Проверка получения списка кошельков.", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.authResponse.Body.Token),
				"Platform-Locale": "en",
			},
		}
		resp := s.publicService.GetWallets(sCtx, req)

		var foundWallet *models.WalletData
		for _, wallet := range resp.Body.Wallets {
			if wallet.ID == testData.walletCreatedEvent.Payload.WalletUUID {
				foundWallet = &wallet
				break
			}
		}

		sCtx.Assert().NotNil(foundWallet, "Кошелёк найден в списке")
		sCtx.Assert().Equal("USD", foundWallet.Currency, "Валюта кошелька совпадает с ожидаемой")
		sCtx.Assert().Equal(constants.ZeroAmount, foundWallet.Balance, "Баланс кошелька равен 0")
		sCtx.Assert().False(foundWallet.Default, "Кошелёк не помечен как \"по умолчанию\"")
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
