package public

import (
	"CB_auto/internal/client/public/models"
	"CB_auto/internal/client/types"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

// PublicAPI объединяет все методы публичного API.
type PublicAPI interface {
	// Player методы
	FastRegistration(sCtx provider.StepCtx, req *types.Request[models.FastRegistrationRequestBody]) *types.Response[models.FastRegistrationResponseBody]
	FullRegistration(sCtx provider.StepCtx, req *types.Request[models.FullRegistrationRequestBody]) *types.Response[struct{}]
	TokenCheck(sCtx provider.StepCtx, req *types.Request[models.TokenCheckRequestBody]) *types.Response[models.TokenCheckResponseBody]
	UpdatePlayer(sCtx provider.StepCtx, req *types.Request[models.UpdatePlayerRequestBody]) *types.Response[models.UpdatePlayerResponseBody]
	VerifyIdentity(sCtx provider.StepCtx, req *types.Request[models.VerifyIdentityRequestBody]) *types.Response[struct{}]
	GetVerificationStatus(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[[]models.VerificationStatusResponseItem]
	RequestContactVerification(sCtx provider.StepCtx, req *types.Request[models.RequestVerificationRequestBody]) *types.Response[struct{}]
	ConfirmContact(sCtx provider.StepCtx, req *types.Request[models.ConfirmContactRequestBody]) *types.Response[struct{}]
	VerifyContact(sCtx provider.StepCtx, req *types.Request[models.VerifyContactRequestBody]) *types.Response[models.VerifyContactResponseBody]

	// Wallet методы
	GetWallets(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetWalletsResponseBody]
	CreateWallet(sCtx provider.StepCtx, req *types.Request[models.CreateWalletRequestBody]) *types.Response[models.CreateWalletResponseBody]
	SwitchWallet(sCtx provider.StepCtx, req *types.Request[models.SwitchWalletRequestBody]) *types.Response[struct{}]
	RemoveWallet(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[struct{}]

	// Limits методы
	SetSingleBetLimit(sCtx provider.StepCtx, req *types.Request[models.SetSingleBetLimitRequestBody]) *types.Response[struct{}]
	SetCasinoLossLimit(sCtx provider.StepCtx, req *types.Request[models.SetCasinoLossLimitRequestBody]) *types.Response[struct{}]
	GetTurnoverLimits(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetTurnoverLimitsResponseBody]
	GetSingleBetLimits(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetSingleBetLimitsResponseBody]
	SetRestriction(sCtx provider.StepCtx, req *types.Request[models.SetRestrictionRequestBody]) *types.Response[models.SetRestrictionResponseBody]
	GetRestriction(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.SetRestrictionResponseBody]
	GetCasinoLossLimits(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetCasinoLossLimitsResponseBody]
	SetTurnoverLimit(sCtx provider.StepCtx, req *types.Request[models.SetTurnoverLimitRequestBody]) *types.Response[struct{}]

	// Payment методы
	CreateDeposit(sCtx provider.StepCtx, req *types.Request[models.DepositRequestBody]) *types.Response[struct{}]
}

type publicClient struct {
	client *types.Client
}

func NewClient(baseClient *types.Client) PublicAPI {
	return &publicClient{client: baseClient}
}
