package cap

import (
	"net/http"

	httpClient "CB_auto/internal/client"
	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/types"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

func (c *capClient) GetCapBrand(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[models.GetCapBrandResponseBody] {
	req.Method = http.MethodGet
	req.Path = "/_cap/api/v1/brands/{id}"
	return httpClient.DoRequest[struct{}, models.GetCapBrandResponseBody](sCtx, c.client, req)
}

func (c *capClient) DeleteCapBrand(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[struct{}] {
	req.Method = http.MethodDelete
	req.Path = "/_cap/api/v1/brands/{id}"
	return httpClient.DoRequest[struct{}, struct{}](sCtx, c.client, req)
}

func (c *capClient) CreateCapBrand(sCtx provider.StepCtx, req *types.Request[models.CreateCapBrandRequestBody]) *types.Response[models.CreateCapBrandResponseBody] {
	req.Method = http.MethodPost
	req.Path = "/_cap/api/v1/brands"
	return httpClient.DoRequest[models.CreateCapBrandRequestBody, models.CreateCapBrandResponseBody](sCtx, c.client, req)
}

func (c *capClient) UpdateBrandStatus(sCtx provider.StepCtx, req *types.Request[models.UpdateBrandStatusRequestBody]) *types.Response[struct{}] {
	req.Method = http.MethodPatch
	req.Path = "/_cap/api/v1/brands/{id}/status"
	return httpClient.DoRequest[models.UpdateBrandStatusRequestBody, struct{}](sCtx, c.client, req)
}

func (c *capClient) UpdateCapBrand(sCtx provider.StepCtx, req *types.Request[models.UpdateCapBrandRequestBody]) *types.Response[models.UpdateCapBrandResponseBody] {
	req.Method = http.MethodPatch
	req.Path = "/_cap/api/v1/brands/{id}"
	return httpClient.DoRequest[models.UpdateCapBrandRequestBody, models.UpdateCapBrandResponseBody](sCtx, c.client, req)
}

func (c *capClient) UpdateCapBrandError(sCtx provider.StepCtx, req *types.Request[models.UpdateCapBrandRequestBody]) *types.Response[models.ErrorResponse] {
	req.Method = http.MethodPatch
	req.Path = "/_cap/api/v1/brands/{id}"
	return httpClient.DoRequest[models.UpdateCapBrandRequestBody, models.ErrorResponse](sCtx, c.client, req)
}

func (c *capClient) GetCapCategory(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[models.GetCapCategoryResponseBody] {
	req.Method = http.MethodGet
	req.Path = "/_cap/api/v1/categories/{id}"
	return httpClient.DoRequest[struct{}, models.GetCapCategoryResponseBody](sCtx, c.client, req)
}

func (c *capClient) CreateCapCategory(sCtx provider.StepCtx, req *types.Request[models.CreateCapCategoryRequestBody]) *types.Response[models.CreateCapCategoryResponseBody] {
	req.Method = http.MethodPost
	req.Path = "/_cap/api/v1/categories"
	return httpClient.DoRequest[models.CreateCapCategoryRequestBody, models.CreateCapCategoryResponseBody](sCtx, c.client, req)
}

func (c *capClient) DeleteCapCategory(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[struct{}] {
	req.Method = http.MethodDelete
	req.Path = "/_cap/api/v1/categories/{id}"
	return httpClient.DoRequest[struct{}, struct{}](sCtx, c.client, req)
}

func (c *capClient) UpdateCapCategory(sCtx provider.StepCtx, req *types.Request[models.UpdateCapCategoryRequestBody]) *types.Response[models.UpdateCapCategoryResponseBody] {
	req.Method = http.MethodPatch
	req.Path = "/_cap/api/v1/categories/{id}"
	return httpClient.DoRequest[models.UpdateCapCategoryRequestBody, models.UpdateCapCategoryResponseBody](sCtx, c.client, req)
}

func (c *capClient) UpdateCapCategoryError(sCtx provider.StepCtx, req *types.Request[models.UpdateCapCategoryRequestBody]) *types.Response[models.ErrorResponse] {
	req.Method = http.MethodPatch
	req.Path = "/_cap/api/v1/categories/{id}"
	return httpClient.DoRequest[models.UpdateCapCategoryRequestBody, models.ErrorResponse](sCtx, c.client, req)
}

func (c *capClient) UpdateCapCollectionStatus(sCtx provider.StepCtx, req *types.Request[models.UpdateCapCollectionStatusRequestBody]) *types.Response[models.UpdateCapCollectionStatusResponseBody] {
	req.Method = http.MethodPatch
	req.Path = "/_cap/api/v1/categories/{id}/status"
	return httpClient.DoRequest[models.UpdateCapCollectionStatusRequestBody, models.UpdateCapCollectionStatusResponseBody](sCtx, c.client, req)
}

func (c *capClient) UpdateCapCategoryStatus(sCtx provider.StepCtx, req *types.Request[models.UpdateCapCategoryStatusRequestBody]) *types.Response[models.UpdateCapCategoryStatusResponseBody] {
	req.Method = http.MethodPatch
	req.Path = "/_cap/api/v1/categories/{id}/status"
	return httpClient.DoRequest[models.UpdateCapCategoryStatusRequestBody, models.UpdateCapCategoryStatusResponseBody](sCtx, c.client, req)
}
