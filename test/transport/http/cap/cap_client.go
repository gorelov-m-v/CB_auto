package cap

import (
	httpClient "CB_auto/test/transport/http"
	"CB_auto/test/transport/http/cap/models"
	"fmt"
	"net/http"
)

type CapAPI interface {
	CheckAdmin(body models.AdminCheckRequestBody) (*models.AdminCheckResponseBody, *httpClient.RequestDetails, *httpClient.ResponseDetails, error)
	CreateCapBrand(body models.CreateCapBrandRequestBody, headers models.CreateCapBrandRequestHeaders) (*models.CreateCapBrandResponseBody, *httpClient.RequestDetails, *httpClient.ResponseDetails, error)
	GetCapBrand(uuid string, headers models.GetCapBrandRequestHeaders) (*models.GetCapBrandResponseBody, *httpClient.RequestDetails, *httpClient.ResponseDetails, error)
}

type capClient struct {
	client *httpClient.Client
}

func NewCapClient(client *httpClient.Client) CapAPI {
	return &capClient{client: client}
}

func (c *capClient) CheckAdmin(body models.AdminCheckRequestBody) (*models.AdminCheckResponseBody, *httpClient.RequestDetails, *httpClient.ResponseDetails, error) {
	req := &httpClient.Request[models.AdminCheckRequestBody]{
		Method: http.MethodPost,
		Path:   "/_cap/api/token/check",
		Body:   &body,
	}
	resp, reqDetails, respDetails, err := httpClient.DoRequest[models.AdminCheckRequestBody, models.AdminCheckResponseBody](c.client, req)
	if err != nil {
		return nil, reqDetails, respDetails, fmt.Errorf("CheckAdmin failed: %v", err)
	}
	return &resp.Body, reqDetails, respDetails, nil
}

func (c *capClient) CreateCapBrand(body models.CreateCapBrandRequestBody, headers models.CreateCapBrandRequestHeaders) (*models.CreateCapBrandResponseBody, *httpClient.RequestDetails, *httpClient.ResponseDetails, error) {
	req := &httpClient.Request[models.CreateCapBrandRequestBody]{
		Method: http.MethodPost,
		Path:   "/_cap/api/v1/brands",
		Headers: map[string]string{
			"Authorization":   headers.Authorization,
			"Platform-Nodeid": headers.PlatformNodeID,
		},
		Body: &body,
	}
	resp, reqDetails, respDetails, err := httpClient.DoRequest[models.CreateCapBrandRequestBody, models.CreateCapBrandResponseBody](c.client, req)
	if err != nil {
		return nil, reqDetails, respDetails, fmt.Errorf("CreateCapBrand failed: %v", err)
	}
	return &resp.Body, reqDetails, respDetails, nil
}

func (c *capClient) GetCapBrand(uuid string, headers models.GetCapBrandRequestHeaders) (*models.GetCapBrandResponseBody, *httpClient.RequestDetails, *httpClient.ResponseDetails, error) {
	path := fmt.Sprintf("/_cap/api/v1/brands/%s", uuid)
	req := &httpClient.Request[struct{}]{
		Method: http.MethodGet,
		Path:   path,
		Headers: map[string]string{
			"Authorization":   headers.Authorization,
			"Platform-Nodeid": headers.PlatformNodeID,
		},
		Body: nil,
	}
	resp, reqDetails, respDetails, err := httpClient.DoRequest[struct{}, models.GetCapBrandResponseBody](c.client, req)
	if err != nil {
		return nil, reqDetails, respDetails, fmt.Errorf("GetCapBrand failed: %v", err)
	}
	return &resp.Body, reqDetails, respDetails, nil
}
