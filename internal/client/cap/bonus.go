package cap

import (
	"net/http"

	httpClient "CB_auto/internal/client"
	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/types"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

func (c *capClient) CreateLabel(sCtx provider.StepCtx, req *types.Request[models.CreateLabelRequestBody]) *types.Response[models.CreateLabelResponseBody] {
	req.Method = http.MethodPost
	req.Path = "/_cap/bonus/api/v1/label/create"
	return httpClient.DoRequest[models.CreateLabelRequestBody, models.CreateLabelResponseBody](sCtx, c.client, req)
}

func (c *capClient) GetLabel(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[models.GetLabelResponseBody] {
	req.Method = http.MethodGet
	req.Path = "/_cap/bonus/api/v1/label/show/{labelUUID}"
	return httpClient.DoRequest[struct{}, models.GetLabelResponseBody](sCtx, c.client, req)
}
