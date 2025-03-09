package public

import (
	"net/http"

	httpClient "CB_auto/internal/client"
	"CB_auto/internal/client/public/models"
	"CB_auto/internal/client/types"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

func (c *publicClient) SetSingleBetLimit(sCtx provider.StepCtx, req *types.Request[models.SetSingleBetLimitRequestBody]) *types.Response[struct{}] {
	req.Method = http.MethodPost
	req.Path = "/_front_api/api/v1/player/single-limits/single-bet"
	return httpClient.DoRequest[models.SetSingleBetLimitRequestBody, struct{}](sCtx, c.client, req)
}

func (c *publicClient) SetCasinoLossLimit(sCtx provider.StepCtx, req *types.Request[models.SetCasinoLossLimitRequestBody]) *types.Response[struct{}] {
	req.Method = http.MethodPost
	req.Path = "/_front_api/api/v1/player/recalculated-limits/casino-loss"
	return httpClient.DoRequest[models.SetCasinoLossLimitRequestBody, struct{}](sCtx, c.client, req)
}

func (c *publicClient) GetTurnoverLimits(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetTurnoverLimitsResponseBody] {
	req.Method = http.MethodGet
	req.Path = "/_front_api/api/v1/player/recalculated-limits/turnover-of-funds"
	return httpClient.DoRequest[any, models.GetTurnoverLimitsResponseBody](sCtx, c.client, req)
}

func (c *publicClient) GetSingleBetLimits(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetSingleBetLimitsResponseBody] {
	req.Method = http.MethodGet
	req.Path = "/_front_api/api/v1/player/single-limits/single-bet"
	return httpClient.DoRequest[any, models.GetSingleBetLimitsResponseBody](sCtx, c.client, req)
}

func (c *publicClient) SetRestriction(sCtx provider.StepCtx, req *types.Request[models.SetRestrictionRequestBody]) *types.Response[models.SetRestrictionResponseBody] {
	req.Method = http.MethodPost
	req.Path = "/_front_api/api/v1/player/restrictions"
	return httpClient.DoRequest[models.SetRestrictionRequestBody, models.SetRestrictionResponseBody](sCtx, c.client, req)
}

func (c *publicClient) GetRestriction(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.SetRestrictionResponseBody] {
	req.Method = http.MethodGet
	req.Path = "/_front_api/api/v1/player/restrictions"
	return httpClient.DoRequest[any, models.SetRestrictionResponseBody](sCtx, c.client, req)
}

func (c *publicClient) GetCasinoLossLimits(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetCasinoLossLimitsResponseBody] {
	req.Method = http.MethodGet
	req.Path = "/_front_api/api/v1/player/recalculated-limits/casino-loss"
	return httpClient.DoRequest[any, models.GetCasinoLossLimitsResponseBody](sCtx, c.client, req)
}

func (c *publicClient) SetTurnoverLimit(sCtx provider.StepCtx, req *types.Request[models.SetTurnoverLimitRequestBody]) *types.Response[struct{}] {
	req.Method = http.MethodPost
	req.Path = "/_front_api/api/v1/player/recalculated-limits/turnover-of-funds"
	return httpClient.DoRequest[models.SetTurnoverLimitRequestBody, struct{}](sCtx, c.client, req)
}
