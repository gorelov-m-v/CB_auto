package test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"CB_auto/internal/client/factory"
	publicAPI "CB_auto/internal/client/public"
	publicModels "CB_auto/internal/client/public/models"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/internal/repository/wallet"
	"CB_auto/internal/transport/kafka"
	"CB_auto/internal/transport/nats"
	"CB_auto/internal/transport/redis"
	"CB_auto/pkg/utils"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/suite"
)

type SingleBetLimitSuite struct {
	suite.Suite
	config       *config.Config
	publicClient publicAPI.PublicAPI
	kafka        *kafka.Kafka
	natsClient   *nats.NatsClient
	walletRepo   *wallet.Repository
	redisClient  *redis.RedisClient
}

func (s *SingleBetLimitSuite) BeforeAll(t provider.T) {
	t.Epic("Лимиты")

	t.WithNewStep("Чтение конфигурационного файла", func(sCtx provider.StepCtx) {
		s.config = config.ReadConfig(t)
	})

	t.WithNewStep("Инициализация Public API клиента", func(sCtx provider.StepCtx) {
		s.publicClient = factory.InitClient[publicAPI.PublicAPI](sCtx, s.config, clientTypes.Public)
	})

	t.WithNewStep("Инициализация Kafka клиента", func(sCtx provider.StepCtx) {
		s.kafka = kafka.GetInstance(t, s.config)
	})

	t.WithNewStep("Инициализация NATS клиента", func(sCtx provider.StepCtx) {
		s.natsClient = nats.NewClient(&s.config.Nats)
	})

	t.WithNewStep("Соединение с базой данных wallet.", func(sCtx provider.StepCtx) {
		s.walletRepo = wallet.NewRepository(repository.OpenConnector(t, &s.config.MySQL, repository.Wallet).DB(), &s.config.MySQL)
	})

	t.WithNewStep("Инициализация Redis клиента", func(sCtx provider.StepCtx) {
		s.redisClient = redis.NewRedisClient(t, &s.config.Redis, redis.WalletClient)
	})
}

