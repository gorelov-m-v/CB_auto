package cap

import (
	httpClient "CB_auto/internal/client"
	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"fmt"
	"log"
	"net/http"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

type CapAPI interface {
	GetToken(sCtx provider.StepCtx) string
	CheckAdmin(sCtx provider.StepCtx, req *types.Request[models.AdminCheckRequestBody]) *types.Response[models.AdminCheckResponseBody]
	CreateCapBrand(sCtx provider.StepCtx, req *types.Request[models.CreateCapBrandRequestBody]) *types.Response[models.CreateCapBrandResponseBody]
	GetCapBrand(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[models.GetCapBrandResponseBody]
	DeleteCapBrand(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[struct{}]
	UpdateBlockers(sCtx provider.StepCtx, req *types.Request[models.BlockersRequestBody]) *types.Response[struct{}]
	GetBlockers(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetBlockersResponseBody]
	GetPlayerLimits(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetPlayerLimitsResponseBody]
	CreateBalanceAdjustment(ctx provider.StepCtx, req *types.Request[models.CreateBalanceAdjustmentRequestBody]) *types.Response[models.CreateBalanceAdjustmentResponseBody]
	CreateBlockAmount(sCtx provider.StepCtx, req *types.Request[models.CreateBlockAmountRequestBody]) *types.Response[models.CreateBlockAmountResponseBody]
	GetBlockAmountList(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.BlockAmountListResponseBody]
	GetWalletList(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetWalletListResponseBody]
	DeleteBlockAmount(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[struct{}]

	UpdateBrandStatus(sCtx provider.StepCtx, req *types.Request[models.UpdateBrandStatusRequestBody]) *types.Response[struct{}]
	UpdateCapBrand(sCtx provider.StepCtx, req *types.Request[models.UpdateCapBrandRequestBody]) *types.Response[models.UpdateCapBrandResponseBody]
	UpdateCapBrandError(sCtx provider.StepCtx, req *types.Request[models.UpdateCapBrandRequestBody]) *types.Response[models.ErrorResponse]

	GetCapCategory(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[models.GetCapCategoryResponseBody]
	CreateCapCategory(sCtx provider.StepCtx, req *types.Request[models.CreateCapCategoryRequestBody]) *types.Response[models.CreateCapCategoryResponseBody]
	DeleteCapCategory(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[struct{}]
	CreateLabel(sCtx provider.StepCtx, req *types.Request[models.CreateLabelRequestBody]) *types.Response[models.CreateLabelResponseBody]
	GetLabel(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[models.GetLabelResponseBody]
	UpdateCapCategory(sCtx provider.StepCtx, req *types.Request[models.UpdateCapCategoryRequestBody]) *types.Response[models.UpdateCapCategoryResponseBody]
	UpdateCapCategoryError(sCtx provider.StepCtx, req *types.Request[models.UpdateCapCategoryRequestBody]) *types.Response[models.ErrorResponse]
}

type capClient struct {
	client       *types.Client
	tokenStorage *TokenStorage
}

func NewClient(sCtx provider.StepCtx, cfg *config.Config, baseClient *types.Client) CapAPI {
	client := &capClient{client: baseClient}
	client.tokenStorage = NewTokenStorage(sCtx, cfg, client)
	return client
}

func (c *capClient) GetToken(sCtx provider.StepCtx) string {
	return c.tokenStorage.GetToken(sCtx)
}

func (c *capClient) CheckAdmin(sCtx provider.StepCtx, req *types.Request[models.AdminCheckRequestBody]) *types.Response[models.AdminCheckResponseBody] {
	req.Method = "POST"
	req.Path = "/_cap/api/token/check"
	resp, err := httpClient.DoRequest[models.AdminCheckRequestBody, models.AdminCheckResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("CheckAdmin failed: %v", err)
		sCtx.Errorf("CheckAdmin failed: %v", err)
	}

	return resp
}

func (c *capClient) CreateCapBrand(sCtx provider.StepCtx, req *types.Request[models.CreateCapBrandRequestBody]) *types.Response[models.CreateCapBrandResponseBody] {
	req.Method = "POST"
	req.Path = "/_cap/api/v1/brands"
	resp, err := httpClient.DoRequest[models.CreateCapBrandRequestBody, models.CreateCapBrandResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("CreateCapBrand failed: %v", err)
	}

	return resp
}

func (c *capClient) GetCapBrand(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[models.GetCapBrandResponseBody] {
	req.Method = "GET"
	req.Path = "/_cap/api/v1/brands/{id}"
	resp, err := httpClient.DoRequest[struct{}, models.GetCapBrandResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("GetCapBrand failed: %v", err)
	}

	return resp
}

func (c *capClient) DeleteCapBrand(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[struct{}] {
	req.Method = "DELETE"
	req.Path = "/_cap/api/v1/brands/{id}"
	resp, err := httpClient.DoRequest[struct{}, struct{}](sCtx, c.client, req)
	if err != nil {
		log.Printf("DeleteCapBrand failed: %v", err)
	}

	return resp
}

func (c *capClient) UpdateBlockers(sCtx provider.StepCtx, req *types.Request[models.BlockersRequestBody]) *types.Response[struct{}] {
	req.Method = "PATCH"
	req.Path = "/_cap/api/v1/players/{player_uuid}/blockers"
	resp, err := httpClient.DoRequest[models.BlockersRequestBody, struct{}](sCtx, c.client, req)
	if err != nil {
		log.Printf("UpdateBlockers failed: %v", err)
	}

	return resp
}

func (c *capClient) GetBlockers(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetBlockersResponseBody] {
	req.Method = "GET"
	req.Path = "/_cap/api/v1/players/{player_uuid}/blockers"
	resp, err := httpClient.DoRequest[any, models.GetBlockersResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("GetBlockers failed: %v", err)
	}

	return resp
}

func (c *capClient) GetPlayerLimits(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetPlayerLimitsResponseBody] {
	req.Method = "GET"
	req.Path = "/_cap/api/v1/player/{playerID}/limits"
	req.QueryParams = map[string]string{
		"sort":    "status",
		"page":    "1",
		"perPage": "10",
	}

	resp, err := httpClient.DoRequest[any, models.GetPlayerLimitsResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("GetPlayerLimits failed: %v", err)
	}

	return resp
}

func (c *capClient) CreateLabel(sCtx provider.StepCtx, req *types.Request[models.CreateLabelRequestBody]) *types.Response[models.CreateLabelResponseBody] {
	req.Method = "POST"
	req.Path = "/_cap/bonus/api/v1/label/create"

	resp, err := httpClient.DoRequest[models.CreateLabelRequestBody, models.CreateLabelResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("CreateLabel failed: %v", err)
	}

	return resp
}

func (c *capClient) GetLabel(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[models.GetLabelResponseBody] {
	req.Method = "GET"
	req.Path = "/_cap/bonus/api/v1/label/show/{labelUUID}"

	resp, err := httpClient.DoRequest[struct{}, models.GetLabelResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("GetLabel failed: %v", err)
	}

	return resp
}

func (c *capClient) CreateCapCategory(sCtx provider.StepCtx, req *types.Request[models.CreateCapCategoryRequestBody]) *types.Response[models.CreateCapCategoryResponseBody] {
	req.Method = "POST"
	req.Path = "/_cap/api/v1/categories"
	resp, err := httpClient.DoRequest[models.CreateCapCategoryRequestBody, models.CreateCapCategoryResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("CreateCapCategory failed: %v", err)
		sCtx.Errorf("CreateCapCategory failed: %v", err)
		return &types.Response[models.CreateCapCategoryResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("CreateCapCategory failed: %v", err),
			},
		}
	}
	if resp.StatusCode != http.StatusOK {
		sCtx.Errorf("CreateCapCategory returned status %d: %v", resp.StatusCode, resp.Error)
	}
	return resp
}

func (c *capClient) GetCapCategory(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[models.GetCapCategoryResponseBody] {
	req.Method = "GET"
	req.Path = "/_cap/api/v1/categories/{id}"
	resp, err := httpClient.DoRequest[struct{}, models.GetCapCategoryResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("GetCapCategory failed: %v", err)
		return &types.Response[models.GetCapCategoryResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("GetCapCategory failed: %v", err),
			},
		}
	}

	return resp
}

func (c *capClient) DeleteCapCategory(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[struct{}] {
	req.Method = "DELETE"
	req.Path = "/_cap/api/v1/categories/{id}"
	resp, err := httpClient.DoRequest[struct{}, struct{}](sCtx, c.client, req)
	if err != nil {
		log.Printf("DeleteCapCategory failed: %v", err)
		return &types.Response[struct{}]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("DeleteCapCategory failed: %v", err),
			},
		}
	}
	return resp
}

func (c *capClient) UpdateBrandStatus(sCtx provider.StepCtx, req *types.Request[models.UpdateBrandStatusRequestBody]) *types.Response[struct{}] {
	req.Method = "PATCH"
	req.Path = "/_cap/api/v1/brands/{id}/status"
	resp, err := httpClient.DoRequest[models.UpdateBrandStatusRequestBody, struct{}](sCtx, c.client, req)
	if err != nil {
		log.Printf("UpdateBrandStatus failed: %v", err)
		return &types.Response[struct{}]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("UpdateBrandStatus failed: %v", err),
			},
		}
	}
	return resp
}

func (c *capClient) UpdateCapBrand(sCtx provider.StepCtx, req *types.Request[models.UpdateCapBrandRequestBody]) *types.Response[models.UpdateCapBrandResponseBody] {
	req.Method = "PATCH"
	req.Path = "/_cap/api/v1/brands/{id}"
	resp, err := httpClient.DoRequest[models.UpdateCapBrandRequestBody, models.UpdateCapBrandResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("UpdateCapBrand failed: %v", err)
		return &types.Response[models.UpdateCapBrandResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("UpdateCapBrand failed: %v", err),
			},
		}
	}
	return resp
}

func (c *capClient) UpdateCapBrandError(sCtx provider.StepCtx, req *types.Request[models.UpdateCapBrandRequestBody]) *types.Response[models.ErrorResponse] {
	req.Method = "PATCH"
	req.Path = "/_cap/api/v1/brands/{id}"
	resp, err := httpClient.DoRequest[models.UpdateCapBrandRequestBody, models.ErrorResponse](sCtx, c.client, req)
	if err != nil {
		log.Printf("UpdateCapBrand failed: %v", err)
		return &types.Response[models.ErrorResponse]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("UpdateCapBrand failed: %v", err),
			},
		}
	}
	return resp
}

func (c *capClient) UpdateCapCategory(sCtx provider.StepCtx, req *types.Request[models.UpdateCapCategoryRequestBody]) *types.Response[models.UpdateCapCategoryResponseBody] {
	req.Method = "PATCH"
	req.Path = "/_cap/api/v1/categories/{id}"
	resp, err := httpClient.DoRequest[models.UpdateCapCategoryRequestBody, models.UpdateCapCategoryResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("UpdateCategoty failed: %v", err)
		return &types.Response[models.UpdateCapCategoryResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("UpdateCapCategoty failed: %v", err),
			},
		}
	}
	return resp
}

func (c *capClient) UpdateCapCategoryError(sCtx provider.StepCtx, req *types.Request[models.UpdateCapCategoryRequestBody]) *types.Response[models.ErrorResponse] {
	req.Method = "PATCH"
	req.Path = "/_cap/api/v1/categories/{id}"
	resp, err := httpClient.DoRequest[models.UpdateCapCategoryRequestBody, models.ErrorResponse](sCtx, c.client, req)
	if err != nil {
		log.Printf("UpdateCapCategory failed: %v", err)
		return &types.Response[models.ErrorResponse]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("UpdateCapCategory failed: %v", err),
			},
		}
	}
	return resp
}

