package http

import (
	"CB_auto/test/config"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	httpClient *http.Client
	serviceURL string
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

func DoRequest[T Params, V any](c *Client, method string, p T) (*V, error) {
	req, err := makeRequest(method, c.serviceURL, p)
	if err != nil {
		return nil, fmt.Errorf("makeRequest failed: %v", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("httpClient.Do failed: %v", err)
	}
	defer resp.Body.Close()

	var result V
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("response decode failed: %v", err)
	}

	return &result, nil
}

func makeRequest[T Params](method string, serviceURL string, p T) (*http.Request, error) {
	var body []byte
	reqURI := p.GetPath()
	queryParams := p.GetQueryParams()

	if method == http.MethodPost || method == http.MethodPut {
		body = p.GetBody()
	}

	if len(queryParams) > 0 {
		query := "?"
		for key, value := range queryParams {
			query += fmt.Sprintf("%s=%s&", key, value)
		}
		reqURI += query[:len(query)-1]
	}

	req, err := http.NewRequest(method, fmt.Sprint(serviceURL, reqURI), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %v", err)
	}

	headers := p.GetQueryHeaders()
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return req, nil
}
