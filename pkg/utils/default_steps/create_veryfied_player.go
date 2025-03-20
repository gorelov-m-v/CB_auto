package utils

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	capAPI "CB_auto/internal/client/cap"
	capModels "CB_auto/internal/client/cap/models"
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

type PlayerData struct {
	Auth       *clientTypes.Response[publicModels.TokenCheckResponseBody]
	WalletData redis.WalletFullData
}

func CreateVerifiedPlayer(
	sCtx provider.StepCtx,
	publicClient publicAPI.PublicAPI,
	capClient capAPI.CapAPI,
	kafkaClient *kafka.Kafka,
	config *config.Config,
	redisPlayerClient *redis.RedisClient,
	redisWalletClient *redis.RedisClient,
	natsClient *nats.NatsClient,
	depositAmount float64,
) PlayerData {
	// Внутренняя структура для хранения оперативных данных при создании игрока
	var registrationData struct {
		registrationMessage      kafka.PlayerMessage
		authorizationResponse    *clientTypes.Response[publicModels.TokenCheckResponseBody]
		fullRegRequest           *clientTypes.Request[publicModels.FullRegistrationRequestBody]
		fullRegResponse          *clientTypes.Response[struct{}]
		verifyContactResponse    *clientTypes.Response[publicModels.VerifyContactResponseBody]
		depositEvent             *nats.NatsMessage[nats.DepositedMoneyPayload]
		verifications            *clientTypes.Response[[]publicModels.VerificationStatusResponseItem]
		phoneVerificationRequest *clientTypes.Request[publicModels.RequestVerificationRequestBody]
		emailVerificationRequest *clientTypes.Request[publicModels.RequestVerificationRequestBody]
		phoneConfirmationMessage kafka.PlayerMessage
		emailConfirmationMessage kafka.PlayerMessage
		wallets                  redis.WalletsMap
		depositeEvent            *nats.NatsMessage[nats.DepositedMoneyPayload]
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

	// Шаг 5: Создание запроса на подтверждение личности
	sCtx.WithNewStep("Создание запроса на подтверждение личности", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[publicModels.VerifyIdentityRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", registrationData.authorizationResponse.Body.Token),
			},
			Body: &publicModels.VerifyIdentityRequestBody{
				Number:     "305003277",
				Type:       publicModels.VerificationTypeIdentity,
				IssuedDate: "1421463275.791",
				ExpiryDate: "1921463275.791",
			},
		}

		resp := publicClient.VerifyIdentity(sCtx, req)
		sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Верификация идентичности")
	})

	// Шаг 6: Получение статуса верификации
	sCtx.WithNewStep("Получение статуса верификации", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[any]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", registrationData.authorizationResponse.Body.Token),
			},
		}

		registrationData.verifications = publicClient.GetVerificationStatus(sCtx, req)
		sCtx.Require().Equal(http.StatusOK, registrationData.verifications.StatusCode, "Получение статуса верификации")
	})

	// Шаг 7: Обновление статуса верификации
	sCtx.WithNewStep("Обновление статуса верификации", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[capModels.UpdateVerificationStatusRequestBody]{
			Headers: map[string]string{
				"Authorization":   fmt.Sprintf("Bearer %s", capClient.GetToken(sCtx)),
				"Platform-NodeID": config.Node.ProjectID,
			},
			PathParams: map[string]string{
				"verification_id": registrationData.verifications.Body[0].DocumentID,
			},
			Body: &capModels.UpdateVerificationStatusRequestBody{
				Note:   "",
				Reason: "",
				Status: capModels.VerificationStatusApproved,
			},
		}

		resp := capClient.UpdateVerificationStatus(sCtx, req)
		sCtx.Require().Equal(http.StatusNoContent, resp.StatusCode, "Обновление статуса верификации")
	})

	// Шаг 9: Запрос верификации email
	sCtx.WithNewStep("Запрос верификации email", func(sCtx provider.StepCtx) {
		registrationData.emailVerificationRequest = &clientTypes.Request[publicModels.RequestVerificationRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", registrationData.authorizationResponse.Body.Token),
			},
			Body: &publicModels.RequestVerificationRequestBody{
				Contact: fmt.Sprintf("test%d@example.com", time.Now().Unix()),
				Type:    publicModels.ContactTypeEmail,
			},
		}

		resp := publicClient.RequestContactVerification(sCtx, registrationData.emailVerificationRequest)
		sCtx.Require().Equal(http.StatusOK, resp.StatusCode, "Запрос верификации email")
	})

	// Шаг 10: Получение сообщения о подтверждении  email
	sCtx.WithNewStep("Получение сообщения о подтверждении телефона и email", func(sCtx provider.StepCtx) {
		registrationData.emailConfirmationMessage = kafka.FindMessageByFilter(sCtx, kafkaClient, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == kafka.PlayerEventConfirmationEmail &&
				msg.Player.Email == registrationData.emailVerificationRequest.Body.Contact
		})

	})

	// Шаг 13: Подтверждение email
	sCtx.WithNewStep("Подтверждение email", func(sCtx provider.StepCtx) {
		emailContext, _ := registrationData.emailConfirmationMessage.GetConfirmationContext()
		confirmEmailReq := &clientTypes.Request[publicModels.ConfirmContactRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", registrationData.authorizationResponse.Body.Token),
			},
			Body: &publicModels.ConfirmContactRequestBody{
				Contact: registrationData.emailVerificationRequest.Body.Contact,
				Type:    publicModels.ContactTypeEmail,
				Code:    emailContext.ConfirmationCode,
			},
		}

		resp := publicClient.ConfirmContact(sCtx, confirmEmailReq)
		sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Подтверждение email")
	})

	// Шаг 14: Установка лимита на одиночную ставку
	sCtx.WithNewStep("Установка лимита на одиночную ставку", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[publicModels.SetSingleBetLimitRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", registrationData.authorizationResponse.Body.Token),
			},
			Body: &publicModels.SetSingleBetLimitRequestBody{
				Amount:   "100",
				Currency: config.Node.DefaultCurrency,
			},
		}

		singleBetLimitResp := publicClient.SetSingleBetLimit(sCtx, req)
		sCtx.Require().Equal(http.StatusCreated, singleBetLimitResp.StatusCode, "Установка лимита на одиночную ставку")
	})

	// Шаг 15: Установка лимита на оборот средств
	sCtx.WithNewStep("Установка лимита на оборот средств", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[publicModels.SetTurnoverLimitRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", registrationData.authorizationResponse.Body.Token),
			},
			Body: &publicModels.SetTurnoverLimitRequestBody{
				Amount:    "100",
				Currency:  config.Node.DefaultCurrency,
				Type:      publicModels.LimitPeriodDaily,
				StartedAt: int(time.Now().Unix()),
			},
		}

		resp := publicClient.SetTurnoverLimit(sCtx, req)
		sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Установка лимита на оборот средств")
	})

	// Шаг 16: Получение WalletUUID из Redis
	sCtx.WithNewStep("Получение WalletUUID из Redis", func(sCtx provider.StepCtx) {
		var wallets redis.WalletsMap
		err := redisPlayerClient.GetWithRetry(sCtx, registrationData.registrationMessage.Player.ExternalID, &wallets)

		registrationData.wallets = wallets
		sCtx.Require().NoError(err, "Значение кошелька получено из Redis")
	})

	// Шаг 17: Создание депозита (если сумма > 0)
	sCtx.WithNewStep("Создание депозита", func(sCtx provider.StepCtx) {
		if depositAmount > 0 {
			time.Sleep(5 * time.Second)

			req := &clientTypes.Request[publicModels.DepositRequestBody]{
				Headers: map[string]string{
					"Authorization": fmt.Sprintf("Bearer %s", registrationData.authorizationResponse.Body.Token),
				},
				Body: &publicModels.DepositRequestBody{
					Amount:          fmt.Sprintf("%.0f", depositAmount),
					PaymentMethodID: int(publicModels.Fake),
					Currency:        config.Node.DefaultCurrency,
					Country:         config.Node.DefaultCountry,
					Redirect: publicModels.DepositRedirectURLs{
						Failed:  publicModels.DepositRedirectURLFailed,
						Success: publicModels.DepositRedirectURLSuccess,
						Pending: publicModels.DepositRedirectURLPending,
					},
				},
			}

			resp := publicClient.CreateDeposit(sCtx, req)
			sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Депозит успешно создан")
		}
	})
	// Шаг 18: Получение агрегата кошелька из Redis
	if depositAmount > 0 {
		// Получение события депозита из NATS
		sCtx.WithNewStep("Получение события депозита из NATS", func(sCtx provider.StepCtx) {
			subject := fmt.Sprintf("%s.wallet.*.%s.*", config.Nats.StreamPrefix,
				registrationData.registrationMessage.Player.ExternalID)

			registrationData.depositEvent = nats.FindMessageInStream(
				sCtx, natsClient, subject, func(payload nats.DepositedMoneyPayload, msgType string) bool {
					return msgType == string(nats.DepositedMoneyType) &&
						payload.Amount == fmt.Sprintf("%.0f", depositAmount) &&
						payload.CurrencyCode == config.Node.DefaultCurrency
				})

			sCtx.Require().NotEmpty(registrationData.depositEvent, "Событие депозита получено из NATS")
		})

		// Получение обновленных данных кошелька из Redis
		sCtx.WithNewStep("Получение обновленных данных кошелька из Redis", func(sCtx provider.StepCtx) {
			var walletUUID string
			for _, wallet := range registrationData.wallets {
				walletUUID = wallet.WalletUUID
				break
			}

			var updatedWalletData redis.WalletFullData
			err := redisWalletClient.GetWithSeqCheck(
				sCtx,
				walletUUID,
				&updatedWalletData,
				int(registrationData.depositEvent.Sequence))
			sCtx.Require().NoError(err, "Получены обновленные данные кошелька из Redis")

			playerData = PlayerData{
				Auth:       registrationData.authorizationResponse,
				WalletData: updatedWalletData,
			}
		})
		return playerData
	} else {
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
				&updatedWalletData,
			)
			sCtx.Require().NoError(err, "Получены обновленные данные кошелька из Redis")

			playerData = PlayerData{
				Auth:       registrationData.authorizationResponse,
				WalletData: updatedWalletData,
			}
		})
		return playerData
	}
}
