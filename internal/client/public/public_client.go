package public

import (
	httpClient "CB_auto/internal/client"
	"CB_auto/internal/client/public/models"
	"CB_auto/internal/client/types"
	"fmt"
	"log"
	"net/http"
)

type PublicAPI interface {
	FastRegistration(req *types.Request[models.FastRegistrationRequestBody]) *types.Response[models.FastRegistrationResponseBody]
	TokenCheck(req *types.Request[models.TokenCheckRequestBody]) *types.Response[models.TokenCheckResponseBody]
	GetWallets(req *types.Request[any]) *types.Response[models.GetWalletsResponseBody]
	CreateWallet(req *types.Request[models.CreateWalletRequestBody]) *types.Response[models.CreateWalletResponseBody]
	SwitchWallet(req *types.Request[models.SwitchWalletRequestBody]) *types.Response[struct{}]
	RemoveWallet(req *types.Request[any]) *types.Response[struct{}]
}

type publicClient struct {
	client *types.Client
}

func NewClient(baseClient *types.Client) PublicAPI {
	return &publicClient{client: baseClient}
}

func (c *publicClient) FastRegistration(req *types.Request[models.FastRegistrationRequestBody]) *types.Response[models.FastRegistrationResponseBody] {
	req.Method = "POST"
	req.Path = "/_front_api/api/v1/registration/fast"
	resp, err := httpClient.DoRequest[models.FastRegistrationRequestBody, models.FastRegistrationResponseBody](c.client, req)
	if err != nil {
		log.Printf("FastRegistration failed: %v", err)
		return &types.Response[models.FastRegistrationResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("FastRegistration failed: %v", err),
			},
		}
	}
	return resp
}

func (c *publicClient) TokenCheck(req *types.Request[models.TokenCheckRequestBody]) *types.Response[models.TokenCheckResponseBody] {
	req.Method = "POST"
	req.Path = "/_front_api/api/token/check"
	resp, err := httpClient.DoRequest[models.TokenCheckRequestBody, models.TokenCheckResponseBody](c.client, req)
	if err != nil {
		log.Printf("TokenCheck failed: %v", err)
		return &types.Response[models.TokenCheckResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("TokenCheck failed: %v", err),
			},
		}
	}
	return resp
}

func (c *publicClient) GetWallets(req *types.Request[any]) *types.Response[models.GetWalletsResponseBody] {
	req.Method = "GET"
	req.Path = "/_front_api/api/v1/wallets"
	resp, err := httpClient.DoRequest[any, models.GetWalletsResponseBody](c.client, req)
	if err != nil {
		log.Printf("GetWallets failed: %v", err)
		return &types.Response[models.GetWalletsResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("GetWallets failed: %v", err),
			},
		}
	}
	return resp
}

func (c *publicClient) CreateWallet(req *types.Request[models.CreateWalletRequestBody]) *types.Response[models.CreateWalletResponseBody] {
	req.Method = "POST"
	req.Path = "/_front_api/api/v1/wallets"
	resp, err := httpClient.DoRequest[models.CreateWalletRequestBody, models.CreateWalletResponseBody](c.client, req)
	if err != nil {
		log.Printf("CreateWallet failed: %v", err)
		return &types.Response[models.CreateWalletResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("CreateWallet failed: %v", err),
			},
		}
	}
	return resp
}

func (c *publicClient) SwitchWallet(req *types.Request[models.SwitchWalletRequestBody]) *types.Response[struct{}] {
	req.Method = "PUT"
	req.Path = "/_front_api/api/v1/wallets/switch"
	resp, err := httpClient.DoRequest[models.SwitchWalletRequestBody, struct{}](c.client, req)
	if err != nil {
		log.Printf("SwitchWallet failed: %v", err)
		return &types.Response[struct{}]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("SwitchWallet failed: %v", err),
			},
		}
	}
	return resp
}

func (c *publicClient) RemoveWallet(req *types.Request[any]) *types.Response[struct{}] {
	req.Method = "DELETE"
	req.Path = "/_front_api/api/v1/wallets/remove"
	resp, err := httpClient.DoRequest[any, struct{}](c.client, req)
	if err != nil {
		log.Printf("RemoveWallet failed: %v", err)
		return &types.Response[struct{}]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("RemoveWallet failed: %v", err),
			},
		}
	}
	return resp
}