func (c *capClient) CreateBalanceAdjustment(sCtx provider.StepCtx, req *types.Request[models.CreateBalanceAdjustmentRequestBody]) *types.Response[models.CreateBalanceAdjustmentResponseBody] {
	req.Method = "POST"
	req.Path = "/_cap/api/v1/wallet/{player_uuid}/create-balance-adjustment"
	resp, err := httpClient.DoRequest[models.CreateBalanceAdjustmentRequestBody, models.CreateBalanceAdjustmentResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("CreateBalanceAdjustment failed: %v", err)
		return &types.Response[models.CreateBalanceAdjustmentResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("CreateBalanceAdjustment failed: %v", err),
			},
		}
	}

	return resp
}

func (c *capClient) CreateBlockAmount(sCtx provider.StepCtx, req *types.Request[models.CreateBlockAmountRequestBody]) *types.Response[models.CreateBlockAmountResponseBody] {
	req.Method = "POST"
	req.Path = "/_cap/api/v1/wallet/{player_uuid}/create-block-amount"
	resp, err := httpClient.DoRequest[models.CreateBlockAmountRequestBody, models.CreateBlockAmountResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("CreateBlockAmount failed: %v", err)
		return &types.Response[models.CreateBlockAmountResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("CreateBlockAmount failed: %v", err),
			},
		}
	}

	return resp
}

