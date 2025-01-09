package requests

import (
	httpClient "CB_auto/test/transport/http"
	"fmt"
	"net/http"
)

const (
	pathCreateBrand = "/_cap/api/v1/brands"
)

type CreateCapBrandRequest struct {
	Headers CreateCapBrandRequestHeaders
	Body    CreateCapBrandRequestBody
}

type CreateCapBrandResponse struct {
	Body       CreateCapBrandResponseBody
	StatusCode int
}

type CreateCapBrandRequestHeaders struct {
	Authorization  string `json:"Authorization"`
	PlatformNodeID string `json:"platform-nodeid"`
}

type CreateCapBrandRequestBody struct {
	Sort        int               `json:"sort"`
	Alias       string            `json:"alias"`
	Names       map[string]string `json:"names"`
	Description string            `json:"description"`
}

type CreateCapBrandResponseBody struct {
	ID string `json:"id"`
}

func CreateCapBrand(client *httpClient.Client, request *httpClient.Request[CreateCapBrandRequestBody]) (*httpClient.Response[CreateCapBrandResponseBody], error) {
	request.Path = pathCreateBrand
	request.Method = http.MethodPost

	response, err := httpClient.DoRequest[CreateCapBrandRequestBody, CreateCapBrandResponseBody](client, request)
	if err != nil {
		return nil, fmt.Errorf("CheckAdmin failed: %v", err)
	}

	return response, nil
}
