package requests

import (
	httpClient "CB_auto/test/transport/http"
	"fmt"
	"net/http"
)

const (
	basePath = "/_cap/api/v1/brands"
)

type GetCapBrandRequest struct {
	PathParams GetCapBrandRequestPathParams
	Headers    GetCapBrandRequestHeaders
}

type GetCapBrandResponse struct {
	Body       CreateCapBrandResponseBody
	StatusCode int
}

type GetCapBrandRequestPathParams struct {
	UUID string
}

type GetCapBrandRequestHeaders struct {
	Authorization  string
	PlatformNodeID string
}

type GetCapBrandResponseBody struct {
	ID          string            `json:"id"`
	Names       map[string]string `json:"names"`
	Alias       string            `json:"alias"`
	Description string            `json:"description"`
	GameIDs     []string          `json:"gameIds"`
	Status      int               `json:"status"`
	Sort        int               `json:"sort"`
	NodeID      string            `json:"nodeId"`
	CreatedAt   int64             `json:"createdAt"`
	UpdatedAt   int64             `json:"updatedAt"`
	CreatedBy   string            `json:"createdBy"`
	UpdatedBy   string            `json:"updatedBy"`
	Icon        string            `json:"icon"`
	Logo        string            `json:"logo"`
	ColorLogo   string            `json:"colorLogo"`
}

func GetCapBrand(client *httpClient.Client, request *httpClient.Request[GetCapBrandRequestPathParams]) (*httpClient.Response[GetCapBrandResponseBody], error) {
	request.Path = fmt.Sprintf("%s/%s", basePath, request.PathParams["UUID"])
	request.Method = http.MethodGet

	response, err := httpClient.DoRequest[GetCapBrandRequestPathParams, GetCapBrandResponseBody](client, request)
	if err != nil {
		return nil, fmt.Errorf("GetCapBrand failed: %v", err)
	}

	return response, nil
}
