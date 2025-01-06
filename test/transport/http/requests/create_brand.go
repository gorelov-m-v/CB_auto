package requests

import (
	httpClient "CB_auto/test/transport/http"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type CreateCapBrandRequest struct {
	Headers CreateCapBrandHeaders
	Body    CreateCapBrandBody
}

type CreateCapBrandHeaders struct {
	Authorization  string `json:"Authorization"`
	PlatformNodeID string `json:"platform-nodeid"`
}

type CreateCapBrandBody struct {
	Sort        int               `json:"sort"`
	Alias       string            `json:"alias"`
	Names       map[string]string `json:"names"`
	Description string            `json:"description"`
}

type CreateCapBrandResponse struct {
	ID string `json:"id"`
}

func (p CreateCapBrandRequest) GetPath() string {
	return "/_cap/api/v1/brands"
}

func (p CreateCapBrandRequest) GetQueryParams() map[string]string {
	return nil
}

func (p CreateCapBrandRequest) GetBody() []byte {
	bodyBytes, err := json.Marshal(p.Body)
	if err != nil {
		log.Printf("getBody marshal failed: %v", err)
		return nil
	}

	return bodyBytes
}

func (p CreateCapBrandRequest) GetQueryHeaders() map[string]string {
	return map[string]string{
		"Authorization":   p.Headers.Authorization,
		"platform-nodeid": p.Headers.PlatformNodeID,
	}
}

func (p CreateCapBrandRequest) GetPathParams() map[string]string {
	return nil
}

func CreateCapBrand(client *httpClient.Client, request CreateCapBrandRequest) (*CreateCapBrandResponse, error) {
	response, err := httpClient.DoRequest[CreateCapBrandRequest, CreateCapBrandResponse](client, http.MethodPost, request)
	if err != nil {
		return nil, fmt.Errorf("CreateCapBrand failed: %v", err)
	}

	return response, nil
}
