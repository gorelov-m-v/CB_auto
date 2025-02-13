package public

import (
	httpClient "CB_auto/test/transport/http"
	"CB_auto/test/transport/http/public/models"
	"fmt"
)

type PublicAPI interface {
	FastRegistration(req *httpClient.Request[models.FastRegistrationRequestBody]) *httpClient.Response[models.FastRegistrationResponseBody]
}

type publicClient struct {
	client *httpClient.Client
}

func NewPublicClient(client *httpClient.Client) PublicAPI {
	return &publicClient{client: client}
}

func (c *publicClient) FastRegistration(req *httpClient.Request[models.FastRegistrationRequestBody]) *httpClient.Response[models.FastRegistrationResponseBody] {
	req.Method = "POST"
	req.Path = "/_front_api/api/v1/registration/fast"
	resp, err := httpClient.DoRequest[models.FastRegistrationRequestBody, models.FastRegistrationResponseBody](c.client, req)
	if err != nil {
		panic(fmt.Sprintf("FastRegistration не удался: %v", err))
	}
	return resp
}
