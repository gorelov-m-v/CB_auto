package cap

import (
	"net/http"

	httpClient "CB_auto/internal/client"
	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/types"
	"CB_auto/internal/config"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

type CapAPI interface {
	// Common
	GetToken(sCtx provider.StepCtx) string
	CheckAdmin(sCtx provider.StepCtx, req *types.Request[models.AdminCheckRequestBody]) *types.Response[models.AdminCheckResponseBody]

	// Player
	UpdateVerificationStatus(sCtx provider.StepCtx, req *types.Request[models.UpdateVerificationStatusRequestBody]) *types.Response[struct{}]

	// Wallet
	UpdateBlockers(sCtx provider.StepCtx, req *types.Request[models.BlockersRequestBody]) *types.Response[struct{}]
	GetPlayerLimits(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetPlayerLimitsResponseBody]
	GetBlockers(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetBlockersResponseBody]
	CreateBalanceAdjustment(sCtx provider.StepCtx, req *types.Request[models.CreateBalanceAdjustmentRequestBody]) *types.Response[struct{}]
	CreateBlockAmount(sCtx provider.StepCtx, req *types.Request[models.CreateBlockAmountRequestBody]) *types.Response[models.CreateBlockAmountResponseBody]
	GetBlockAmountList(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.BlockAmountListResponseBody]
	GetWalletList(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetWalletListResponseBody]
	DeleteBlockAmount(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[struct{}]

	// Gambling
	GetCapBrand(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[models.GetCapBrandResponseBody]
	DeleteCapBrand(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[struct{}]
	CreateCapBrand(sCtx provider.StepCtx, req *types.Request[models.CreateCapBrandRequestBody]) *types.Response[models.CreateCapBrandResponseBody]
	UpdateBrandStatus(sCtx provider.StepCtx, req *types.Request[models.UpdateBrandStatusRequestBody]) *types.Response[struct{}]
	UpdateCapBrand(sCtx provider.StepCtx, req *types.Request[models.UpdateCapBrandRequestBody]) *types.Response[models.UpdateCapBrandResponseBody]
	UpdateCapBrandError(sCtx provider.StepCtx, req *types.Request[models.UpdateCapBrandRequestBody]) *types.Response[models.ErrorResponse]
	GetCapCategory(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[models.GetCapCategoryResponseBody]
	CreateCapCategory(sCtx provider.StepCtx, req *types.Request[models.CreateCapCategoryRequestBody]) *types.Response[models.CreateCapCategoryResponseBody]
	DeleteCapCategory(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[struct{}]
	UpdateCapCategory(sCtx provider.StepCtx, req *types.Request[models.UpdateCapCategoryRequestBody]) *types.Response[models.UpdateCapCategoryResponseBody]
	UpdateCapCategoryError(sCtx provider.StepCtx, req *types.Request[models.UpdateCapCategoryRequestBody]) *types.Response[models.ErrorResponse]

	// Bonus
	CreateLabel(sCtx provider.StepCtx, req *types.Request[models.CreateLabelRequestBody]) *types.Response[models.CreateLabelResponseBody]
	GetLabel(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[models.GetLabelResponseBody]
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
	req.Method = http.MethodPost
	req.Path = "/_cap/api/token/check"
	return httpClient.DoRequest[models.AdminCheckRequestBody, models.AdminCheckResponseBody](sCtx, c.client, req)
}
