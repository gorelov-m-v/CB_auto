package public

import (
	"net/http"

	httpClient "CB_auto/internal/client"
	"CB_auto/internal/client/public/models"
	"CB_auto/internal/client/types"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

func (c *publicClient) GetWallets(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetWalletsResponseBody] {
	req.Method = http.MethodGet
	req.Path = "/_front_api/api/v1/wallets"
	return httpClient.DoRequest[any, models.GetWalletsResponseBody](sCtx, c.client, req)
}

func (c *publicClient) CreateWallet(sCtx provider.StepCtx, req *types.Request[models.CreateWalletRequestBody]) *types.Response[models.CreateWalletResponseBody] {
	req.Method = http.MethodPost
	req.Path = "/_front_api/api/v1/wallets"
	return httpClient.DoRequest[models.CreateWalletRequestBody, models.CreateWalletResponseBody](sCtx, c.client, req)
}

func (c *publicClient) SwitchWallet(sCtx provider.StepCtx, req *types.Request[models.SwitchWalletRequestBody]) *types.Response[struct{}] {
	req.Method = http.MethodPut
	req.Path = "/_front_api/api/v1/wallets/switch"
	return httpClient.DoRequest[models.SwitchWalletRequestBody, struct{}](sCtx, c.client, req)
}

func (c *publicClient) RemoveWallet(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[struct{}] {
	req.Method = http.MethodDelete
	req.Path = "/_front_api/api/v1/wallets/remove"
	return httpClient.DoRequest[any, struct{}](sCtx, c.client, req)
}
