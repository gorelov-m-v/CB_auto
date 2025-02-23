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

type UpdateBlockersSuite struct {
	suite.Suite
	config        *config.Config
	publicService publicAPI.PublicAPI
	capService    capAPI.CapAPI
	natsClient    *nats.NatsClient
	kafka         *kafka.Kafka
	walletDB      *repository.Connector
	walletRepo    *wallet.Repository
}

func (s *UpdateBlockersSuite) BeforeAll(t provider.T) {
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
		s.kafka = kafka.GetInstance(t, s.config, kafka.PlayerTopic)
	})

	t.WithNewStep("Соединение с базой данных wallet.", func(sCtx provider.StepCtx) {
		connector := repository.OpenConnector(t, &s.config.MySQL, repository.Wallet)
		s.walletDB = &connector
		s.walletRepo = wallet.NewRepository(s.walletDB.DB(), &s.config.MySQL)
	})
}

func (s *UpdateBlockersSuite) TestUpdateBlockers(t provider.T) {
	t.Epic("Wallet")
	t.Feature("Управление блокировками игрока")
	t.Tags("Wallet", "Blockers")
	t.Title("Проверка управления блокировками игрока")

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
		subject := fmt.Sprintf("%s.wallet.*.%s.*",
			s.config.Nats.StreamPrefix,
			testData.registrationMessage.Player.ExternalID,
		)
		s.natsClient.SubscribeWithDeliverAll(subject)

		testData.walletCreatedEvent = nats.FindMessageByFilter(
			sCtx, s.natsClient, func(wallet nats.WalletCreatedPayload, msgType string) bool {
				return wallet.WalletType == nats.TypeReal &&
					wallet.WalletStatus == nats.StatusEnabled &&
					wallet.IsBasic
			})

		sCtx.Assert().NotEmpty(testData.walletCreatedEvent.Payload.WalletUUID, "UUID кошелька в ивенте `wallet_created` не пустой")
	})

	t.WithNewStep("Обновление блокировок игрока.", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[capModels.BlockersRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken()),
				"Platform-Locale": "en",
				"Content-Type":    "application/json",
			},
			PathParams: map[string]string{
				"player_uuid": testData.registrationMessage.Player.ExternalID,
			},
			Body: &capModels.BlockersRequestBody{
				GamblingEnabled: false,
				BettingEnabled:  true,
			},
		}
		resp := s.capService.UpdateBlockers(sCtx, req)

		sCtx.Assert().Equal(http.StatusNoContent, resp.StatusCode, "Статус код ответа равен 204")
	})

	t.WithNewStep("Проверка события обновления блокировок в NATS.", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*",
			s.config.Nats.StreamPrefix,
			testData.registrationMessage.Player.ExternalID,
		)
		s.natsClient.SubscribeWithDeliverAll(subject)

		testData.blockersSettedEvent = nats.FindMessageByFilter(
			sCtx, s.natsClient, func(msg nats.BlockersSettedPayload, msgType string) bool {
				return msgType == "setting_prevent_gamble_setted"
			})

		sCtx.Assert().False(testData.blockersSettedEvent.Payload.IsGamblingActive, "Гэмблинг отключен")
		sCtx.Assert().True(testData.blockersSettedEvent.Payload.IsBettingActive, "Беттинг включен")
	})

	t.WithNewAsyncStep("Проверка блокировок в БД.", func(sCtx provider.StepCtx) {
		walletFromDatabase := s.walletRepo.GetWallet(sCtx, map[string]interface{}{
			"player_uuid": testData.registrationMessage.Player.ExternalID,
		})

		sCtx.Assert().False(walletFromDatabase.IsGamblingActive, "Гэмблинг отключен в БД")
		sCtx.Assert().True(walletFromDatabase.IsBettingActive, "Беттинг включен в БД")
	})

	t.WithNewAsyncStep("Проверка получения блокировок через API.", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", s.capService.GetToken()),
				"Platform-Locale": "en",
			},
			PathParams: map[string]string{
				"player_uuid": testData.registrationMessage.Player.ExternalID,
			},
		}
		resp := s.capService.GetBlockers(sCtx, req)

		sCtx.Assert().Equal(http.StatusOK, resp.StatusCode, "Статус код ответа равен 200")
		sCtx.Assert().False(resp.Body.GamblingEnabled, "Гэмблинг отключен в ответе API")
		sCtx.Assert().True(resp.Body.BettingEnabled, "Беттинг включен в ответе API")
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
