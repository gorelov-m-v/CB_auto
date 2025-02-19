package cap

import (
	httpClient "CB_auto/internal/client"
	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/config"
	"fmt"
	"log"
	"net/http"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

type CapAPI interface {
	GetToken() string
	CheckAdmin(req *httpClient.Request[models.AdminCheckRequestBody]) *httpClient.Response[models.AdminCheckResponseBody]
	CreateCapBrand(req *httpClient.Request[models.CreateCapBrandRequestBody]) *httpClient.Response[models.CreateCapBrandResponseBody]
	GetCapBrand(req *httpClient.Request[struct{}]) *httpClient.Response[models.GetCapBrandResponseBody]
	DeleteCapBrand(req *httpClient.Request[struct{}]) *httpClient.Response[struct{}]
	UpdateBlockers(req *httpClient.Request[models.BlockersRequestBody]) *httpClient.Response[struct{}]
	GetBlockers(req *httpClient.Request[any]) *httpClient.Response[models.GetBlockersResponseBody]
}

type capClient struct {
	client       *httpClient.Client
	tokenStorage *TokenStorage
}

func NewCapClient(t provider.T, cfg *config.Config, client *httpClient.Client) CapAPI {
	c := &capClient{client: client}
	c.tokenStorage = NewTokenStorage(t, cfg, c)
	return c
}

func (c *capClient) GetToken() string {
	return c.tokenStorage.GetToken()
}

func (c *capClient) CheckAdmin(req *httpClient.Request[models.AdminCheckRequestBody]) *httpClient.Response[models.AdminCheckResponseBody] {
	req.Method = "POST"
	req.Path = "/_cap/api/token/check"
	resp, err := httpClient.DoRequest[models.AdminCheckRequestBody, models.AdminCheckResponseBody](c.client, req)
	if err != nil {
		log.Printf("CheckAdmin failed: %v", err)
		return &httpClient.Response[models.AdminCheckResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &httpClient.ErrorResponse{
				Body: fmt.Sprintf("CheckAdmin failed: %v", err),
			},
		}
	}
	return resp
}

func (c *capClient) CreateCapBrand(req *httpClient.Request[models.CreateCapBrandRequestBody]) *httpClient.Response[models.CreateCapBrandResponseBody] {
	req.Method = "POST"
	req.Path = "/_cap/api/v1/brands"
	resp, err := httpClient.DoRequest[models.CreateCapBrandRequestBody, models.CreateCapBrandResponseBody](c.client, req)
	if err != nil {
		log.Printf("CreateCapBrand failed: %v", err)
		return &httpClient.Response[models.CreateCapBrandResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &httpClient.ErrorResponse{
				Body: fmt.Sprintf("CreateCapBrand failed: %v", err),
			},
		}
	}
	return resp
}

func (c *capClient) GetCapBrand(req *httpClient.Request[struct{}]) *httpClient.Response[models.GetCapBrandResponseBody] {
	req.Method = "GET"
	req.Path = "/_cap/api/v1/brands/{id}"
	resp, err := httpClient.DoRequest[struct{}, models.GetCapBrandResponseBody](c.client, req)
	if err != nil {
		log.Printf("GetCapBrand failed: %v", err)
		return &httpClient.Response[models.GetCapBrandResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &httpClient.ErrorResponse{
				Body: fmt.Sprintf("GetCapBrand failed: %v", err),
			},
		}
	}
	return resp
}

func (c *capClient) DeleteCapBrand(req *httpClient.Request[struct{}]) *httpClient.Response[struct{}] {
	req.Method = "DELETE"
	req.Path = "/_cap/api/v1/brands/{id}"
	resp, err := httpClient.DoRequest[struct{}, struct{}](c.client, req)
	if err != nil {
		log.Printf("DeleteCapBrand failed: %v", err)
		return &httpClient.Response[struct{}]{
			StatusCode: http.StatusInternalServerError,
			Error: &httpClient.ErrorResponse{
				Body: fmt.Sprintf("DeleteCapBrand failed: %v", err),
			},
		}
	}
	return resp
}

func (c *capClient) UpdateBlockers(req *httpClient.Request[models.BlockersRequestBody]) *httpClient.Response[struct{}] {
	req.Method = "PATCH"
	req.Path = "/_cap/api/v1/players/{player_uuid}/blockers"
	resp, err := httpClient.DoRequest[models.BlockersRequestBody, struct{}](c.client, req)
	if err != nil {
		log.Printf("UpdateBlockers failed: %v", err)
		return &httpClient.Response[struct{}]{
			StatusCode: http.StatusInternalServerError,
			Error: &httpClient.ErrorResponse{
				Body: fmt.Sprintf("UpdateBlockers failed: %v", err),
			},
		}
	}
	return resp
}

func (c *capClient) GetBlockers(req *httpClient.Request[any]) *httpClient.Response[models.GetBlockersResponseBody] {
	req.Method = "GET"
	req.Path = "/_cap/api/v1/players/{player_uuid}/blockers"
	resp, err := httpClient.DoRequest[any, models.GetBlockersResponseBody](c.client, req)
	if err != nil {
		log.Printf("GetBlockers failed: %v", err)
		return &httpClient.Response[models.GetBlockersResponseBody]{
			StatusCode: http.StatusInternalServerError,
			Error: &httpClient.ErrorResponse{
				Body: fmt.Sprintf("GetBlockers failed: %v", err),
			},
		}
	}
	return resp
}
