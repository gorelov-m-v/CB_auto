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
	GetToken() string
	CheckAdmin(req *types.Request[models.AdminCheckRequestBody]) *types.Response[models.AdminCheckResponseBody]
	CreateCapBrand(req *types.Request[models.CreateCapBrandRequestBody]) *types.Response[models.CreateCapBrandResponseBody]
	GetCapBrand(req *types.Request[struct{}]) *types.Response[models.GetCapBrandResponseBody]
	DeleteCapBrand(req *types.Request[struct{}]) *types.Response[struct{}]
	UpdateBlockers(req *types.Request[models.BlockersRequestBody]) *types.Response[struct{}]
	GetBlockers(req *types.Request[any]) *types.Response[models.GetBlockersResponseBody]
	UpdateBrandStatus(req *types.Request[models.UpdateBrandStatusRequestBody]) *types.Response[struct{}]
	UpdateCapBrand(req *types.Request[models.UpdateCapBrandRequestBody]) *types.Response[models.UpdateCapBrandResponseBody]
	UpdateCapBrandError(req *types.Request[models.UpdateCapBrandRequestBody]) *types.Response[models.ErrorResponse]
}

type capClient struct {
	client       *types.Client
	tokenStorage *TokenStorage
}

func NewClient(t provider.T, cfg *config.Config, baseClient *types.Client) CapAPI {
	client := &capClient{client: baseClient}
	client.tokenStorage = NewTokenStorage(t, cfg, client)
	return client
}

func (c *capClient) GetToken() string {
	return c.tokenStorage.GetToken()
}

func (c *capClient) CheckAdmin(req *types.Request[models.AdminCheckRequestBody]) *types.Response[models.AdminCheckResponseBody] {
	req.Method = "POST"
	req.Path = "/_cap/api/token/check"
	resp, err := httpClient.DoRequest[models.AdminCheckRequestBody, models.AdminCheckResponseBody](c.client, req)
	if err != nil {
		log.Printf("CheckAdmin failed: %v", err)
		return &types.Response[models.AdminCheckResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("CheckAdmin failed: %v", err),
			},
		}
	}
	return resp
}

func (c *capClient) CreateCapBrand(req *types.Request[models.CreateCapBrandRequestBody]) *types.Response[models.CreateCapBrandResponseBody] {
	req.Method = "POST"
	req.Path = "/_cap/api/v1/brands"
	resp, err := httpClient.DoRequest[models.CreateCapBrandRequestBody, models.CreateCapBrandResponseBody](c.client, req)
	if err != nil {
		log.Printf("CreateCapBrand failed: %v", err)
		return &types.Response[models.CreateCapBrandResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("CreateCapBrand failed: %v", err),
			},
		}
	}
	return resp
}

func (c *capClient) GetCapBrand(req *types.Request[struct{}]) *types.Response[models.GetCapBrandResponseBody] {
	req.Method = "GET"
	req.Path = "/_cap/api/v1/brands/{id}"
	resp, err := httpClient.DoRequest[struct{}, models.GetCapBrandResponseBody](c.client, req)
	if err != nil {
		log.Printf("GetCapBrand failed: %v", err)
		return &types.Response[models.GetCapBrandResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("GetCapBrand failed: %v", err),
			},
		}
	}
	return resp
}

func (c *capClient) DeleteCapBrand(req *types.Request[struct{}]) *types.Response[struct{}] {
	req.Method = "DELETE"
	req.Path = "/_cap/api/v1/brands/{id}"
	resp, err := httpClient.DoRequest[struct{}, struct{}](c.client, req)
	if err != nil {
		log.Printf("DeleteCapBrand failed: %v", err)
		return &types.Response[struct{}]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("DeleteCapBrand failed: %v", err),
			},
		}
	}
	return resp
}

func (c *capClient) UpdateBlockers(req *types.Request[models.BlockersRequestBody]) *types.Response[struct{}] {
	req.Method = "PATCH"
	req.Path = "/_cap/api/v1/players/{player_uuid}/blockers"
	resp, err := httpClient.DoRequest[models.BlockersRequestBody, struct{}](c.client, req)
	if err != nil {
		log.Printf("UpdateBlockers failed: %v", err)
		return &types.Response[struct{}]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("UpdateBlockers failed: %v", err),
			},
		}
	}
	return resp
}

func (c *capClient) GetBlockers(req *types.Request[any]) *types.Response[models.GetBlockersResponseBody] {
	req.Method = "GET"
	req.Path = "/_cap/api/v1/players/{player_uuid}/blockers"
	resp, err := httpClient.DoRequest[any, models.GetBlockersResponseBody](c.client, req)
	if err != nil {
		log.Printf("GetBlockers failed: %v", err)
		return &types.Response[models.GetBlockersResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &types.ErrorResponse{
				Body: fmt.Sprintf("GetBlockers failed: %v", err),
			},
		}
	}
	return resp
}

func (c *capClient) UpdateBrandStatus(req *types.Request[models.UpdateBrandStatusRequestBody]) *types.Response[struct{}] {
	req.Method = "PATCH"
	req.Path = "/_cap/api/v1/brands/{id}/status"
	resp, err := httpClient.DoRequest[models.UpdateBrandStatusRequestBody, struct{}](c.client, req)
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

func (c *capClient) UpdateCapBrand(req *types.Request[models.UpdateCapBrandRequestBody]) *types.Response[models.UpdateCapBrandResponseBody] {
	req.Method = "PATCH"
	req.Path = "/_cap/api/v1/brands/{id}"
	resp, err := httpClient.DoRequest[models.UpdateCapBrandRequestBody, models.UpdateCapBrandResponseBody](c.client, req)
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

func (c *capClient) UpdateCapBrandError(req *types.Request[models.UpdateCapBrandRequestBody]) *types.Response[models.ErrorResponse] {
	req.Method = "PATCH"
	req.Path = "/_cap/api/v1/brands/{id}"
	resp, err := httpClient.DoRequest[models.UpdateCapBrandRequestBody, models.ErrorResponse](c.client, req)
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
