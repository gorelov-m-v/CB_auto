package cap

import (
	"fmt"
	"net/http"

	httpClient "CB_auto/internal/client"
	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/types"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

func (c *capClient) UpdateVerificationStatus(sCtx provider.StepCtx, req *types.Request[models.UpdateVerificationStatusRequestBody]) *types.Response[struct{}] {
	req.Method = http.MethodPatch
	req.Path = fmt.Sprintf("/_cap/api/v1/players/verification/%s", req.PathParams["verification_id"])
	return httpClient.DoRequest[models.UpdateVerificationStatusRequestBody, struct{}](sCtx, c.client, req)
}
