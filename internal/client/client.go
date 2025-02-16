package http

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

	"CB_auto/internal/config"
)

type Client struct {
	httpClient *http.Client
	serviceURL string
}

type Request[T any] struct {
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	PathParams  map[string]string `json:"path_params,omitempty"`
	QueryParams map[string]string `json:"query_params,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        *T                `json:"body,omitempty"`
}

type Response[V any] struct {
	Body       V              `json:"body"`
	StatusCode int            `json:"status_code"`
	Headers    http.Header    `json:"headers"`
	Error      *ErrorResponse `json:"error,omitempty"`
}

type ErrorResponse struct {
	Body string `json:"body"`
}

type ClientType string

const (
	Cap    ClientType = "cap"
	Public ClientType = "public"
)

func InitClient(cfg *config.Config, clientType ClientType) (*Client, error) {
	var baseURL string
	switch clientType {
	case Cap:
		baseURL = cfg.HTTP.CapURL
	case Public:
		baseURL = cfg.HTTP.PublicURL
	default:
		return nil, fmt.Errorf("unknown client type: %s", clientType)
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		serviceURL: u.String(),
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.HTTP.Timeout) * time.Second,
		},
	}, nil
}

func DoRequest[T any, V any](c *Client, r *Request[T]) (*Response[V], error) {
	req, err := makeRequest(c.serviceURL, r)
	if err != nil {
		return nil, fmt.Errorf("makeRequest failed: %v", err)
	}

	log.Printf("Request URL: %s, Method: %s, Headers: %+v", req.URL.String(), req.Method, req.Header)
	if r.Body != nil {
		bodyBytes, _ := json.Marshal(r.Body)
		log.Printf("Request Body: %s", string(bodyBytes))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("httpClient.Do failed: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	log.Printf("Response Status: %d, Headers: %+v, Body: %s", resp.StatusCode, resp.Header, string(bodyBytes))

	response := &Response[V]{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
	}

	if resp.StatusCode >= 400 {
		response.Error = &ErrorResponse{Body: string(bodyBytes)}
		return response, nil
	}

	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &response.Body); err != nil {
			return nil, fmt.Errorf("failed to decode response body: %v", err)
		}
	}

	return response, nil
}

func makeRequest[T any](serviceURL string, r *Request[T]) (*http.Request, error) {
	path := r.Path
	if len(r.PathParams) > 0 {
		for key, value := range r.PathParams {
			placeholder := fmt.Sprintf("{%s}", key)
			path = strings.ReplaceAll(path, placeholder, value)
		}
	}

	baseURL, err := url.Parse(serviceURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base url: %v", err)
	}

	relURL, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %v", err)
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
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
	}

	req, err := http.NewRequest(r.Method, fullURL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %v", err)
	}

	for key, value := range r.Headers {
		req.Header.Set(key, value)
	}

	return req, nil
}
