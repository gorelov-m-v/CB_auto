package cap

import (
	httpClient "CB_auto/test/transport/http"
	"CB_auto/test/transport/http/cap/models"
	"fmt"
)

type CapAPI interface {
	CheckAdmin(req *httpClient.Request[models.AdminCheckRequestBody]) *httpClient.Response[models.AdminCheckResponseBody]
	CreateCapBrand(req *httpClient.Request[models.CreateCapBrandRequestBody]) *httpClient.Response[models.CreateCapBrandResponseBody]
	GetCapBrand(req *httpClient.Request[struct{}]) *httpClient.Response[models.GetCapBrandResponseBody]
	DeleteCapBrand(req *httpClient.Request[struct{}]) *httpClient.Response[struct{}]
}

type capClient struct {
	client *httpClient.Client
}

func NewCapClient(client *httpClient.Client) CapAPI {
	return &capClient{client: client}
}

func (c *capClient) CheckAdmin(req *httpClient.Request[models.AdminCheckRequestBody]) *httpClient.Response[models.AdminCheckResponseBody] {
	req.Method = "POST"
	req.Path = "/_cap/api/token/check"
	resp, err := httpClient.DoRequest[models.AdminCheckRequestBody, models.AdminCheckResponseBody](c.client, req)
	if err != nil {
		panic(fmt.Sprintf("CheckAdmin не удался: %v", err))
	}
	return resp
}

func (c *capClient) CreateCapBrand(req *httpClient.Request[models.CreateCapBrandRequestBody]) *httpClient.Response[models.CreateCapBrandResponseBody] {
	req.Method = "POST"
	req.Path = "/_cap/api/v1/brands"
	resp, err := httpClient.DoRequest[models.CreateCapBrandRequestBody, models.CreateCapBrandResponseBody](c.client, req)
	if err != nil {
		panic(fmt.Sprintf("CreateCapBrand не удался: %v", err))
	}
	return resp
}

func (c *capClient) GetCapBrand(req *httpClient.Request[struct{}]) *httpClient.Response[models.GetCapBrandResponseBody] {
	req.Method = "GET"
	req.Path = "/_cap/api/v1/brands/{id}"
	resp, err := httpClient.DoRequest[struct{}, models.GetCapBrandResponseBody](c.client, req)
	if err != nil {
		panic(fmt.Sprintf("GetCapBrand не удался: %v", err))
	}
	return resp
}

func (c *capClient) DeleteCapBrand(req *httpClient.Request[struct{}]) *httpClient.Response[struct{}] {
	req.Method = "DELETE"
	req.Path = "/_cap/api/v1/brands/{id}"
	resp, err := httpClient.DoRequest[struct{}, struct{}](c.client, req)
	if err != nil {
		panic(fmt.Sprintf("DeleteCapBrand не удался: %v", err))
	}
	return resp
}
