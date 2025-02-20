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
	SetSingleBetLimit(req *types.Request[models.SetSingleBetLimitRequestBody]) *types.Response[struct{}]
	SetCasinoLossLimit(req *types.Request[models.SetCasinoLossLimitRequestBody]) *types.Response[struct{}]
	GetTurnoverLimits(req *types.Request[any]) *types.Response[models.GetTurnoverLimitsResponseBody]
	GetSingleBetLimits(req *types.Request[any]) *types.Response[models.GetSingleBetLimitsResponseBody]
	SetRestriction(req *types.Request[models.SetRestrictionRequestBody]) *types.Response[models.SetRestrictionResponseBody]
	GetRestriction(req *types.Request[any]) *types.Response[models.SetRestrictionResponseBody]
	GetCasinoLossLimits(req *types.Request[any]) *types.Response[models.GetCasinoLossLimitsResponseBody]
	SetTurnoverLimit(req *types.Request[models.SetTurnoverLimitRequestBody]) *types.Response[struct{}]
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

func (c *publicClient) SetSingleBetLimit(req *types.Request[models.SetSingleBetLimitRequestBody]) *types.Response[struct{}] {
	req.Method = "POST"
	req.Path = "/_front_api/api/v1/player/single-limits/single-bet"
	resp, err := httpClient.DoRequest[models.SetSingleBetLimitRequestBody, struct{}](c.client, req)
	if err != nil {
		log.Printf("SetSingleBetLimit failed: %v", err)
		return &types.Response[struct{}]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("SetSingleBetLimit failed: %v", err),
			},
		}
	}
	return resp
}

func (c *publicClient) SetCasinoLossLimit(req *types.Request[models.SetCasinoLossLimitRequestBody]) *types.Response[struct{}] {
	req.Method = "POST"
	req.Path = "/_front_api/api/v1/player/recalculated-limits/casino-loss"
	resp, err := httpClient.DoRequest[models.SetCasinoLossLimitRequestBody, struct{}](c.client, req)
	if err != nil {
		log.Printf("SetCasinoLossLimit failed: %v", err)
		return &types.Response[struct{}]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("SetCasinoLossLimit failed: %v", err),
			},
		}
	}
	return resp
}

func (c *publicClient) GetTurnoverLimits(req *types.Request[any]) *types.Response[models.GetTurnoverLimitsResponseBody] {
	req.Method = "GET"
	req.Path = "/_front_api/api/v1/player/recalculated-limits/turnover-of-funds"
	resp, err := httpClient.DoRequest[any, models.GetTurnoverLimitsResponseBody](c.client, req)
	if err != nil {
		log.Printf("GetTurnoverLimits failed: %v", err)
		return &types.Response[models.GetTurnoverLimitsResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("GetTurnoverLimits failed: %v", err),
			},
		}
	}
	return resp
}

func (c *publicClient) GetSingleBetLimits(req *types.Request[any]) *types.Response[models.GetSingleBetLimitsResponseBody] {
	req.Method = "GET"
	req.Path = "/_front_api/api/v1/player/single-limits/single-bet"
	resp, err := httpClient.DoRequest[any, models.GetSingleBetLimitsResponseBody](c.client, req)
	if err != nil {
		log.Printf("GetSingleBetLimits failed: %v", err)
		return &types.Response[models.GetSingleBetLimitsResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("GetSingleBetLimits failed: %v", err),
			},
		}
	}
	return resp
}

func (c *publicClient) SetRestriction(req *types.Request[models.SetRestrictionRequestBody]) *types.Response[models.SetRestrictionResponseBody] {
	req.Method = "POST"
	req.Path = "/_front_api/api/v1/player/restrictions"
	resp, err := httpClient.DoRequest[models.SetRestrictionRequestBody, models.SetRestrictionResponseBody](c.client, req)
	if err != nil {
		log.Printf("SetRestriction failed: %v", err)
		return &types.Response[models.SetRestrictionResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("SetRestriction failed: %v", err),
			},
		}
	}
	return resp
}

func (c *publicClient) GetRestriction(req *types.Request[any]) *types.Response[models.SetRestrictionResponseBody] {
	req.Method = "GET"
	req.Path = "/_front_api/api/v1/player/restrictions"
	resp, err := httpClient.DoRequest[any, models.SetRestrictionResponseBody](c.client, req)
	if err != nil {
		log.Printf("GetRestriction failed: %v", err)
		return &types.Response[models.SetRestrictionResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("GetRestriction failed: %v", err),
			},
		}
	}
	return resp
}

func (c *publicClient) GetCasinoLossLimits(req *types.Request[any]) *types.Response[models.GetCasinoLossLimitsResponseBody] {
	req.Method = "GET"
	req.Path = "/_front_api/api/v1/player/recalculated-limits/casino-loss"
	resp, err := httpClient.DoRequest[any, models.GetCasinoLossLimitsResponseBody](c.client, req)
	if err != nil {
		log.Printf("GetCasinoLossLimits failed: %v", err)
		return &types.Response[models.GetCasinoLossLimitsResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("GetCasinoLossLimits failed: %v", err),
			},
		}
	}
	return resp
}

func (c *publicClient) SetTurnoverLimit(req *types.Request[models.SetTurnoverLimitRequestBody]) *types.Response[struct{}] {
	req.Method = "POST"
	req.Path = "/_front_api/api/v1/player/recalculated-limits/turnover-of-funds"
	resp, err := httpClient.DoRequest[models.SetTurnoverLimitRequestBody, struct{}](c.client, req)
	if err != nil {
		log.Printf("SetTurnoverLimit failed: %v", err)
		return &types.Response[struct{}]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("SetTurnoverLimit failed: %v", err),
			},
		}
	}
	return resp
}
