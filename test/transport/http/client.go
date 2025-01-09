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
	Method      string
	Path        string
	PathParams  map[string]string
	QueryParams map[string]string
	Headers     map[string]string
	Body        *T
}

const (
	AuthorizationHeader = "Authorization"
	PlatformNodeHeader  = "Platform-Nodeid"
)

type Response[V any] struct {
	Body       V
	StatusCode int
	Headers    http.Header
	Error      *ErrorResponse
}

type ErrorResponse struct {
	Body string `json:"body"`
}

func InitClient(config *config.Config) (*Client, error) {
	u, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		serviceURL: u.String(),
		httpClient: &http.Client{
			Timeout: time.Duration(config.RequestTimeout) * time.Second,
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

	bodyString := string(bodyBytes)

	response := &Response[V]{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
	}

	if resp.StatusCode >= 400 {
		response.Error = &ErrorResponse{
			Body: bodyString,
		}
		return response, nil
	}

	if err := json.Unmarshal(bodyBytes, &response.Body); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %v", err)
	}

	return response, nil
}

func FormatRequest[T any](request *Request[T]) []byte {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("URL: %s\n", request.Path))

	if len(request.Headers) > 0 {
		var headerStrings []string
		for key, value := range request.Headers {
			headerStrings = append(headerStrings, fmt.Sprintf("%s: %s", key, value))
		}
		result.WriteString(fmt.Sprintf("Request Headers:\n%s\n", strings.Join(headerStrings, ", ")))
	}

	if request.Body != nil {
		requestBody, err := json.MarshalIndent(request.Body, "", "  ")
		if err == nil {
			result.WriteString(fmt.Sprintf("Request Body:\n%s\n", string(requestBody)))
		} else {
			result.WriteString("Request Body: <failed to marshal body>\n")
		}
	}

	return []byte(result.String())
}

func FormatResponse[V any](response *Response[V]) []byte {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("StatusCode: %d\n", response.StatusCode))

	if response.Error != nil {
		result.WriteString(fmt.Sprintf("Error Body: %s\n", response.Error.Body))
	} else {
		responseBody, err := json.MarshalIndent(response.Body, "", "  ")
		if err == nil {
			result.WriteString(fmt.Sprintf("Response Body:\n%s\n", string(responseBody)))
		} else {
			result.WriteString("Response Body: <failed to marshal body>\n")
		}
	}

	return []byte(result.String())
}

func makeRequest[T any](serviceURL string, r *Request[T]) (*http.Request, error) {
	path := r.Path
	if len(r.PathParams) > 0 {
		for key, value := range r.PathParams {
			placeholder := fmt.Sprintf("{%s}", key)
			path = strings.Replace(path, placeholder, value, -1)
		}
	}

	var body []byte
	if r.Body != nil {
		var err error
		body, err = json.Marshal(r.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
	}

	fullURL := fmt.Sprintf("%s%s", serviceURL, path)
	if len(r.QueryParams) > 0 {
		query := url.Values{}
		for key, value := range r.QueryParams {
			query.Add(key, value)
		}
		fullURL = fmt.Sprintf("%s?%s", fullURL, query.Encode())
	}

	req, err := http.NewRequest(r.Method, fullURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %v", err)
	}

	for key, value := range r.Headers {
		req.Header.Set(key, value)
	}

	return req, nil
}
