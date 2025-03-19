package cap

import (
	"net/http"

	httpClient "CB_auto/internal/client"
	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/types"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

func (c *capClient) GetGames(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[models.GetCapGamesResponseBody] {
	req.Method = http.MethodGet
	req.Path = "/_cap/api/v1/games/{id}"
	return httpClient.DoRequest[struct{}, models.GetCapGamesResponseBody](sCtx, c.client, req)
}

func (c *capClient) UpdateGamesStatus(sCtx provider.StepCtx, req *types.Request[models.UpdateGamesStatusRequestBody]) *types.Response[models.UpdateCapGamesResponseBody] {
	req.Method = http.MethodPatch
	req.Path = "/_cap/api/v1/games/{id}"
	return httpClient.DoRequest[models.UpdateGamesStatusRequestBody, models.UpdateCapGamesResponseBody](sCtx, c.client, req)
}

func (c *capClient) UpdateGames(sCtx provider.StepCtx, req *types.Request[models.UpdateCapGamesRequestBody]) *types.Response[models.UpdateCapGamesResponseBody] {
	req.Method = http.MethodPut
	req.Path = "/_cap/api/v1/games/{id}"
	return httpClient.DoRequest[models.UpdateCapGamesRequestBody, models.UpdateCapGamesResponseBody](sCtx, c.client, req)
}
