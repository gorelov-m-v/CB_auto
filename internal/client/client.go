package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
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

func makeRequest[T any](serviceURL string, r *types.Request[T]) (*http.Request, []byte, error) {
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

	var bodyBytes []byte
	var req *http.Request

	if r.Multipart != nil {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		for name, value := range r.Multipart.Fields {
			if err := writer.WriteField(name, value); err != nil {
				return nil, nil, fmt.Errorf("failed to write field %s: %v", name, err)
			}
		}

		for field, file := range r.Multipart.Files {
			part, err := writer.CreateFormFile(field, file.Filename)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create form file %s: %v", field, err)
			}
			if _, err := part.Write(file.Data); err != nil {
				return nil, nil, fmt.Errorf("failed to write file data for %s: %v", field, err)
			}
		}

		if err := writer.Close(); err != nil {
			return nil, nil, fmt.Errorf("failed to close multipart writer: %v", err)
		}

		bodyBytes = body.Bytes()
		req, err = http.NewRequest(r.Method, fullURL.String(), bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, nil, fmt.Errorf("request creation failed: %v", err)
		}

		req.Header.Set("Content-Type", writer.FormDataContentType())
	} else {
		var bodyBuffer io.Reader
		if r.Body != nil {
			bodyBytes, err = json.Marshal(r.Body)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to marshal request body: %v", err)
			}
			bodyBuffer = bytes.NewBuffer(bodyBytes)
		}

		req, err = http.NewRequest(r.Method, fullURL.String(), bodyBuffer)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create request: %v", err)
		}

		if r.Body != nil && req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	for key, value := range r.Headers {
		req.Header.Set(key, value)
	}

	return req, bodyBytes, nil
}

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

func DoRequest[T any, V any](sCtx provider.StepCtx, c *types.Client, request *types.Request[T]) *types.Response[V] {
	req, bodyBytes, err := makeRequest(c.ServiceURL, request)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return &types.Response[V]{
			Error: &types.ErrorResponse{Body: err.Error()},
		}
	}

	log.Printf("Request URL: %s, Method: %s, Headers: %+v", req.URL.String(), req.Method, req.Header)
	if len(bodyBytes) > 0 {
		log.Printf("Request Body: %s", string(bodyBytes))
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		log.Printf("HTTP request failed: %v", err)
		return &types.Response[V]{
			Error: &types.ErrorResponse{Body: err.Error()},
		}
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return &types.Response[V]{
			Error: &types.ErrorResponse{Body: err.Error()},
		}
	}

	log.Printf("Response Status: %d, Headers: %+v, Body: %s", resp.StatusCode, resp.Header, string(responseBody))

	response := &types.Response[V]{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
	}

	if resp.StatusCode >= 400 {
		response.Error = &types.ErrorResponse{
			Body: string(responseBody),
		}
		return response
	}

	if len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, &response.Body); err != nil {
			log.Printf("Failed to decode response body: %v", err)
			response.Error = &types.ErrorResponse{Body: err.Error()}
		}
	}

	sCtx.WithAttachments(allure.NewAttachment("HTTP request", allure.JSON, utils.CreateHttpAttachRequest(request)))
	sCtx.WithAttachments(allure.NewAttachment("HTTP response", allure.JSON, utils.CreateHttpAttachResponse(response)))

	return response
}
