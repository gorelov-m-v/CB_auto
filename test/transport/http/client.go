package http

import (
	"CB_auto/test/config"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
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

func InitClient(cfg *config.Config) (*Client, error) {
	u, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		serviceURL: u.String(),
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.RequestTimeout) * time.Second,
		},
	}, nil
}

func DoRequest[T any, V any](c *Client, r *Request[T]) (*Response[V], error) {
	req, err := makeRequest(c.serviceURL, r)
	if err != nil {
		return nil, fmt.Errorf("makeRequest failed: %v", err)
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

	fmt.Printf("Response Status: %d\nResponse Body: %s\n", resp.StatusCode, string(bodyBytes))

	response := &Response[V]{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
	}

	if resp.StatusCode >= 400 {
		response.Error = &ErrorResponse{Body: string(bodyBytes)}
		return response, nil
	}

	if err := json.Unmarshal(bodyBytes, &response.Body); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %v", err)
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