func (s *SingleBetLimitSuite) TestSingleBetLimit(t provider.T) {
	t.Feature("single-bet лимит")
	t.Title("Проверка создания single-bet лимита в Kafka, NATS, Redis, MySQL, Public API, CAP API")

	var testData struct {
		registrationResponse  *clientTypes.Response[publicModels.FastRegistrationResponseBody]
		authorizationResponse *clientTypes.Response[publicModels.TokenCheckResponseBody]
		registrationMessage   kafka.PlayerMessage
		depositRequest        *clientTypes.Request[publicModels.DepositRequestBody]
		depositResponse       *clientTypes.Response[struct{}]
		updatePlayerRequest   *clientTypes.Request[publicModels.UpdatePlayerRequestBody]
		updatePlayerResponse  *clientTypes.Response[publicModels.UpdatePlayerResponseBody]
		setSingleBetLimitReq  *clientTypes.Request[publicModels.SetSingleBetLimitRequestBody]
		setTurnoverLimitReq   *clientTypes.Request[publicModels.SetTurnoverLimitRequestBody]
	}

	t.WithNewStep("Регистрация нового игрока", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[publicModels.FastRegistrationRequestBody]{
			Body: &publicModels.FastRegistrationRequestBody{
				Country:  s.config.Node.DefaultCountry,
				Currency: s.config.Node.DefaultCurrency,
			},
		}

		testData.registrationResponse = s.publicClient.FastRegistration(sCtx, req)

		sCtx.Assert().Equal(http.StatusOK, testData.registrationResponse.StatusCode, "Успешная регистрация")
	})

	t.WithNewStep("Проверка Kafka-сообщения о регистрации игрока", func(sCtx provider.StepCtx) {
		testData.registrationMessage = kafka.FindMessageByFilter[kafka.PlayerMessage](sCtx, s.kafka, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == "player.signUpFast" &&
				msg.Player.AccountID == testData.registrationResponse.Body.Username
		})

		sCtx.Require().NotEmpty(testData.registrationMessage.Player.ID, "ID игрока не пустой")
	})

	t.WithNewStep("Получение токена авторизации", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[publicModels.TokenCheckRequestBody]{
			Body: &publicModels.TokenCheckRequestBody{
				Username: testData.registrationResponse.Body.Username,
				Password: testData.registrationResponse.Body.Password,
			},
		}

		testData.authorizationResponse = s.publicClient.TokenCheck(sCtx, req)

		sCtx.Require().Equal(http.StatusOK, testData.authorizationResponse.StatusCode, "Успешная авторизация")
	})

	t.WithNewStep("Обновление данных игрока", func(sCtx provider.StepCtx) {
		testData.updatePlayerRequest = &clientTypes.Request[publicModels.UpdatePlayerRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
				"Content-Type":    "application/json",
				"Platform-Locale": "en",
			},
			Body: &publicModels.UpdatePlayerRequestBody{
				FirstName:        "Test",
				LastName:         "Test",
				Gender:           1,
				City:             "sadsadasd",
				Postcode:         "12334512",
				PermanentAddress: "1231asfsad",
				PersonalID:       utils.Get(utils.PERSONAL_ID, 10),
				Profession:       "QA",
				IBAN:             utils.Get(utils.IBAN, 10),
				Birthday:         "1980-01-01",
				Country:          "LV",
			},
		}

		testData.updatePlayerResponse = s.publicClient.UpdatePlayer(sCtx, testData.updatePlayerRequest)
		sCtx.Require().Equal(http.StatusOK, testData.updatePlayerResponse.StatusCode, "Статус код ответа равен 200")

		sCtx.Assert().Equal("Test", testData.updatePlayerResponse.Body.FirstName, "Имя обновлено корректно")
		sCtx.Assert().Equal("Test", testData.updatePlayerResponse.Body.LastName, "Фамилия обновлена корректно")
		sCtx.Assert().Equal("LV", testData.updatePlayerResponse.Body.Country, "Страна обновлена корректно")
	})

	t.WithNewStep("Установка лимита на одиночную ставку", func(sCtx provider.StepCtx) {
		testData.setSingleBetLimitReq = &clientTypes.Request[publicModels.SetSingleBetLimitRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
			},
			Body: &publicModels.SetSingleBetLimitRequestBody{
				Amount:   "100",
				Currency: s.config.Node.DefaultCurrency,
			},
		}

		resp := s.publicClient.SetSingleBetLimit(sCtx, testData.setSingleBetLimitReq)

		sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Лимит на ставку установлен")
	})

	t.WithNewStep("Установка лимита на оборот средств", func(sCtx provider.StepCtx) {
		testData.setTurnoverLimitReq = &clientTypes.Request[publicModels.SetTurnoverLimitRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
			},
			Body: &publicModels.SetTurnoverLimitRequestBody{
				Amount:    "100",
				Currency:  s.config.Node.DefaultCurrency,
				Type:      "daily",
				StartedAt: time.Now().Unix(),
			},
		}

		resp := s.publicClient.SetTurnoverLimit(sCtx, testData.setTurnoverLimitReq)

		sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Лимит на оборот средств установлен")
	})

	time.Sleep(10 * time.Second)

	t.WithNewStep("Создание депозита", func(sCtx provider.StepCtx) {
		testData.depositRequest = &clientTypes.Request[publicModels.DepositRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", testData.authorizationResponse.Body.Token),
				"Content-Type":    "application/json",
				"Platform-Locale": "en",
			},
			Body: &publicModels.DepositRequestBody{
				Amount:          "10",
				PaymentMethodID: 1001,
				Currency:        s.config.Node.DefaultCurrency,
				Country:         s.config.Node.DefaultCountry,
				Redirect: publicModels.DepositRedirectURLs{
					Failed:  publicModels.DepositRedirectURLFailed,
					Success: publicModels.DepositRedirectURLSuccess,
					Pending: publicModels.DepositRedirectURLPending,
				},
			},
		}

		testData.depositResponse = s.publicClient.CreateDeposit(sCtx, testData.depositRequest)
		sCtx.Require().Equal(http.StatusCreated, testData.depositResponse.StatusCode, "Статус код ответа равен 201")
	})
}

func (s *SingleBetLimitSuite) AfterAll(t provider.T) {
	kafka.CloseInstance(t)
	if s.natsClient != nil {
		s.natsClient.Close()
	}
}

func TestSingleBetLimitSuite(t *testing.T) {
	t.Parallel()
	suite.RunSuite(t, new(SingleBetLimitSuite))
}
