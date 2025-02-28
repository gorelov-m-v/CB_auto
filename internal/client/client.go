package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"CB_auto/internal/client/types"
	"CB_auto/internal/config"
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
)

func InitClient(t provider.T, cfg *config.Config, clientType types.ClientType) *types.Client {
	var baseURL string
	switch clientType {
	case types.Cap:
		baseURL = cfg.HTTP.CapURL
	case types.Public:
		baseURL = cfg.HTTP.PublicURL
	default:
		t.Fatalf("Неизвестный тип клиента: %s", clientType)
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("Ошибка при парсинге URL: %v", err)
	}

	return &types.Client{
		ServiceURL: u.String(),
		HttpClient: &http.Client{
			Timeout: time.Duration(cfg.HTTP.Timeout) * time.Second,
		},
	}
}

func DoRequest[T any, V any](sCtx provider.StepCtx, c *types.Client, request *types.Request[T]) (*types.Response[V], error) {
	req, bodyBytes, err := makeRequest(c.ServiceURL, request)
	if err != nil {
		return nil, fmt.Errorf("makeRequest failed: %v", err)
	}

	log.Printf("Request URL: %s, Method: %s, Headers: %+v", req.URL.String(), req.Method, req.Header)
	if len(bodyBytes) > 0 {
		log.Printf("Request Body: %s", string(bodyBytes))
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("httpClient.Do failed: %v", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	log.Printf("Response Status: %d, Headers: %+v, Body: %s", resp.StatusCode, resp.Header, string(responseBody))

	response := &types.Response[V]{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
	}

	if resp.StatusCode >= 400 {
		response.Error = &types.ErrorResponse{Body: string(responseBody)}
		return response, nil
	}

	if len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, &response.Body); err != nil {
			return nil, fmt.Errorf("failed to decode response body: %v", err)
		}
	}

	sCtx.WithAttachments(allure.NewAttachment("HTTP request", allure.JSON, utils.CreateHttpAttachRequest(request)))
	sCtx.WithAttachments(allure.NewAttachment("HTTP response", allure.JSON, utils.CreateHttpAttachResponse(response)))

	return response, nil
}

func makeRequest[T any](serviceURL string, r *types.Request[T]) (*http.Request, []byte, error) {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}

	path := r.Path
	if len(r.PathParams) > 0 {
		for key, value := range r.PathParams {
			placeholder := fmt.Sprintf("{%s}", key)
			path = strings.ReplaceAll(path, placeholder, url.PathEscape(value))
		}
	}

	baseURL, err := url.Parse(serviceURL)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid base url: %v", err)
	}

	relURL, err := url.Parse(path)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid path: %v", err)
	}

	fullURL := baseURL.ResolveReference(relURL)

	if len(r.QueryParams) > 0 {
		query := fullURL.Query()
		for key, value := range r.QueryParams {
			query.Set(key, value)
		}
		fullURL.RawQuery = query.Encode()
	}

	var body []byte
	if r.Body != nil {
		body, err = json.Marshal(r.Body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
		if _, exists := r.Headers["Content-Type"]; !exists {
			r.Headers["Content-Type"] = "application/json"
		}
	}

	req, err := http.NewRequest(r.Method, fullURL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("request creation failed: %v", err)
	}

	for key, value := range r.Headers {
		req.Header.Set(key, value)
	}

	return req, body, nil
}
