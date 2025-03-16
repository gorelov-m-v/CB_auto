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
		depositEvent             *nats.NatsMessage[nats.DepositedMoneyPayload]
		registrationResponse     *clientTypes.Response[publicModels.FastRegistrationResponseBody]
		authorizationResponse    *clientTypes.Response[publicModels.TokenCheckResponseBody]
		verifications            *clientTypes.Response[[]publicModels.VerificationStatusResponseItem]
		phoneVerificationRequest *clientTypes.Request[publicModels.RequestVerificationRequestBody]
		emailVerificationRequest *clientTypes.Request[publicModels.RequestVerificationRequestBody]
		phoneConfirmMsg          kafka.PlayerMessage
		emailConfirmMsg          kafka.PlayerMessage
		wallets                  redis.WalletsMap
		depositeEvent            *nats.NatsMessage[nats.DepositedMoneyPayload]
	}

	playerData := PlayerData{}

	// Шаг 1: Регистрация игрока
	sCtx.WithNewStep("Создание и верификация игрока", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[publicModels.FastRegistrationRequestBody]{
			Body: &publicModels.FastRegistrationRequestBody{
				Country:  config.Node.DefaultCountry,
				Currency: config.Node.DefaultCurrency,
			},
		}

		registrationData.registrationResponse = publicClient.FastRegistration(sCtx, req)
		sCtx.Require().Equal(http.StatusOK, registrationData.registrationResponse.StatusCode, "Успешная регистрация")
	})

	// Шаг 2: Получение Kafka-сообщения о регистрации
	sCtx.WithNewStep("Получение Kafka-сообщения о регистрации", func(sCtx provider.StepCtx) {
		registrationData.registrationMessage = kafka.FindMessageByFilter(sCtx, kafkaClient, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == string(kafka.PlayerEventSignUpFast) &&
				msg.Player.AccountID == registrationData.registrationResponse.Body.Username
		})
		sCtx.Require().NotEmpty(registrationData.registrationMessage.Player.ID, "ID игрока не пустой")
	})

	// Шаг 3: Авторизация
	sCtx.WithNewStep("Авторизация", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[publicModels.TokenCheckRequestBody]{
			Body: &publicModels.TokenCheckRequestBody{
				Username: registrationData.registrationResponse.Body.Username,
				Password: registrationData.registrationResponse.Body.Password,
			},
		}

		registrationData.authorizationResponse = publicClient.TokenCheck(sCtx, req)
		sCtx.Require().Equal(http.StatusOK, registrationData.authorizationResponse.StatusCode, "Успешная авторизация")
	})

	// Шаг 4: Обновление данных игрока
	sCtx.WithNewStep("Обновление данных игрока", func(sCtx provider.StepCtx) {
		req := &clientTypes.Request[publicModels.UpdatePlayerRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", registrationData.authorizationResponse.Body.Token),
			},
			Body: &publicModels.UpdatePlayerRequestBody{
				FirstName:        "Test",
				LastName:         "Test",
				Gender:           1,
				City:             "TestCity",
				Postcode:         "12345",
				PermanentAddress: "Test Address",
				PersonalID:       utils.Get(utils.PERSONAL_ID),
				Profession:       "QA",
				IBAN:             utils.Get(utils.IBAN),
				Birthday:         "1980-01-01",
				Country:          config.Node.DefaultCountry,
			},
		}

		resp := publicClient.UpdatePlayer(sCtx, req)
		sCtx.Require().Equal(http.StatusOK, resp.StatusCode, "Обновление данных игрока")
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

	// Шаг 8: Запрос верификации телефона
	sCtx.WithNewStep("Запрос верификации телефона", func(sCtx provider.StepCtx) {
		registrationData.phoneVerificationRequest = &clientTypes.Request[publicModels.RequestVerificationRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", registrationData.authorizationResponse.Body.Token),
			},
			Body: &publicModels.RequestVerificationRequestBody{
				Contact: utils.Get(utils.PHONE),
				Type:    publicModels.ContactTypePhone,
			},
		}

		resp := publicClient.RequestContactVerification(sCtx, registrationData.phoneVerificationRequest)
		sCtx.Require().Equal(http.StatusOK, resp.StatusCode, "Запрос верификации телефона")
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

	// Шаг 10: Получение сообщения о подтверждении телефона и email
	sCtx.WithNewStep("Получение сообщения о подтверждении телефона и email", func(sCtx provider.StepCtx) {
		phoneNumberWithoutPlus := strings.TrimPrefix(registrationData.phoneVerificationRequest.Body.Contact, "+")

		registrationData.phoneConfirmMsg = kafka.FindMessageByFilter(sCtx, kafkaClient, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == string(kafka.PlayerEventConfirmationPhone) &&
				msg.Player.AccountID == registrationData.registrationResponse.Body.Username &&
				msg.Player.Phone == phoneNumberWithoutPlus
		})

		registrationData.emailConfirmMsg = kafka.FindMessageByFilter(sCtx, kafkaClient, func(msg kafka.PlayerMessage) bool {
			return msg.Message.EventType == string(kafka.PlayerEventConfirmationEmail) &&
				msg.Player.AccountID == registrationData.registrationResponse.Body.Username &&
				msg.Player.Email == registrationData.emailVerificationRequest.Body.Contact
		})

	})

	// Шаг 12: Подтверждение телефона
	sCtx.WithNewStep("Подтверждение телефона", func(sCtx provider.StepCtx) {
		phoneContext, _ := registrationData.phoneConfirmMsg.GetConfirmationContext()
		req := &clientTypes.Request[publicModels.ConfirmContactRequestBody]{
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", registrationData.authorizationResponse.Body.Token),
			},
			Body: &publicModels.ConfirmContactRequestBody{
				Contact: registrationData.phoneVerificationRequest.Body.Contact,
				Type:    publicModels.ContactTypePhone,
				Code:    phoneContext.ConfirmationCode,
			},
		}

		resp := publicClient.ConfirmContact(sCtx, req)
		sCtx.Require().Equal(http.StatusCreated, resp.StatusCode, "Подтверждение телефона")
	})

	// Шаг 13: Подтверждение email
	sCtx.WithNewStep("Подтверждение email", func(sCtx provider.StepCtx) {
		emailContext, _ := registrationData.emailConfirmMsg.GetConfirmationContext()
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
				StartedAt: time.Now().Unix(),
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

	// Шаг 18: Получение события депозита из NATS
	sCtx.WithNewStep("Получение события депозита из NATS", func(sCtx provider.StepCtx) {
		subject := fmt.Sprintf("%s.wallet.*.%s.*", config.Nats.StreamPrefix,
			registrationData.registrationMessage.Player.ExternalID)

		registrationData.depositEvent = nats.FindMessageInStream(
			sCtx, natsClient, subject, func(payload nats.DepositedMoneyPayload, msgType string) bool {
				return msgType == string(nats.DepositedMoney) &&
					payload.Amount == fmt.Sprintf("%.0f", depositAmount) &&
					payload.CurrencyCode == config.Node.DefaultCurrency
			})

		sCtx.Require().NotEmpty(registrationData.depositEvent, "Событие депозита получено из NATS")
	})

	// Шаг 19: Получение обновленных данных кошелька из Redis
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

	// Возвращаем информацию для авторизации и данные кошелька
	return playerData
}
