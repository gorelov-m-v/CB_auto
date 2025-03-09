package cap

import (
	"net/http"

	httpClient "CB_auto/internal/client"
	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/types"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

func (c *capClient) UpdateBlockers(sCtx provider.StepCtx, req *types.Request[models.BlockersRequestBody]) *types.Response[struct{}] {
	req.Method = http.MethodPatch
	req.Path = "/_cap/api/v1/players/{player_uuid}/blockers"
	return httpClient.DoRequest[models.BlockersRequestBody, struct{}](sCtx, c.client, req)
}

func (c *capClient) GetPlayerLimits(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetPlayerLimitsResponseBody] {
	req.Method = http.MethodGet
	req.Path = "/_cap/api/v1/player/{playerID}/limits"
	req.QueryParams = map[string]string{
		"sort":    "status",
		"page":    "1",
		"perPage": "10",
	}
	return httpClient.DoRequest[any, models.GetPlayerLimitsResponseBody](sCtx, c.client, req)
}

func (c *capClient) GetBlockers(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetBlockersResponseBody] {
	req.Method = http.MethodGet
	req.Path = "/_cap/api/v1/players/{player_uuid}/blockers"
	return httpClient.DoRequest[any, models.GetBlockersResponseBody](sCtx, c.client, req)
}

func (c *capClient) CreateBalanceAdjustment(sCtx provider.StepCtx, req *types.Request[models.CreateBalanceAdjustmentRequestBody]) *types.Response[struct{}] {
	req.Method = http.MethodPost
	req.Path = "/_cap/api/v1/wallet/{player_uuid}/create-balance-adjustment"
	return httpClient.DoRequest[models.CreateBalanceAdjustmentRequestBody, struct{}](sCtx, c.client, req)
}

func (c *capClient) CreateBlockAmount(sCtx provider.StepCtx, req *types.Request[models.CreateBlockAmountRequestBody]) *types.Response[models.CreateBlockAmountResponseBody] {
	req.Method = "POST"
	req.Path = "/_cap/api/v1/wallet/{player_uuid}/create-block-amount"
	return httpClient.DoRequest[models.CreateBlockAmountRequestBody, models.CreateBlockAmountResponseBody](sCtx, c.client, req)
}

func (c *capClient) GetBlockAmountList(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.BlockAmountListResponseBody] {
	req.Method = "GET"
	req.Path = "/_cap/api/v1/wallet/{player_uuid}/block-amount-list"
	return httpClient.DoRequest[any, models.BlockAmountListResponseBody](sCtx, c.client, req)
}

func (c *capClient) GetWalletList(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetWalletListResponseBody] {
	req.Method = "GET"
	req.Path = "/_cap/api/v2/wallet/{player_uuid}/list"
	return httpClient.DoRequest[any, models.GetWalletListResponseBody](sCtx, c.client, req)
}

func (c *capClient) DeleteBlockAmount(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[struct{}] {
	req.Method = "DELETE"
	req.Path = "/_cap/api/v1/wallet/delete-amount-block/{block_uuid}"
	return httpClient.DoRequest[any, struct{}](sCtx, c.client, req)
}
