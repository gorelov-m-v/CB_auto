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

	_ "github.com/go-sql-driver/mysql"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type RemoveWalletSuite struct {
	suite.Suite
	config        *config.Config
	publicService publicAPI.PublicAPI
	natsClient    *nats.NatsClient
	redisClient   *redis.RedisClient
	kafka         *kafka.Kafka
	walletDB      *repository.Connector
	walletRepo    *wallet.Repository
}

func (s *RemoveWalletSuite) BeforeAll(t provider.T) {
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
		s.redisClient = redis.NewRedisClient(t, &s.config.Redis)
	})

	t.WithNewStep("Инициализация Kafka.", func(sCtx provider.StepCtx) {
		s.kafka = kafka.GetInstance(t, s.config)
	})

	t.WithNewStep("Соединение с базой данных wallet.", func(sCtx provider.StepCtx) {
		connector := repository.OpenConnector(t, &s.config.MySQL, repository.Wallet)
		s.walletDB = &connector
		s.walletRepo = wallet.NewRepository(s.walletDB.DB(), &s.config.MySQL)
	})
}

func (s *RemoveWalletSuite) TestRemoveWallet(t provider.T) {
	t.Epic("Wallet")
	t.Feature("Удаление дополнительного кошелька")
	t.Tags("Wallet", "Remove")
	t.Title("Проверка удаления дополнительного кошелька")

	type TestData struct {
		registrationResponse         *clientTypes.Response[models.FastRegistrationResponseBody]
		authResponse                 *clientTypes.Response[models.TokenCheckResponseBody]
		createWalletResponse         *clientTypes.Response[models.CreateWalletResponseBody]
		registrationMessage          kafka.PlayerMessage
		additionalWalletCreatedEvent *nats.NatsMessage[nats.WalletCreatedPayload]
		walletDisabledEvent          *nats.NatsMessage[nats.WalletDisabledPayload]
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

	t.WithNewStep("Получение сообщения о регистрации игрока из Kafka.", func(sCtx provider.StepCtx) {
		testData.registrationMessage = kafka.FindMessageByFilter(sCtx, s.kafka, func(msg kafka.PlayerMessage) bool {
			return msg.Player.AccountID == testData.registrationResponse.Body.Username
		})

		sCtx.Require().NotEmpty(testData.registrationMessage.Player.ExternalID, "External ID игрока в регистрации не пустой")
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

	t.WithNewStep("Получение сообщения о создании дополнительного кошелька из NATS.", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, testData.registrationMessage.Player.ExternalID)

		testData.additionalWalletCreatedEvent = nats.FindMessageInStream(
			sCtx, s.natsClient, subject, func(wallet nats.WalletCreatedPayload, msgType string) bool {
				return wallet.WalletType == nats.TypeReal &&
					wallet.WalletStatus == nats.StatusEnabled &&
					!wallet.IsBasic &&
					wallet.Currency == "USD"
			})

		sCtx.Assert().NotEmpty(testData.additionalWalletCreatedEvent.Payload.WalletUUID, "UUID кошелька в ивенте `wallet_created` не пустой")
		sCtx.Assert().Equal(testData.registrationMessage.Player.ExternalID, testData.additionalWalletCreatedEvent.Payload.PlayerUUID, "UUID игрока в ивенте `wallet_created` совпадает с ожидаемым")
		sCtx.Assert().Equal("USD", testData.additionalWalletCreatedEvent.Payload.Currency, "Валюта в ивенте `wallet_created` совпадает с ожидаемой")
		sCtx.Assert().Equal(nats.TypeReal, testData.additionalWalletCreatedEvent.Payload.WalletType, "Тип кошелька в ивенте `wallet_created` – реальный")
		sCtx.Assert().Equal(nats.StatusEnabled, testData.additionalWalletCreatedEvent.Payload.WalletStatus, "Статус кошелька в ивенте `wallet_created` – включён")
		sCtx.Assert().False(testData.additionalWalletCreatedEvent.Payload.IsBasic, "Кошелёк не помечен как базовый")
	})

	t.WithNewStep("Удаление дополнительного кошелька.", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.authResponse.Body.Token),
				"Platform-Locale": "en",
			},
			QueryParams: map[string]string{
				"currency": "USD",
			},
		}
		removeResp := s.publicService.RemoveWallet(sCtx, req)

		sCtx.Assert().Equal(http.StatusOK, removeResp.StatusCode, "Статус код ответа равен 200")
	})

	t.WithNewStep("Проверка события отключения кошелька в NATS.", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.%s",
			s.config.Nats.StreamPrefix,
			testData.registrationMessage.Player.ExternalID,
			testData.additionalWalletCreatedEvent.Payload.WalletUUID)

		testData.walletDisabledEvent = nats.FindMessageInStream(
			sCtx, s.natsClient, subject, func(msg nats.WalletDisabledPayload, msgType string) bool {
				return msgType == "wallet_disabled"
			})

		sCtx.Assert().NotNil(testData.walletDisabledEvent)
	})

	t.WithNewAsyncStep("Проверка отсутствия кошелька в списке.", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.authResponse.Body.Token),
				"Platform-Locale": "en",
			},
		}
		resp := s.publicService.GetWallets(sCtx, req)

		var foundWallet *models.WalletData
		for _, wallet := range resp.Body.Wallets {
			if wallet.Currency == "USD" {
				foundWallet = &wallet
				break
			}
		}

		sCtx.Assert().Nil(foundWallet, "Удаленный кошелек отсутствует в списке")
	})

	t.WithNewAsyncStep("Проверка отключения кошелька в Redis.", func(sCtx provider.StepCtx) {
		key := testData.registrationMessage.Player.ExternalID
		wallets := s.redisClient.GetWithRetry(sCtx, key)

		var disabledWallet redis.WalletData
		for _, w := range wallets {
			if w.WalletUUID == testData.additionalWalletCreatedEvent.Payload.WalletUUID {
				disabledWallet = w
				break
			}
		}

		sCtx.Assert().Equal(int(nats.StatusDisabled), disabledWallet.Status, "Статус кошелька в Redis – отключен")
	})

	t.WithNewAsyncStep("Проверка отключения кошелька в БД.", func(sCtx provider.StepCtx) {
		walletFromDatabase := s.walletRepo.GetWallet(sCtx, map[string]interface{}{
			"uuid":          testData.additionalWalletCreatedEvent.Payload.WalletUUID,
			"wallet_status": int(nats.StatusDisabled),
		})

		sCtx.Assert().NotNil(walletFromDatabase, "Кошелек найден в БД со статусом отключен")
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
