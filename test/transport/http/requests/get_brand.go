package requests

import (
	httpClient "CB_auto/test/transport/http"
	"fmt"
	"net/http"
)

type GetCapBrandRequest struct {
	PathParams GetCapBrandPathParams
	Headers    GetCapBrandHeaders
}

type GetCapBrandPathParams struct {
	UUID string
}

type GetCapBrandHeaders struct {
	Authorization  string
	PlatformNodeID string
}

type GetCapBrandResponse struct {
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

func (p GetCapBrandRequest) GetPath() string {
	return "/_cap/api/v1/brands/" + p.PathParams.UUID
}

func (p GetCapBrandRequest) GetQueryParams() map[string]string {
	return nil
}

func (p GetCapBrandRequest) GetBody() []byte {
	return nil
}

func (p GetCapBrandRequest) GetQueryHeaders() map[string]string {
	return map[string]string{
		"Authorization":   p.Headers.Authorization,
		"platform-nodeid": p.Headers.PlatformNodeID,
	}
}

func (p GetCapBrandRequest) GetPathParams() map[string]string {
	return map[string]string{
		"UUID": p.PathParams.UUID,
	}
}

func GetCapBrand(client *httpClient.Client, request GetCapBrandRequest) (*GetCapBrandResponse, error) {
	response, err := httpClient.DoRequest[GetCapBrandRequest, GetCapBrandResponse](client, http.MethodGet, request)
	if err != nil {
		return nil, fmt.Errorf("GetCapBrand failed: %v", err)
	}

	return response, nil
}
