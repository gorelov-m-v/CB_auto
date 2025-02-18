package test

import (
	"fmt"
	"testing"

	client "CB_auto/internal/client"
	capAPI "CB_auto/internal/client/cap"
	capModels "CB_auto/internal/client/cap/models"
	publicAPI "CB_auto/internal/client/public"
	publicModels "CB_auto/internal/client/public/models"
	"CB_auto/internal/config"
	"CB_auto/internal/database"
	"CB_auto/internal/database/wallet"
	"CB_auto/internal/transport/kafka"
	"CB_auto/internal/transport/nats"
	"CB_auto/pkg/utils"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type UpdateBlockersSuite struct {
	suite.Suite
	client        *client.Client
	capClient     *client.Client
	config        *config.Config
	publicService publicAPI.PublicAPI
	capService    capAPI.CapAPI
	natsClient    *nats.NatsClient
	kafka         *kafka.Kafka
	walletDB      *database.Connector
}

func (s *UpdateBlockersSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла.", func(sCtx provider.StepCtx) {
		cfg, err := config.ReadConfig()
		if err != nil {
			t.Fatalf("Ошибка при чтении конфигурации: %v", err)
		}
		s.config = cfg
	})

	t.WithNewStep("Инициализация http-клиентов и сервисов.", func(sCtx provider.StepCtx) {
		publicClient, err := client.InitClient(s.config, client.Public)
		if err != nil {
			t.Fatalf("InitClient для public не удался: %v", err)
		}
		s.client = publicClient
		s.publicService = publicAPI.NewPublicClient(s.client)

		capClient, err := client.InitClient(s.config, client.Cap)
		if err != nil {
			t.Fatalf("InitClient для cap не удался: %v", err)
		}
		s.capClient = capClient
		s.capService = capAPI.NewCapClient(s.capClient)
	})

	t.WithNewStep("Инициализация NATS клиента.", func(sCtx provider.StepCtx) {
		natsClient, err := nats.NewClient(&s.config.Nats)
		if err != nil {
			t.Fatalf("NewClient NATS не удался: %v", err)
		}
		s.natsClient = natsClient
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

func (s *UpdateBlockersSuite) TestUpdateBlockers(t provider.T) {
	t.Epic("Wallet")
	t.Feature("Управление блокировками игрока")
	t.Tags("Wallet", "Blockers")
	t.Title("Проверка управления блокировками игрока")

	type TestData struct {
		registrationResponse      *publicModels.FastRegistrationResponseBody
		playerRegistrationMessage *kafka.PlayerMessage
		walletCreatedEvent        *nats.NatsMessage[nats.WalletCreatedPayload]
		capAuthResponse           *capModels.AdminCheckResponseBody
		blockersSettedEvent       *nats.NatsMessage[nats.BlockersSettedPayload]
	}
	var testData TestData

	t.WithNewStep("Регистрация пользователя.", func(sCtx provider.StepCtx) {
		createReq := &client.Request[publicModels.FastRegistrationRequestBody]{
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: &publicModels.FastRegistrationRequestBody{
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
		message := kafka.FindMessageByFilter(s.kafka, t, func(msg kafka.PlayerMessage) bool {
			return msg.Player.AccountID == testData.registrationResponse.Username
		})
		playerRegistrationMessage := kafka.ParseMessage[kafka.PlayerMessage](t, message)
		testData.playerRegistrationMessage = &playerRegistrationMessage

		sCtx.WithAttachments(allure.NewAttachment("Kafka Player Message", allure.JSON, utils.CreatePrettyJSON(testData.playerRegistrationMessage)))
	})

	t.WithNewStep("Получение сообщения о создании основного кошелька из NATS.", func(sCtx provider.StepCtx) {
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
		sCtx.WithAttachments(allure.NewAttachment("NATS Wallet Message", allure.JSON, utils.CreatePrettyJSON(testData.walletCreatedEvent)))
	})

	t.WithNewStep("Авторизация в CAP.", func(sCtx provider.StepCtx) {
		authReq := &client.Request[capModels.AdminCheckRequestBody]{
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: &capModels.AdminCheckRequestBody{
				UserName: s.config.Cap.AdminUsername,
				Password: s.config.Cap.AdminPassword,
			},
		}
		authResp := s.capService.CheckAdmin(authReq)
		testData.capAuthResponse = &authResp.Body

		sCtx.Assert().NotEmpty(authResp.Body.Token, "Токен авторизации не пустой")
		sCtx.Assert().NotEmpty(authResp.Body.RefreshToken, "Refresh токен не пустой")

		sCtx.WithAttachments(allure.NewAttachment("CAP Auth Request", allure.JSON, utils.CreateHttpAttachRequest(authReq)))
		sCtx.WithAttachments(allure.NewAttachment("CAP Auth Response", allure.JSON, utils.CreateHttpAttachResponse(authResp)))
	})

	t.WithNewStep("Обновление блокировок игрока.", func(sCtx provider.StepCtx) {
		updateReq := &client.Request[capModels.BlockersRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.capAuthResponse.Token),
				"Platform-Locale": "en",
				"Content-Type":    "application/json",
			},
			PathParams: map[string]string{
				"player_uuid": testData.playerRegistrationMessage.Player.ExternalID,
			},
			Body: &capModels.BlockersRequestBody{
				GamblingEnabled: false,
				BettingEnabled:  true,
			},
		}
		updateResp := s.capService.UpdateBlockers(updateReq)

		sCtx.Assert().Equal(204, updateResp.StatusCode, "Статус код ответа равен 204")

		sCtx.WithAttachments(allure.NewAttachment("UpdateBlockers Request", allure.JSON, utils.CreateHttpAttachRequest(updateReq)))
		sCtx.WithAttachments(allure.NewAttachment("UpdateBlockers Response", allure.JSON, utils.CreateHttpAttachResponse(updateResp)))
	})

	t.WithNewStep("Проверка события обновления блокировок в NATS.", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*",
			s.config.Nats.StreamPrefix,
			testData.playerRegistrationMessage.Player.ExternalID,
		)
		s.natsClient.Subscribe(t, subject)

		testData.blockersSettedEvent = nats.FindMessageByFilter[nats.BlockersSettedPayload](
			s.natsClient, t, func(msg nats.BlockersSettedPayload, msgType string) bool {
				return msgType == "setting_prevent_gamble_setted"
			},
		)

		sCtx.Assert().False(testData.blockersSettedEvent.Payload.IsGamblingActive, "Гэмблинг отключен")
		sCtx.Assert().True(testData.blockersSettedEvent.Payload.IsBettingActive, "Беттинг включен")

		sCtx.WithAttachments(allure.NewAttachment("BlockersSetted Event", allure.JSON, utils.CreatePrettyJSON(testData.blockersSettedEvent)))
	})

	t.WithNewAsyncStep("Проверка блокировок в БД.", func(sCtx provider.StepCtx) {
		walletRepo := wallet.NewRepository(s.walletDB.DB, &s.config.MySQL)
		walletFromDatabase := walletRepo.GetWallet(t, map[string]interface{}{
			"player_uuid": testData.playerRegistrationMessage.Player.ExternalID,
		})

		sCtx.Assert().False(walletFromDatabase.IsGamblingActive, "Гэмблинг отключен в БД")
		sCtx.Assert().True(walletFromDatabase.IsBettingActive, "Беттинг включен в БД")

		sCtx.WithAttachments(allure.NewAttachment("Wallet DB Data", allure.JSON, utils.CreatePrettyJSON(walletFromDatabase)))
	})

	t.WithNewAsyncStep("Проверка получения блокировок через API.", func(sCtx provider.StepCtx) {
		getBlockersReq := &client.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.capAuthResponse.Token),
				"Platform-Locale": "en",
			},
			PathParams: map[string]string{
				"player_uuid": testData.playerRegistrationMessage.Player.ExternalID,
			},
		}
		getBlockersResp := s.capService.GetBlockers(getBlockersReq)

		sCtx.Assert().Equal(200, getBlockersResp.StatusCode, "Статус код ответа равен 200")
		sCtx.Assert().False(getBlockersResp.Body.GamblingEnabled, "Гэмблинг отключен в ответе API")
		sCtx.Assert().True(getBlockersResp.Body.BettingEnabled, "Беттинг включен в ответе API")

		sCtx.WithAttachments(allure.NewAttachment("GetBlockers Request", allure.JSON, utils.CreateHttpAttachRequest(getBlockersReq)))
		sCtx.WithAttachments(allure.NewAttachment("GetBlockers Response", allure.JSON, utils.CreateHttpAttachResponse(getBlockersResp)))
	})
}

func (s *UpdateBlockersSuite) AfterAll(t provider.T) {
	if s.natsClient != nil {
		s.natsClient.Close()
	}
	if s.walletDB != nil {
		if err := s.walletDB.Close(); err != nil {
			t.Errorf("Ошибка при закрытии соединения с wallet DB: %v", err)
		}
	}
}

func TestUpdateBlockersSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(UpdateBlockersSuite))
}
