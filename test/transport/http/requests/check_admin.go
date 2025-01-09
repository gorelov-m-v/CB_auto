package requests

import (
	httpClient "CB_auto/test/transport/http"
	"fmt"
	"net/http"
)

const (
	pathAdminCheck = "/_cap/api/token/check"
)

type AdminCheckRequest struct {
	Body AdminCheckRequestBody
}

type AdminCheckResponse struct {
	Body       AdminCheckResponseBody
	StatusCode int
}

type AdminCheckRequestBody struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

type AdminCheckResponseBody struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

func CheckAdmin(client *httpClient.Client, request *httpClient.Request[AdminCheckRequestBody]) (*httpClient.Response[AdminCheckResponseBody], error) {
	request.Path = pathAdminCheck
	request.Method = http.MethodPost

	response, err := httpClient.DoRequest[AdminCheckRequestBody, AdminCheckResponseBody](client, request)
	if err != nil {
		return nil, fmt.Errorf("CheckAdmin failed: %v", err)
	}

	return response, nil
}
