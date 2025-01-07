package requests

import (
	httpClient "CB_auto/test/transport/http"
	"fmt"
)

const (
	pathAdminCheck   = "/_cap/api/token/check"
	methodAdminCheck = "POST"
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

func CheckAdmin1(client *httpClient.Client, request *httpClient.Request[AdminCheckRequestBody]) (*httpClient.Response[AdminCheckResponseBody], error) {
	request.Path = pathAdminCheck
	request.Method = methodAdminCheck
	response, err := httpClient.DoRequest1[AdminCheckRequestBody, AdminCheckResponseBody](client, request)
	if err != nil {
		return nil, fmt.Errorf("CheckAdmin failed: %v", err)
	}

	return response, nil
}
