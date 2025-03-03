package public

import (
	"log"

	httpClient "CB_auto/internal/client"
	"CB_auto/internal/client/public/models"
	"CB_auto/internal/client/types"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

type PublicAPI interface {
	FastRegistration(sCtx provider.StepCtx, req *types.Request[models.FastRegistrationRequestBody]) *types.Response[models.FastRegistrationResponseBody]
	TokenCheck(sCtx provider.StepCtx, req *types.Request[models.TokenCheckRequestBody]) *types.Response[models.TokenCheckResponseBody]
	GetWallets(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetWalletsResponseBody]
	CreateWallet(sCtx provider.StepCtx, req *types.Request[models.CreateWalletRequestBody]) *types.Response[models.CreateWalletResponseBody]
	SwitchWallet(sCtx provider.StepCtx, req *types.Request[models.SwitchWalletRequestBody]) *types.Response[struct{}]
	RemoveWallet(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[struct{}]
	SetSingleBetLimit(sCtx provider.StepCtx, req *types.Request[models.SetSingleBetLimitRequestBody]) *types.Response[struct{}]
	SetCasinoLossLimit(sCtx provider.StepCtx, req *types.Request[models.SetCasinoLossLimitRequestBody]) *types.Response[struct{}]
	GetTurnoverLimits(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetTurnoverLimitsResponseBody]
	GetSingleBetLimits(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetSingleBetLimitsResponseBody]
	SetRestriction(sCtx provider.StepCtx, req *types.Request[models.SetRestrictionRequestBody]) *types.Response[models.SetRestrictionResponseBody]
	GetRestriction(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.SetRestrictionResponseBody]
	GetCasinoLossLimits(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetCasinoLossLimitsResponseBody]
	SetTurnoverLimit(sCtx provider.StepCtx, req *types.Request[models.SetTurnoverLimitRequestBody]) *types.Response[struct{}]
	CreateDeposit(sCtx provider.StepCtx, req *types.Request[models.DepositRequestBody]) *types.Response[struct{}]
	UpdatePlayer(sCtx provider.StepCtx, req *types.Request[models.UpdatePlayerRequestBody]) *types.Response[models.UpdatePlayerResponseBody]
}

type publicClient struct {
	client *types.Client
}

func NewClient(baseClient *types.Client) PublicAPI {
	return &publicClient{client: baseClient}
}

func (c *publicClient) FastRegistration(sCtx provider.StepCtx, req *types.Request[models.FastRegistrationRequestBody]) *types.Response[models.FastRegistrationResponseBody] {
	req.Method = "POST"
	req.Path = "/_front_api/api/v1/registration/fast"
	resp, err := httpClient.DoRequest[models.FastRegistrationRequestBody, models.FastRegistrationResponseBody](sCtx, c.client, req)

	if err != nil {
		log.Printf("FastRegistration failed: %v", err)
	}

	return resp
}

func (c *publicClient) TokenCheck(sCtx provider.StepCtx, req *types.Request[models.TokenCheckRequestBody]) *types.Response[models.TokenCheckResponseBody] {
	req.Method = "POST"
	req.Path = "/_front_api/api/token/check"
	resp, err := httpClient.DoRequest[models.TokenCheckRequestBody, models.TokenCheckResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("TokenCheck failed: %v", err)
	}

	return resp
}

func (c *publicClient) GetWallets(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetWalletsResponseBody] {
	req.Method = "GET"
	req.Path = "/_front_api/api/v1/wallets"
	resp, err := httpClient.DoRequest[any, models.GetWalletsResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("GetWallets failed: %v", err)
	}

	return resp
}

func (c *publicClient) CreateWallet(sCtx provider.StepCtx, req *types.Request[models.CreateWalletRequestBody]) *types.Response[models.CreateWalletResponseBody] {
	req.Method = "POST"
	req.Path = "/_front_api/api/v1/wallets"
	resp, err := httpClient.DoRequest[models.CreateWalletRequestBody, models.CreateWalletResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("CreateWallet failed: %v", err)
	}

	return resp
}

func (c *publicClient) SwitchWallet(sCtx provider.StepCtx, req *types.Request[models.SwitchWalletRequestBody]) *types.Response[struct{}] {
	req.Method = "PUT"
	req.Path = "/_front_api/api/v1/wallets/switch"
	resp, err := httpClient.DoRequest[models.SwitchWalletRequestBody, struct{}](sCtx, c.client, req)
	if err != nil {
		log.Printf("SwitchWallet failed: %v", err)
	}

	return resp
}

func (c *publicClient) RemoveWallet(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[struct{}] {
	req.Method = "DELETE"
	req.Path = "/_front_api/api/v1/wallets/remove"
	resp, err := httpClient.DoRequest[any, struct{}](sCtx, c.client, req)
	if err != nil {
		log.Printf("RemoveWallet failed: %v", err)
	}

	return resp
}

func (c *publicClient) SetSingleBetLimit(sCtx provider.StepCtx, req *types.Request[models.SetSingleBetLimitRequestBody]) *types.Response[struct{}] {
	req.Method = "POST"
	req.Path = "/_front_api/api/v1/player/single-limits/single-bet"
	resp, err := httpClient.DoRequest[models.SetSingleBetLimitRequestBody, struct{}](sCtx, c.client, req)
	if err != nil {
		log.Printf("SetSingleBetLimit failed: %v", err)
	}

	return resp
}

func (c *publicClient) SetCasinoLossLimit(sCtx provider.StepCtx, req *types.Request[models.SetCasinoLossLimitRequestBody]) *types.Response[struct{}] {
	req.Method = "POST"
	req.Path = "/_front_api/api/v1/player/recalculated-limits/casino-loss"
	resp, err := httpClient.DoRequest[models.SetCasinoLossLimitRequestBody, struct{}](sCtx, c.client, req)
	if err != nil {
		log.Printf("SetCasinoLossLimit failed: %v", err)
	}

	return resp
}

func (c *publicClient) GetTurnoverLimits(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetTurnoverLimitsResponseBody] {
	req.Method = "GET"
	req.Path = "/_front_api/api/v1/player/recalculated-limits/turnover-of-funds"
	resp, err := httpClient.DoRequest[any, models.GetTurnoverLimitsResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("GetTurnoverLimits failed: %v", err)
	}

	return resp
}

func (c *publicClient) GetSingleBetLimits(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetSingleBetLimitsResponseBody] {
	req.Method = "GET"
	req.Path = "/_front_api/api/v1/player/single-limits/single-bet"
	resp, err := httpClient.DoRequest[any, models.GetSingleBetLimitsResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("GetSingleBetLimits failed: %v", err)
	}

	return resp
}

func (c *publicClient) SetRestriction(sCtx provider.StepCtx, req *types.Request[models.SetRestrictionRequestBody]) *types.Response[models.SetRestrictionResponseBody] {
	req.Method = "POST"
	req.Path = "/_front_api/api/v1/player/restrictions"
	resp, err := httpClient.DoRequest[models.SetRestrictionRequestBody, models.SetRestrictionResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("SetRestriction failed: %v", err)
	}

	return resp
}

func (c *publicClient) GetRestriction(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.SetRestrictionResponseBody] {
	req.Method = "GET"
	req.Path = "/_front_api/api/v1/player/restrictions"
	resp, err := httpClient.DoRequest[any, models.SetRestrictionResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("GetRestriction failed: %v", err)
	}

	return resp
}

func (c *publicClient) GetCasinoLossLimits(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetCasinoLossLimitsResponseBody] {
	req.Method = "GET"
	req.Path = "/_front_api/api/v1/player/recalculated-limits/casino-loss"
	resp, err := httpClient.DoRequest[any, models.GetCasinoLossLimitsResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("GetCasinoLossLimits failed: %v", err)
	}

	return resp
}

func (c *publicClient) SetTurnoverLimit(sCtx provider.StepCtx, req *types.Request[models.SetTurnoverLimitRequestBody]) *types.Response[struct{}] {
	req.Method = "POST"
	req.Path = "/_front_api/api/v1/player/recalculated-limits/turnover-of-funds"
	resp, err := httpClient.DoRequest[models.SetTurnoverLimitRequestBody, struct{}](sCtx, c.client, req)
	if err != nil {
		log.Printf("SetTurnoverLimit failed: %v", err)
	}

	return resp
}

func (c *publicClient) CreateDeposit(sCtx provider.StepCtx, req *types.Request[models.DepositRequestBody]) *types.Response[struct{}] {
	req.Method = "POST"
	req.Path = "/_front_api/api/v2/payment/deposit"
	resp, err := httpClient.DoRequest[models.DepositRequestBody, struct{}](sCtx, c.client, req)
	if err != nil {
		log.Printf("CreateDeposit failed: %v", err)
	}

	return resp
}

func (c *publicClient) UpdatePlayer(sCtx provider.StepCtx, req *types.Request[models.UpdatePlayerRequestBody]) *types.Response[models.UpdatePlayerResponseBody] {
	req.Method = "PUT"
	req.Path = "/_front_api/api/v1/player"
	resp, err := httpClient.DoRequest[models.UpdatePlayerRequestBody, models.UpdatePlayerResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("UpdatePlayer failed: %v", err)
	}

	return resp
}
