package public

import (
	httpClient "CB_auto/internal/client"
	"CB_auto/internal/client/public/models"
	"fmt"
	"log"
	"net/http"
)

type PublicAPI interface {
	FastRegistration(req *httpClient.Request[models.FastRegistrationRequestBody]) *httpClient.Response[models.FastRegistrationResponseBody]
	TokenCheck(req *httpClient.Request[models.TokenCheckRequestBody]) *httpClient.Response[models.TokenCheckResponseBody]
	GetWallets(req *httpClient.Request[any]) *httpClient.Response[models.GetWalletsResponseBody]
	CreateWallet(req *httpClient.Request[models.CreateWalletRequestBody]) *httpClient.Response[models.CreateWalletResponseBody]
}

type publicClient struct {
	client *httpClient.Client
}

func NewPublicClient(client *httpClient.Client) PublicAPI {
	return &publicClient{client: client}
}

func (c *publicClient) FastRegistration(req *httpClient.Request[models.FastRegistrationRequestBody]) *httpClient.Response[models.FastRegistrationResponseBody] {
	req.Method = "POST"
	req.Path = "/_front_api/api/v1/registration/fast"
	resp, err := httpClient.DoRequest[models.FastRegistrationRequestBody, models.FastRegistrationResponseBody](c.client, req)
	if err != nil {
		log.Printf("FastRegistration failed: %v", err)
		return &httpClient.Response[models.FastRegistrationResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &httpClient.ErrorResponse{
				Body: fmt.Sprintf("FastRegistration failed: %v", err),
			},
		}
	}
	return resp
}

func (c *publicClient) TokenCheck(req *httpClient.Request[models.TokenCheckRequestBody]) *httpClient.Response[models.TokenCheckResponseBody] {
	req.Method = "POST"
	req.Path = "/_front_api/api/token/check"
	resp, err := httpClient.DoRequest[models.TokenCheckRequestBody, models.TokenCheckResponseBody](c.client, req)
	if err != nil {
		log.Printf("TokenCheck failed: %v", err)
		return &httpClient.Response[models.TokenCheckResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &httpClient.ErrorResponse{
				Body: fmt.Sprintf("TokenCheck failed: %v", err),
			},
		}
	}
	return resp
}

func (c *publicClient) GetWallets(req *httpClient.Request[any]) *httpClient.Response[models.GetWalletsResponseBody] {
	req.Method = "GET"
	req.Path = "/_front_api/api/v1/wallets"
	resp, err := httpClient.DoRequest[any, models.GetWalletsResponseBody](c.client, req)
	if err != nil {
		log.Printf("GetWallets failed: %v", err)
		return &httpClient.Response[models.GetWalletsResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &httpClient.ErrorResponse{
				Body: fmt.Sprintf("GetWallets failed: %v", err),
			},
		}
	}
	return resp
}

func (c *publicClient) CreateWallet(req *httpClient.Request[models.CreateWalletRequestBody]) *httpClient.Response[models.CreateWalletResponseBody] {
	req.Method = "POST"
	req.Path = "/_front_api/api/v1/wallets"
	resp, err := httpClient.DoRequest[models.CreateWalletRequestBody, models.CreateWalletResponseBody](c.client, req)
	if err != nil {
		log.Printf("CreateWallet failed: %v", err)
		return &httpClient.Response[models.CreateWalletResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &httpClient.ErrorResponse{
				Body: fmt.Sprintf("CreateWallet failed: %v", err),
			},
		}
	}
	return resp
}
