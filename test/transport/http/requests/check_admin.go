package requests

import (
	httpClient "CB_auto/test/transport/http"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type AdminCheckRequest struct {
	Body AdminCheckBody
}

type AdminCheckBody struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

type AdminCheckResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

func (p AdminCheckRequest) GetPath() string {
	return "/_cap/api/token/check"
}

func (p AdminCheckRequest) GetQueryParams() map[string]string {
	return nil
}

func (p AdminCheckRequest) GetBody() []byte {
	bodyBytes, err := json.Marshal(p.Body)
	if err != nil {
		log.Printf("getBody marshal failed: %v", err)
		return nil
	}

	return bodyBytes
}

func (p AdminCheckRequest) GetQueryHeaders() map[string]string {
	return nil
}

func (p AdminCheckRequest) GetPathParams() map[string]string {
	return nil
}

func CheckAdmin(client *httpClient.Client, request AdminCheckRequest) (*AdminCheckResponse, error) {
	response, err := httpClient.DoRequest[AdminCheckRequest, AdminCheckResponse](client, http.MethodPost, request)
	if err != nil {
		return nil, fmt.Errorf("CheckAdmin failed: %v", err)
	}

	return response, nil
}
