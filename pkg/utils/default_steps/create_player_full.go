package utils

import (
	"net/http"
	"strings"

	publicAPI "CB_auto/internal/client/public"
	publicModels "CB_auto/internal/client/public/models"
	clientTypes "CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/internal/transport/kafka"
	"CB_auto/internal/transport/nats"
	"CB_auto/internal/transport/redis"
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

// CreatePlayerWithFullRegistration создаёт игрока через полную регистрацию с подтверждением телефона
func CreatePlayerWithFullRegistration(
	sCtx provider.StepCtx,
	publicClient publicAPI.PublicAPI,
	kafkaClient *kafka.Kafka,
	config *config.Config,
	redisPlayerClient *redis.RedisClient,
	redisWalletClient *redis.RedisClient,
) PlayerData {
	// Внутренняя структура для хранения данных при создании игрока
	var registrationData struct {
		phoneVerificationRequest *clientTypes.Request[publicModels.RequestVerificationRequestBody]
		phoneConfirmationMessage kafka.PlayerMessage
		verifyContactResponse    *clientTypes.Response[publicModels.VerifyContactResponseBody]
		fullRegRequest           *clientTypes.Request[publicModels.FullRegistrationRequestBody]
		fullRegResponse          *clientTypes.Response[struct{}]
		authorizationResponse    *clientTypes.Response[publicModels.TokenCheckResponseBody]
		registrationMessage      kafka.PlayerMessage
		depositEvent             *nats.NatsMessage[nats.DepositedMoneyPayload]
		wallets                  redis.WalletsMap
	}

	playerData := PlayerData{}

	// Шаг 1: Запрос подтверждения телефона
	sCtx.WithNewStep("Запрос подтверждения телефона", func(sCtx provider.StepCtx) {
		registrationData.phoneVerificationRequest = &clientTypes.Request[publicModels.RequestVerificationRequestBody]{
			Body: &publicModels.RequestVerificationRequestBody{
				Contact: utils.Get(utils.PHONE),
				Type:    publicModels.ContactTypePhone,
			},
		}

		resp := publicClient.RequestContactVerification(sCtx, registrationData.phoneVerificationRequest)
		sCtx.Require().Equal(http.StatusOK, resp.StatusCode, "Public API: Запрос на подтверждение телефона успешно отправлен")
		sCtx.Logf("Отправлен запрос на подтверждение телефона: %s", registrationData.phoneVerificationRequest.Body.Contact)
	})

	// Шаг 2: Получение кода подтверждения из Kafka
	sCtx.WithNewStep("Получение кода подтверждения из Kafka", func(sCtx provider.StepCtx) {
		phoneNumberWithoutPlus := strings.TrimPrefix(registrationData.phoneVerificationRequest.Body.Contact, "+")

		registrationData.phoneConfirmationMessage = kafka.FindMessageByFilter(sCtx, kafkaClient, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == kafka.PlayerEventConfirmationPhone &&
				msg.Player.Phone == phoneNumberWithoutPlus
		})

		sCtx.Require().Equal(string(kafka.PlayerEventConfirmationPhone), string(registrationData.phoneConfirmationMessage.Message.EventType),
			"Kafka: Сообщение player.confirmationPhone найдено")
		sCtx.Assert().Equal(phoneNumberWithoutPlus, registrationData.phoneConfirmationMessage.Player.Phone,
			"Kafka: Телефон в сообщении соответствует запрошенному")

		confirmationContext, err := registrationData.phoneConfirmationMessage.GetConfirmationContext()
		sCtx.Require().NoError(err, "Kafka: Контекст с кодом подтверждения успешно получен")
		sCtx.Require().NotEmpty(confirmationContext.ConfirmationCode, "Kafka: Код подтверждения не пустой")
		sCtx.Logf("Получен код подтверждения: %s", confirmationContext.ConfirmationCode)
	})

	// Шаг 3: Вызов эндпоинта верификации контакта
	sCtx.WithNewStep("Вызов эндпоинта верификации контакта", func(sCtx provider.StepCtx) {
		confirmationContext, _ := registrationData.phoneConfirmationMessage.GetConfirmationContext()

		req := &clientTypes.Request[publicModels.VerifyContactRequestBody]{
			Body: &publicModels.VerifyContactRequestBody{
				Contact: strings.TrimPrefix(registrationData.phoneVerificationRequest.Body.Contact, "+"),
				Code:    confirmationContext.ConfirmationCode,
			},
		}

		registrationData.verifyContactResponse = publicClient.VerifyContact(sCtx, req)
		sCtx.Require().Equal(http.StatusOK, registrationData.verifyContactResponse.StatusCode, "Public API: Контакт успешно верифицирован")
		sCtx.Require().NotEmpty(registrationData.verifyContactResponse.Body.Hash, "Public API: Получен непустой хэш верификации")
		sCtx.Logf("Получен хэш верификации: %s", registrationData.verifyContactResponse.Body.Hash)
	})

	// Шаг 4: Выполнение полной регистрации пользователя
	sCtx.WithNewStep("Выполнение полной регистрации пользователя", func(sCtx provider.StepCtx) {
		phoneNumber := strings.TrimPrefix(registrationData.phoneVerificationRequest.Body.Contact, "+")
		birthDate := "1990-01-01"
		password := "SecurePassword123!"

		registrationData.fullRegRequest = &clientTypes.Request[publicModels.FullRegistrationRequestBody]{
			Body: &publicModels.FullRegistrationRequestBody{
				Currency:          config.Node.DefaultCurrency,
				Country:           config.Node.DefaultCountry,
				BonusChoice:       publicModels.BonusChoiceNone,
				Phone:             phoneNumber,
				PhoneConfirmation: registrationData.verifyContactResponse.Body.Hash,
				FirstName:         "Иван",
				LastName:          "Петров",
				Birthday:          birthDate,
				Gender:            publicModels.GenderMale,
				PersonalId:        utils.Get(utils.PERSONAL_ID),
				IBAN:              utils.Get(utils.IBAN),
				City:              "Москва",
				PermanentAddress:  "ул. Примерная, д. 123",
				PostalCode:        "123456",
				Profession:        "Инженер",
				Password:          password,
				RulesAgreement:    true,
				Context:           map[string]any{},
			},
		}

		registrationData.fullRegResponse = publicClient.FullRegistration(sCtx, registrationData.fullRegRequest)
		sCtx.Require().Equal(http.StatusCreated, registrationData.fullRegResponse.StatusCode, "Public API: Полная регистрация выполнена успешно")
	})

	// Шаг 5: Получение Kafka-сообщения о регистрации
	sCtx.WithNewStep("Получение Kafka-сообщения о регистрации", func(sCtx provider.StepCtx) {
		phoneNumber := strings.TrimPrefix(registrationData.phoneVerificationRequest.Body.Contact, "+")
		registrationData.registrationMessage = kafka.FindMessageByFilter(sCtx, kafkaClient, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == kafka.PlayerEventSignUpFull &&
				msg.Player.Phone == phoneNumber
		})

		sCtx.Require().NotEmpty(registrationData.registrationMessage.Player.ExternalID, "ExternalID игрока не пустой")
	})

	// Шаг 6: Авторизация
	sCtx.WithNewStep("Авторизация", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[publicModels.TokenCheckRequestBody]{
			Body: &publicModels.TokenCheckRequestBody{
				Username: registrationData.fullRegRequest.Body.Phone,
				Password: registrationData.fullRegRequest.Body.Password,
			},
		}

		registrationData.authorizationResponse = publicClient.TokenCheck(sCtx, req)
		sCtx.Require().Equal(http.StatusOK, registrationData.authorizationResponse.StatusCode, "Успешная авторизация")
	})

	// Шаг 7: Получение WalletUUID из Redis
	sCtx.WithNewStep("Получение WalletUUID из Redis", func(sCtx provider.StepCtx) {
		var wallets redis.WalletsMap
		err := redisPlayerClient.GetWithRetry(sCtx, registrationData.registrationMessage.Player.ExternalID, &wallets)

		registrationData.wallets = wallets
		sCtx.Require().NoError(err, "Значение кошелька получено из Redis")

		var walletUUIDs []string
		for _, wallet := range wallets {
			walletUUIDs = append(walletUUIDs, wallet.WalletUUID)
		}
		sCtx.Require().NotEmpty(walletUUIDs, "Кошельки найдены")
	})

	// Шаг 8: Получение обновленных данных кошелька из Redis
	sCtx.WithNewStep("Получение обновленных данных кошелька из Redis", func(sCtx provider.StepCtx) {
		var walletUUID string
		for _, wallet := range registrationData.wallets {
			walletUUID = wallet.WalletUUID
			break
		}

		var updatedWalletData redis.WalletFullData
		err := redisWalletClient.GetWithRetry(
			sCtx,
			walletUUID,
			&updatedWalletData)
		sCtx.Require().NoError(err, "Получены обновленные данные кошелька из Redis")

		playerData = PlayerData{
			Auth:       registrationData.authorizationResponse,
			WalletData: updatedWalletData,
		}
	})

	return playerData
}
