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
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

func CreateVerifiedPlayer(
	sCtx provider.StepCtx,
	publicClient publicAPI.PublicAPI,
	capClient capAPI.CapAPI,
	kafkaClient *kafka.Kafka,
	config *config.Config,
) *clientTypes.Response[publicModels.TokenCheckResponseBody] {

	// Шаг 1: Регистрация игрока
	registerReq := &clientTypes.Request[publicModels.FastRegistrationRequestBody]{
		Body: &publicModels.FastRegistrationRequestBody{
			Country:  config.Node.DefaultCountry,
			Currency: config.Node.DefaultCurrency,
		},
	}

	registrationResponse := publicClient.FastRegistration(sCtx, registerReq)
	sCtx.Require().Equal(http.StatusOK, registrationResponse.StatusCode, "Успешная регистрация")

	// Шаг 2: Получение Kafka-сообщения о регистрации
	registrationMessage := kafka.FindMessageByFilter[kafka.PlayerMessage](sCtx, kafkaClient, func(msg kafka.PlayerMessage) bool {
		return msg.Message.EventType == string(kafka.PlayerEventSignUpFast) &&
			msg.Player.AccountID == registrationResponse.Body.Username
	})
	sCtx.Require().NotEmpty(registrationMessage.Player.ID, "ID игрока не пустой")

	// Шаг 3: Авторизация
	authReq := &clientTypes.Request[publicModels.TokenCheckRequestBody]{
		Body: &publicModels.TokenCheckRequestBody{
			Username: registrationResponse.Body.Username,
			Password: registrationResponse.Body.Password,
		},
	}

	authorizationResponse := publicClient.TokenCheck(sCtx, authReq)
	sCtx.Require().Equal(http.StatusOK, authorizationResponse.StatusCode, "Успешная авторизация")

	// Шаг 4: Обновление данных игрока
	updateReq := &clientTypes.Request[publicModels.UpdatePlayerRequestBody]{
		Headers: map[string]string{
			"Authorization":   fmt.Sprintf("Bearer %s", authorizationResponse.Body.Token),
			"Content-Type":    "application/json",
			"Platform-Locale": "en",
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

	updateResponse := publicClient.UpdatePlayer(sCtx, updateReq)
	sCtx.Require().Equal(http.StatusOK, updateResponse.StatusCode, "Обновление данных игрока")

	// Шаг 5: Верификация идентичности
	verifyReq := &clientTypes.Request[publicModels.VerifyIdentityRequestBody]{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", authorizationResponse.Body.Token),
		},
		Body: &publicModels.VerifyIdentityRequestBody{
			Number:     "305003277",
			Type:       publicModels.VerificationTypeIdentity,
			IssuedDate: "1421463275.791",
			ExpiryDate: "1921463275.791",
		},
	}

	verifyResponse := publicClient.VerifyIdentity(sCtx, verifyReq)
	sCtx.Require().Equal(http.StatusCreated, verifyResponse.StatusCode, "Верификация идентичности")

	// Шаг 6: Получение статуса верификации
	statusReq := &clientTypes.Request[any]{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", authorizationResponse.Body.Token),
		},
	}

	verificationStatus := publicClient.GetVerificationStatus(sCtx, statusReq)
	sCtx.Require().Equal(http.StatusOK, verificationStatus.StatusCode, "Получение статуса верификации")

	// Шаг 7: Обновление статуса верификации
	updateStatusReq := &clientTypes.Request[capModels.UpdateVerificationStatusRequestBody]{
		Headers: map[string]string{
			"Authorization":   fmt.Sprintf("Bearer %s", capClient.GetToken(sCtx)),
			"Platform-NodeID": config.Node.ProjectID,
		},
		PathParams: map[string]string{
			"verification_id": verificationStatus.Body[0].DocumentID,
		},
		Body: &capModels.UpdateVerificationStatusRequestBody{
			Note:   "",
			Reason: "",
			Status: capModels.VerificationStatusApproved,
		},
	}

	updateStatusResp := capClient.UpdateVerificationStatus(sCtx, updateStatusReq)
	sCtx.Require().Equal(http.StatusNoContent, updateStatusResp.StatusCode, "Обновление статуса верификации")

	// Шаг 8: Запрос верификации телефона
	phoneReq := &clientTypes.Request[publicModels.RequestVerificationRequestBody]{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", authorizationResponse.Body.Token),
		},
		Body: &publicModels.RequestVerificationRequestBody{
			Contact: utils.Get(utils.PHONE),
			Type:    publicModels.ContactTypePhone,
		},
	}

	phoneVerifyResp := publicClient.RequestContactVerification(sCtx, phoneReq)
	sCtx.Require().Equal(http.StatusOK, phoneVerifyResp.StatusCode, "Запрос верификации телефона")

	// Шаг 9: Запрос верификации email
	emailReq := &clientTypes.Request[publicModels.RequestVerificationRequestBody]{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", authorizationResponse.Body.Token),
		},
		Body: &publicModels.RequestVerificationRequestBody{
			Contact: fmt.Sprintf("test%d@example.com", time.Now().Unix()),
			Type:    publicModels.ContactTypeEmail,
		},
	}

	emailVerifyResp := publicClient.RequestContactVerification(sCtx, emailReq)
	sCtx.Require().Equal(http.StatusOK, emailVerifyResp.StatusCode, "Запрос верификации email")

	// Шаг 10: Получение сообщения о подтверждении телефона
	phoneNumberWithoutPlus := strings.TrimPrefix(phoneReq.Body.Contact, "+")
	phoneConfirmMsg := kafka.FindMessageByFilter[kafka.PlayerMessage](sCtx, kafkaClient, func(msg kafka.PlayerMessage) bool {
		return msg.Message.EventType == string(kafka.PlayerEventConfirmationPhone) &&
			msg.Player.AccountID == registrationResponse.Body.Username &&
			msg.Player.Phone == phoneNumberWithoutPlus
	})

	// Шаг 11: Получение сообщения о подтверждении email
	emailConfirmMsg := kafka.FindMessageByFilter[kafka.PlayerMessage](sCtx, kafkaClient, func(msg kafka.PlayerMessage) bool {
		return msg.Message.EventType == string(kafka.PlayerEventConfirmationEmail) &&
			msg.Player.AccountID == registrationResponse.Body.Username &&
			msg.Player.Email == emailReq.Body.Contact
	})

	// Шаг 12: Подтверждение телефона
	phoneContext, err := phoneConfirmMsg.GetConfirmationContext()
	sCtx.Require().NoError(err, "Получение кода подтверждения телефона")

	confirmPhoneReq := &clientTypes.Request[publicModels.ConfirmContactRequestBody]{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", authorizationResponse.Body.Token),
		},
		Body: &publicModels.ConfirmContactRequestBody{
			Contact: phoneReq.Body.Contact,
			Type:    publicModels.ContactTypePhone,
			Code:    phoneContext.ConfirmationCode,
		},
	}

	confirmPhoneResp := publicClient.ConfirmContact(sCtx, confirmPhoneReq)
	sCtx.Require().Equal(http.StatusCreated, confirmPhoneResp.StatusCode, "Подтверждение телефона")

	// Шаг 13: Подтверждение email
	emailContext, err := emailConfirmMsg.GetConfirmationContext()
	sCtx.Require().NoError(err, "Получение кода подтверждения email")

	confirmEmailReq := &clientTypes.Request[publicModels.ConfirmContactRequestBody]{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", authorizationResponse.Body.Token),
		},
		Body: &publicModels.ConfirmContactRequestBody{
			Contact: emailReq.Body.Contact,
			Type:    publicModels.ContactTypeEmail,
			Code:    emailContext.ConfirmationCode,
		},
	}

	confirmEmailResp := publicClient.ConfirmContact(sCtx, confirmEmailReq)
	sCtx.Require().Equal(http.StatusCreated, confirmEmailResp.StatusCode, "Подтверждение email")

	// Шаг 14: Установка лимита на одиночную ставку

	setSingleBetLimitReq := &clientTypes.Request[publicModels.SetSingleBetLimitRequestBody]{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", authorizationResponse.Body.Token),
		},
		Body: &publicModels.SetSingleBetLimitRequestBody{
			Amount:   "100",
			Currency: config.Node.DefaultCurrency,
		},
	}

	singleBetLimitResp := publicClient.SetSingleBetLimit(sCtx, setSingleBetLimitReq)
	sCtx.Require().Equal(http.StatusCreated, singleBetLimitResp.StatusCode, "Установка лимита на одиночную ставку")

	// Шаг 15: Установка лимита на оборот средств
	setTurnoverLimitReq := &clientTypes.Request[publicModels.SetTurnoverLimitRequestBody]{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", authorizationResponse.Body.Token),
		},
		Body: &publicModels.SetTurnoverLimitRequestBody{
			Amount:    "100",
			Currency:  config.Node.DefaultCurrency,
			Type:      publicModels.LimitPeriodDaily,
			StartedAt: time.Now().Unix(),
		},
	}

	turnoverLimitResp := publicClient.SetTurnoverLimit(sCtx, setTurnoverLimitReq)
	sCtx.Require().Equal(http.StatusCreated, turnoverLimitResp.StatusCode, "Установка лимита на оборот средств")

	// Возвращаем информацию для авторизации
	return authorizationResponse
}