func (c *capClient) GetBlockAmountList(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.BlockAmountListResponseBody] {
	req.Method = "GET"
	req.Path = "/_cap/api/v1/wallet/{player_uuid}/block-amount-list"
	resp, err := httpClient.DoRequest[any, models.BlockAmountListResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("GetBlockAmountList failed: %v", err)
		return &types.Response[models.BlockAmountListResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("GetBlockAmountList failed: %v", err),
			},
		}
	}

	return resp
}

func (c *capClient) GetWalletList(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetWalletListResponseBody] {
	req.Method = "GET"
	req.Path = "/_cap/api/v2/wallet/{player_uuid}/list"
	resp, err := httpClient.DoRequest[any, models.GetWalletListResponseBody](sCtx, c.client, req)
	if err != nil {
		log.Printf("GetWalletList failed: %v", err)
		return &types.Response[models.GetWalletListResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("GetWalletList failed: %v", err),
			},
		}
	}

	return resp
}

func (c *capClient) DeleteBlockAmount(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[struct{}] {
	req.Method = "DELETE"
	req.Path = "/_cap/api/v1/wallet/delete-amount-block/{block_uuid}"
	resp, err := httpClient.DoRequest[any, struct{}](sCtx, c.client, req)
	if err != nil {
		log.Printf("DeleteBlockAmount failed: %v", err)
		return &types.Response[struct{}]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("DeleteBlockAmount failed: %v", err),
			},
		}
	}

	return resp
}
