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
	CheckAdmin(sCtx provider.StepCtx, req *types.Request[models.AdminCheckRequestBody]) *types.Response[models.AdminCheckResponseBody]
	CreateCapBrand(sCtx provider.StepCtx, req *types.Request[models.CreateCapBrandRequestBody]) *types.Response[models.CreateCapBrandResponseBody]
	GetCapBrand(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[models.GetCapBrandResponseBody]
	DeleteCapBrand(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[struct{}]
	UpdateBlockers(sCtx provider.StepCtx, req *types.Request[models.BlockersRequestBody]) *types.Response[struct{}]
	GetBlockers(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetBlockersResponseBody]
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

func (c *capClient) GetToken() string {
	return c.tokenStorage.GetToken()
}

func (c *capClient) CheckAdmin(sCtx provider.StepCtx, req *types.Request[models.AdminCheckRequestBody]) *types.Response[models.AdminCheckResponseBody] {
	req.Method = "POST"
	req.Path = "/_cap/api/token/check"
	resp, err := httpClient.DoRequest[models.AdminCheckRequestBody, models.AdminCheckResponseBody](sCtx, c.client, req)
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

func (c *capClient) CreateCapBrand(sCtx provider.StepCtx, req *types.Request[models.CreateCapBrandRequestBody]) *types.Response[models.CreateCapBrandResponseBody] {
	req.Method = "POST"
	req.Path = "/_cap/api/v1/brands"
	resp, err := httpClient.DoRequest[models.CreateCapBrandRequestBody, models.CreateCapBrandResponseBody](sCtx, c.client, req)
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

func (c *capClient) GetCapBrand(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[models.GetCapBrandResponseBody] {
	req.Method = "GET"
	req.Path = "/_cap/api/v1/brands/{id}"
	resp, err := httpClient.DoRequest[struct{}, models.GetCapBrandResponseBody](sCtx, c.client, req)
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

func (c *capClient) DeleteCapBrand(sCtx provider.StepCtx, req *types.Request[struct{}]) *types.Response[struct{}] {
	req.Method = "DELETE"
	req.Path = "/_cap/api/v1/brands/{id}"
	resp, err := httpClient.DoRequest[struct{}, struct{}](sCtx, c.client, req)
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

func (c *capClient) UpdateBlockers(sCtx provider.StepCtx, req *types.Request[models.BlockersRequestBody]) *types.Response[struct{}] {
	req.Method = "PATCH"
	req.Path = "/_cap/api/v1/players/{player_uuid}/blockers"
	resp, err := httpClient.DoRequest[models.BlockersRequestBody, struct{}](sCtx, c.client, req)
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

func (c *capClient) GetBlockers(sCtx provider.StepCtx, req *types.Request[any]) *types.Response[models.GetBlockersResponseBody] {
	req.Method = "GET"
	req.Path = "/_cap/api/v1/players/{player_uuid}/blockers"
	resp, err := httpClient.DoRequest[any, models.GetBlockersResponseBody](sCtx, c.client, req)
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
