package types

import (
	"net/http"
)

type Client struct {
	HttpClient *http.Client
	ServiceURL string
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
