package test

import (
	"fmt"
	"net/http"
	"testing"

	capAPI "CB_auto/internal/client/cap"
	capModels "CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/factory"
	publicAPI "CB_auto/internal/client/public"
	publicModels "CB_auto/internal/client/public/models"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/internal/repository/wallet"
	"CB_auto/internal/transport/kafka"
	"CB_auto/internal/transport/nats"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type BlockersParam struct {
	GamblingEnabled bool
	BettingEnabled  bool
	Description     string
}

type ParametrizedUpdateBlockersSuite struct {
	suite.Suite
	config        *config.Config
	publicService publicAPI.PublicAPI
	capService    capAPI.CapAPI
	natsClient    *nats.NatsClient
	kafka         *kafka.Kafka
	walletDB      *repository.Connector
	walletRepo    *wallet.Repository
	ParamBlockers []BlockersParam
}

func (s *ParametrizedUpdateBlockersSuite) BeforeAll(t provider.T) {
	t.WithNewStep("Чтение конфигурационного файла.", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация http-клиентов и сервисов.", func(sCtx provider.StepCtx) {
		s.publicService = factory.InitClient[publicAPI.PublicAPI](sCtx, s.config, clientTypes.Public)
		s.capService = factory.InitClient[capAPI.CapAPI](sCtx, s.config, clientTypes.Cap)
	})

	t.WithNewStep("Инициализация NATS клиента.", func(sCtx provider.StepCtx) {
		s.natsClient = nats.NewClient(&s.config.Nats)
	})

	t.WithNewStep("Инициализация Kafka.", func(sCtx provider.StepCtx) {
		s.kafka = kafka.GetInstance(t, s.config)
	})

	t.WithNewStep("Соединение с базой данных wallet.", func(sCtx provider.StepCtx) {
		s.walletRepo = wallet.NewRepository(repository.OpenConnector(t, &s.config.MySQL, repository.Wallet).DB(), &s.config.MySQL)
	})

	s.ParamBlockers = []BlockersParam{
		{GamblingEnabled: true, BettingEnabled: true, Description: "Гэмблинг и беттинг включены"},
		{GamblingEnabled: true, BettingEnabled: false, Description: "Гэмблинг включен, беттинг выключен"},
		{GamblingEnabled: false, BettingEnabled: true, Description: "Гэмблинг выключен, беттинг включен"},
		{GamblingEnabled: false, BettingEnabled: false, Description: "Гэмблинг и беттинг выключены"},
	}
}

func (s *ParametrizedUpdateBlockersSuite) TableTestBlockers(t provider.T, param BlockersParam) {
	t.Epic("Wallet")
	t.Feature("Управление блокировками игрока")
	t.Tags("Wallet", "Blockers")
	t.Title(fmt.Sprintf("Проверка управления блокировками игрока: %s", param.Description))

	type TestData struct {
		registrationResponse *clientTypes.Response[publicModels.FastRegistrationResponseBody]
		registrationMessage  kafka.PlayerMessage
		walletCreatedEvent   *nats.NatsMessage[nats.WalletCreatedPayload]
		blockersSettedEvent  *nats.NatsMessage[nats.BlockersSettedPayload]
	}
	var testData TestData

	t.WithNewStep("Регистрация пользователя.", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[publicModels.FastRegistrationRequestBody]{
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: &publicModels.FastRegistrationRequestBody{
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

	t.WithNewStep("Получение сообщения о создании основного кошелька из NATS.", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, testData.registrationMessage.Player.ExternalID)

		testData.walletCreatedEvent = nats.FindMessageInStream(
			sCtx, s.natsClient, subject, func(wallet nats.WalletCreatedPayload, msgType string) bool {
				return wallet.WalletType == nats.TypeReal &&
					wallet.WalletStatus == nats.StatusEnabled &&
					wallet.IsBasic
			})

		sCtx.Assert().NotEmpty(testData.walletCreatedEvent.Payload.WalletUUID, "UUID кошелька в ивенте `wallet_created` не пустой")
	})

	t.WithNewStep(fmt.Sprintf("Обновление блокировок игрока: %s", param.Description), func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[capModels.BlockersRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-Locale": "en",
				"Content-Type":    "application/json",
			},
			PathParams: map[string]string{
				"player_uuid": testData.registrationMessage.Player.ExternalID,
			},
			Body: &capModels.BlockersRequestBody{
				GamblingEnabled: param.GamblingEnabled,
				BettingEnabled:  param.BettingEnabled,
			},
		}
		resp := s.capService.UpdateBlockers(sCtx, req)

		sCtx.Assert().Equal(http.StatusNoContent, resp.StatusCode, "Статус код ответа равен 204")
	})

	t.WithNewStep("Проверка события обновления блокировок в NATS.", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.%s", s.config.Nats.StreamPrefix, testData.registrationMessage.Player.ExternalID, testData.walletCreatedEvent.Payload.WalletUUID)

		testData.blockersSettedEvent = nats.FindMessageInStream(
			sCtx, s.natsClient, subject, func(msg nats.BlockersSettedPayload, msgType string) bool {
				return msgType == "setting_prevent_gamble_setted"
			})

		sCtx.Assert().Equal(param.GamblingEnabled, testData.blockersSettedEvent.Payload.IsGamblingActive, "Гэмблинг настроен правильно")
		sCtx.Assert().Equal(param.BettingEnabled, testData.blockersSettedEvent.Payload.IsBettingActive, "Беттинг настроен правильно")
	})

	t.WithNewAsyncStep("Проверка блокировок в БД.", func(sCtx provider.StepCtx) {
		walletFromDatabase := s.walletRepo.GetWallet(sCtx, map[string]interface{}{
			"player_uuid":        testData.registrationMessage.Player.ExternalID,
			"is_gambling_active": param.GamblingEnabled,
			"is_betting_active":  param.BettingEnabled,
		})

		sCtx.Assert().Equal(param.GamblingEnabled, walletFromDatabase.IsGamblingActive, "Гэмблинг настроен правильно в БД")
		sCtx.Assert().Equal(param.BettingEnabled, walletFromDatabase.IsBettingActive, "Беттинг настроен правильно в БД")
	})

	t.WithNewAsyncStep("Проверка получения блокировок через API.", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken(sCtx)),
				"Platform-Locale": "en",
			},
			PathParams: map[string]string{
				"player_uuid": testData.registrationMessage.Player.ExternalID,
			},
		}
		resp := s.capService.GetBlockers(sCtx, req)

		sCtx.Assert().Equal(http.StatusOK, resp.StatusCode, "Статус код ответа равен 200")
		sCtx.Assert().Equal(param.GamblingEnabled, resp.Body.GamblingEnabled, "Гэмблинг настроен правильно в ответе API")
		sCtx.Assert().Equal(param.BettingEnabled, resp.Body.BettingEnabled, "Беттинг настроен правильно в ответе API")
	})
}

func (s *ParametrizedUpdateBlockersSuite) AfterAll(t provider.T) {
	if s.natsClient != nil {
		s.natsClient.Close()
	}
	if s.walletDB != nil {
		if err := s.walletDB.Close(); err != nil {
			t.Errorf("Ошибка при закрытии соединения с wallet DB: %v", err)
		}
	}
}

func TestParametrizedUpdateBlockersSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(ParametrizedUpdateBlockersSuite))
}
