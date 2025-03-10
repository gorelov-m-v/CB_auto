package public

import (
	"net/http"

	httpClient "CB_auto/internal/client"
	"CB_auto/internal/client/public/models"
	"CB_auto/internal/client/types"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

func (c *publicClient) FastRegistration(sCtx provider.StepCtx, req *types.Request[models.FastRegistrationRequestBody]) *types.Response[models.FastRegistrationResponseBody] {
	req.Method = http.MethodPost
	req.Path = "/_front_api/api/v1/registration/fast"
	return httpClient.DoRequest[models.FastRegistrationRequestBody, models.FastRegistrationResponseBody](sCtx, c.client, req)
}

func (c *publicClient) TokenCheck(sCtx provider.StepCtx, req *types.Request[models.TokenCheckRequestBody]) *types.Response[models.TokenCheckResponseBody] {
	req.Method = http.MethodPost
	req.Path = "/_front_api/api/token/check"
	return httpClient.DoRequest[models.TokenCheckRequestBody, models.TokenCheckResponseBody](sCtx, c.client, req)
}

func (c *publicClient) UpdatePlayer(sCtx provider.StepCtx, req *types.Request[models.UpdatePlayerRequestBody]) *types.Response[models.UpdatePlayerResponseBody] {
	req.Method = http.MethodPut
	req.Path = "/_front_api/api/v1/player"
	return httpClient.DoRequest[models.UpdatePlayerRequestBody, models.UpdatePlayerResponseBody](sCtx, c.client, req)
}

func (c *publicClient) VerifyIdentity(sCtx provider.StepCtx, req *types.Request[models.VerifyIdentityRequestBody]) *types.Response[struct{}] {
	req.Method = http.MethodPost
	req.Path = "/_front_api/api/v1/player/verification/identity"

	req.SetFormField("number", req.Body.Number)
	req.SetFormField("type", string(req.Body.Type))
	if req.Body.IssuedDate != "" {
		req.SetFormField("issuedDate", req.Body.IssuedDate)
	}
	if req.Body.ExpiryDate != "" {
		req.SetFormField("expiryDate", req.Body.ExpiryDate)
	}
	return httpClient.DoRequest[models.VerifyIdentityRequestBody, struct{}](sCtx, c.client, req)
}

func (c *publicClient) GetVerificationStatus(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[[]models.VerificationStatusResponseItem] {
	req.Method = http.MethodGet
	req.Path = "/_front_api/api/v1/player/verification/status"
	return httpClient.DoRequest[any, []models.VerificationStatusResponseItem](sCtx, c.client, req)
}

func (c *publicClient) RequestContactVerification(sCtx provider.StepCtx, req *types.Request[models.RequestVerificationRequestBody]) *types.Response[struct{}] {
	req.Method = http.MethodPost
	req.Path = "/_front_api/api/v1/contacts/request-verification"
	return httpClient.DoRequest[models.RequestVerificationRequestBody, struct{}](sCtx, c.client, req)
}

func (c *publicClient) ConfirmContact(sCtx provider.StepCtx, req *types.Request[models.ConfirmContactRequestBody]) *types.Response[struct{}] {
	req.Method = http.MethodPost
	req.Path = "/_front_api/api/v1/contacts"
	return httpClient.DoRequest[models.ConfirmContactRequestBody, struct{}](sCtx, c.client, req)
}
