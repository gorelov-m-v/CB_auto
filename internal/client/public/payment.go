package public

import (
	"net/http"

	httpClient "CB_auto/internal/client"
	"CB_auto/internal/client/public/models"
	"CB_auto/internal/client/types"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

func (c *publicClient) CreateDeposit(sCtx provider.StepCtx, req *types.Request[models.DepositRequestBody]) *types.Response[struct{}] {
	req.Method = http.MethodPost
	req.Path = "/_front_api/api/v2/payment/deposit"
	return httpClient.DoRequest[models.DepositRequestBody, struct{}](sCtx, c.client, req)
}
