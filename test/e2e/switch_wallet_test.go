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

type SwitchWalletSuite struct {
	suite.Suite
	config        *config.Config
	publicService publicAPI.PublicAPI
	natsClient    *nats.NatsClient
	redisClient   *redis.RedisClient
	kafka         *kafka.Kafka
	walletDB      *repository.Connector
	walletRepo    *wallet.Repository
}

func (s *SwitchWalletSuite) BeforeAll(t provider.T) {
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
		s.kafka = kafka.NewConsumer(t, s.config, kafka.PlayerTopic)
		s.kafka.StartReading(t)
	})

	t.WithNewStep("Соединение с базой данных wallet.", func(sCtx provider.StepCtx) {
		connector := repository.OpenConnector(t, &s.config.MySQL, repository.Wallet)
		s.walletDB = &connector
		s.walletRepo = wallet.NewRepository(s.walletDB.DB(), &s.config.MySQL)
	})
}

func (s *SwitchWalletSuite) TestSwitchWallet(t provider.T) {
	t.Epic("Wallet")
	t.Feature("Переключение дефолтного кошелька")
	t.Tags("Wallet", "Switch")
	t.Title("Проверка переключения дефолтного кошелька")

	type TestData struct {
		registrationResponse         *clientTypes.Response[models.FastRegistrationResponseBody]
		authResponse                 *clientTypes.Response[models.TokenCheckResponseBody]
		createWalletResponse         *clientTypes.Response[models.CreateWalletResponseBody]
		mainWalletCreatedEvent       *nats.NatsMessage[nats.WalletCreatedPayload]
		additionalWalletCreatedEvent *nats.NatsMessage[nats.WalletCreatedPayload]
		setDefaultStartedEvent       *nats.NatsMessage[nats.SetDefaultStartedPayload]
		defaultUnsettedEvent         *nats.NatsMessage[nats.DefaultUnsettedPayload]
		setDefaultCommittedEvent     *nats.NatsMessage[nats.DefaultSettedPayload]
		playerRegistrationMessage    *kafka.PlayerMessage
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
		message := kafka.FindMessageByFilter(sCtx, s.kafka, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == "player.signUpFast" &&
				msg.Player.AccountID == testData.registrationResponse.Body.Username
		})
		playerRegistrationMessage := kafka.ParseMessage[kafka.PlayerMessage](sCtx, message)
		testData.playerRegistrationMessage = &playerRegistrationMessage
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

	t.WithNewStep("Получение сообщения о создании основного кошелька в NATS.", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, testData.playerRegistrationMessage.Player.ExternalID)
		s.natsClient.SubscribeWithDeliverAll(subject)

		testData.mainWalletCreatedEvent = nats.FindMessageByFilter(
			sCtx, s.natsClient, func(wallet nats.WalletCreatedPayload, msgType string) bool {
				return wallet.WalletType == nats.TypeReal &&
					wallet.WalletStatus == nats.StatusEnabled &&
					wallet.IsBasic
			})
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

	t.WithNewStep("Получение сообщения о создании дополнительного кошелька в NATS.", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, testData.playerRegistrationMessage.Player.ExternalID)
		s.natsClient.SubscribeWithDeliverAll(subject)

		testData.additionalWalletCreatedEvent = nats.FindMessageByFilter(
			sCtx, s.natsClient, func(wallet nats.WalletCreatedPayload, msgType string) bool {
				return wallet.WalletType == nats.TypeReal &&
					wallet.WalletStatus == nats.StatusEnabled &&
					wallet.Currency == "USD" &&
					!wallet.IsBasic
			})
	})

	t.WithNewStep("Переключение дефолтного кошелька.", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[models.SwitchWalletRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.authResponse.Body.Token),
				"Platform-Locale": "en",
				"Content-Type":    "application/json",
			},
			Body: &models.SwitchWalletRequestBody{
				Currency: "USD",
			},
		}
		resp := s.publicService.SwitchWallet(sCtx, req)

		sCtx.Assert().Equal(http.StatusNoContent, resp.StatusCode, "Статус код ответа равен 204")
	})

	t.WithNewStep("Проверка событий смены дефолтного кошелька в NATS.", func(sCtx provider.StepCtx) {
		playerSubject := fmt.Sprintf("%s.wallet.*.%s.*",
			s.config.Nats.StreamPrefix,
			testData.playerRegistrationMessage.Player.ExternalID,
		)
		s.natsClient.SubscribeWithDeliverAll(playerSubject)

		testData.setDefaultStartedEvent = nats.FindMessageByFilter(
			sCtx, s.natsClient, func(msg nats.SetDefaultStartedPayload, msgType string) bool {
				return msgType == "set_default_started"
			})

		testData.defaultUnsettedEvent = nats.FindMessageByFilter(
			sCtx, s.natsClient, func(msg nats.DefaultUnsettedPayload, msgType string) bool {
				return msgType == "default_unsetted"
			})

		testData.setDefaultCommittedEvent = nats.FindMessageByFilter(
			sCtx, s.natsClient, func(msg nats.DefaultSettedPayload, msgType string) bool {
				return msgType == "set_default_committed"
			})

		sCtx.Assert().Less(testData.setDefaultStartedEvent.Seq, testData.defaultUnsettedEvent.Seq, "set_default_started произошло раньше default_unsetted")
		sCtx.Assert().Less(testData.defaultUnsettedEvent.Seq, testData.setDefaultCommittedEvent.Seq, "default_unsetted произошло раньше set_default_committed")
		sCtx.Assert().Equal("set_default_started", testData.setDefaultStartedEvent.Type, "Тип события set_default_started")
		sCtx.Assert().Equal("default_unsetted", testData.defaultUnsettedEvent.Type, "Тип события default_unsetted")
		sCtx.Assert().Equal("set_default_committed", testData.setDefaultCommittedEvent.Type, "Тип события set_default_committed")
	})

	t.WithNewAsyncStep("Смены дефолтного кошелька в БД.", func(sCtx provider.StepCtx) {
		walletRepo := wallet.NewRepository(s.walletDB.DB(), &s.config.MySQL)

		oldDefaultWallet := walletRepo.GetWallet(sCtx, map[string]interface{}{"uuid": testData.mainWalletCreatedEvent.Payload.WalletUUID})
		newDefaultWallet := walletRepo.GetWallet(sCtx, map[string]interface{}{"uuid": testData.additionalWalletCreatedEvent.Payload.WalletUUID})

		sCtx.Assert().True(newDefaultWallet.IsDefault, "Новый кошелёк помечен как дефолтный в БД")
		sCtx.Assert().False(oldDefaultWallet.IsDefault, "Старый кошелёк больше не помечен как дефолтный в БД")
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
