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

type RequestDetails struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    json.RawMessage   `json:"body,omitempty"`
}

type ResponseDetails struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       json.RawMessage   `json:"body,omitempty"`
}

type Client struct {
	httpClient *http.Client
	serviceURL string
}

type Request[T any] struct {
	Method      string
	Path        string
	PathParams  map[string]string
	QueryParams map[string]string
	Headers     map[string]string
	Body        *T
}

type Response[V any] struct {
	Body       V
	StatusCode int
	Headers    http.Header
	Error      *ErrorResponse
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

func DoRequest[T any, V any](c *Client, r *Request[T]) (*Response[V], *RequestDetails, *ResponseDetails, error) {
	req, err := makeRequest(c.serviceURL, r)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("makeRequest failed: %v", err)
	}

	var reqBodyBytes json.RawMessage
	if r.Body != nil {
		reqBodyBytes, err = json.Marshal(r.Body)
		if err != nil {
			reqBodyBytes = []byte{}
		}
	}
	reqDetails := &RequestDetails{
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: r.Headers,
		Body:    reqBodyBytes,
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, reqDetails, nil, fmt.Errorf("httpClient.Do failed: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, reqDetails, nil, fmt.Errorf("failed to read response body: %v", err)
	}

	respDetails := &ResponseDetails{
		StatusCode: resp.StatusCode,
		Headers:    convertHeaders(resp.Header),
		Body:       bodyBytes,
	}

	response := &Response[V]{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
	}

	if resp.StatusCode >= 400 {
		response.Error = &ErrorResponse{Body: string(bodyBytes)}
		return response, reqDetails, respDetails, nil
	}

	if err := json.Unmarshal(bodyBytes, &response.Body); err != nil {
		return nil, reqDetails, respDetails, fmt.Errorf("failed to decode response body: %v", err)
	}

	return response, reqDetails, respDetails, nil
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

func convertHeaders(headers http.Header) map[string]string {
	result := make(map[string]string)
	for k, v := range headers {
		result[k] = strings.Join(v, ", ")
	}
	return result
}
