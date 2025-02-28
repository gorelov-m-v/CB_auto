package test

import (
	"fmt"
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

type FastRegistrationSuite struct {
	suite.Suite
	config        *config.Config
	publicService publicAPI.PublicAPI
	natsClient    *nats.NatsClient
	redisClient   *redis.RedisClient
	kafka         *kafka.Kafka
	walletDB      *repository.Connector
	walletRepo    *wallet.Repository
}

func (s *FastRegistrationSuite) BeforeAll(t provider.T) {
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
		connector := repository.OpenConnector(t, &s.config.MySQL, repository.Wallet)
		s.walletDB = &connector
		s.walletRepo = wallet.NewRepository(s.walletDB.DB(), &s.config.MySQL)
	})
}

func (s *FastRegistrationSuite) TestFastRegistration(t provider.T) {
	t.Epic("Wallet")
	t.Feature("Создание кошелька при регистрации пользователя")
	t.Tags("Wallet", "Registration")
	t.Title("Проверка создания кошелька при регистрации пользователя")

	type TestData struct {
		authResponse              *clientTypes.Response[models.TokenCheckResponseBody]
		registrationResponse      *clientTypes.Response[models.FastRegistrationResponseBody]
		playerRegistrationMessage kafka.PlayerMessage
		walletCreatedEvent        *nats.NatsMessage[nats.WalletCreatedPayload]
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
		testData.playerRegistrationMessage = kafka.FindMessageByFilter[kafka.PlayerMessage](sCtx, s.kafka, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == "player.signUpFast" &&
				msg.Player.AccountID == testData.registrationResponse.Body.Username
		})

		sCtx.Require().NotEmpty(testData.playerRegistrationMessage.Player.ExternalID, "External ID игрока в регистрации не пустой")
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

	t.WithNewStep("Проверка создания кошелька в NATS.", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", s.config.Nats.StreamPrefix, testData.playerRegistrationMessage.Player.ExternalID)

		testData.walletCreatedEvent = nats.FindMessageInStream(sCtx, s.natsClient, subject, func(wallet nats.WalletCreatedPayload, msgType string) bool {
			return wallet.WalletType == nats.TypeReal &&
				wallet.WalletStatus == nats.StatusEnabled &&
				wallet.IsBasic
		})

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
	})

	t.WithNewAsyncStep("Проверка значения в Redis.", func(sCtx provider.StepCtx) {
		var wallets redis.WalletsMap
		err := s.redisClient.GetWithRetry(sCtx, testData.playerRegistrationMessage.Player.ExternalID, &wallets)

		sCtx.Require().NoError(err, "Значение кошелька получено из Redis")

		var wallet redis.WalletData
		for _, w := range wallets {
			wallet = w
			break
		}

		sCtx.Assert().Equal(int(nats.TypeReal), wallet.Type, "Тип кошелька в Redis – реальный")
		sCtx.Assert().Equal(int(nats.StatusEnabled), wallet.Status, "Статус кошелька в Redis – включён")
	})

	t.WithNewAsyncStep("Проверка создания кошелька в БД.", func(sCtx provider.StepCtx) {
		walletFromDatabase := s.walletRepo.GetWallet(sCtx, map[string]interface{}{"uuid": testData.walletCreatedEvent.Payload.WalletUUID})

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
	})

	t.WithNewAsyncStep("Проверка получения списка кошельков.", func(sCtx provider.StepCtx) {
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
		sCtx.Assert().Equal(testData.walletCreatedEvent.Payload.Currency, foundWallet.Currency, "Валюта кошелька совпадает с ожидаемой")
		sCtx.Assert().Equal(constants.ZeroAmount, foundWallet.Balance, "Баланс кошелька равен 0")
		sCtx.Assert().True(foundWallet.Default, "Кошелёк помечен как дефолтный")
	})

	t.WithNewAsyncStep("Проверка данных кошелька в Redis", func(sCtx provider.StepCtx) {
		var redisValue redis.WalletFullData
		err := s.redisClient.GetWithRetry(sCtx, testData.walletCreatedEvent.Payload.WalletUUID, &redisValue)

		sCtx.Require().NoError(err, "Значение кошелька получено из Redis")
		sCtx.Require().NotEmpty(redisValue.Limits, "Должен быть хотя бы один лимит")

		limitData := redisValue.Limits[0]

		sCtx.Assert().Equal(testData.walletCreatedEvent.Payload.WalletUUID, limitData.ExternalID, "ID лимита совпадает")
		sCtx.Assert().Equal("turnover-of-funds", limitData.LimitType, "Тип лимита совпадает")
		sCtx.Assert().Equal("daily", limitData.IntervalType, "Интервал лимита совпадает")
		sCtx.Assert().Equal(testData.walletCreatedEvent.Payload.Currency, limitData.Amount, "Сумма лимита совпадает")
		sCtx.Assert().Equal("0", limitData.Spent, "Потраченная сумма равна 0")
		sCtx.Assert().Equal(testData.walletCreatedEvent.Payload.Currency, limitData.Rest, "Остаток равен сумме лимита")
	})
}

func (s *FastRegistrationSuite) AfterAll(t provider.T) {
	kafka.CloseInstance(t)
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
