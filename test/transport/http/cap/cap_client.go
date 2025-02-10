package cap

import (
	httpClient "CB_auto/test/transport/http"
	"CB_auto/test/transport/http/cap/models"
	"fmt"
	"net/http"
)

type CapAPI interface {
	CheckAdmin(req *httpClient.Request[models.AdminCheckRequestBody]) (*httpClient.Response[models.AdminCheckResponseBody], error)
	CreateCapBrand(req *httpClient.Request[models.CreateCapBrandRequestBody]) (*httpClient.Response[models.CreateCapBrandResponseBody], error)
	GetCapBrand(req *httpClient.Request[struct{}]) (*httpClient.Response[models.GetCapBrandResponseBody], error)
}

type capClient struct {
	client *httpClient.Client
}

func NewCapClient(client *httpClient.Client) CapAPI {
	return &capClient{client: client}
}

func (c *capClient) CheckAdmin(req *httpClient.Request[models.AdminCheckRequestBody]) (*httpClient.Response[models.AdminCheckResponseBody], error) {
	req.Method = http.MethodPost
	req.Path = "/_cap/api/token/check"
	resp, err := httpClient.DoRequest[models.AdminCheckRequestBody, models.AdminCheckResponseBody](c.client, req)
	if err != nil {
		return nil, fmt.Errorf("CheckAdmin failed: %v", err)
	}
	return resp, nil
}

func (c *capClient) CreateCapBrand(req *httpClient.Request[models.CreateCapBrandRequestBody]) (*httpClient.Response[models.CreateCapBrandResponseBody], error) {
	req.Method = http.MethodPost
	req.Path = "/_cap/api/v1/brands"
	resp, err := httpClient.DoRequest[models.CreateCapBrandRequestBody, models.CreateCapBrandResponseBody](c.client, req)
	if err != nil {
		return nil, fmt.Errorf("CreateCapBrand failed: %v", err)
	}
	return resp, nil
}

func (c *capClient) GetCapBrand(req *httpClient.Request[struct{}]) (*httpClient.Response[models.GetCapBrandResponseBody], error) {
	req.Method = http.MethodGet
	req.Path = "/_cap/api/v1/brands/{id}"
	resp, err := httpClient.DoRequest[struct{}, models.GetCapBrandResponseBody](c.client, req)
	if err != nil {
		return nil, fmt.Errorf("GetCapBrand failed: %v", err)
	}
	return resp, nil
}
