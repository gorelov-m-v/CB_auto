package types

import (
	"net/http"
)

type Client struct {
	ServiceURL string
	HttpClient *http.Client
}

type ClientType string

const (
	Cap    ClientType = "cap"
	Public ClientType = "public"
)

type Request[T any] struct {
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	PathParams  map[string]string `json:"path_params,omitempty"`
	QueryParams map[string]string `json:"query_params,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        *T                `json:"body,omitempty"`
	Multipart   *MultipartForm    `json:"multipart,omitempty"`
}

type Response[T any] struct {
	StatusCode int            `json:"status_code"`
	Headers    http.Header    `json:"headers,omitempty"`
	Body       T              `json:"body,omitempty"`
	Error      *ErrorResponse `json:"error,omitempty"`
}

type ErrorResponse struct {
	Body       string `json:"body,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
}

type MultipartForm struct {
	Fields   map[string]string   `json:"fields,omitempty"`
	Files    map[string]FileData `json:"files,omitempty"`
	Boundary string              `json:"boundary,omitempty"`
}

type FileData struct {
	Filename    string `json:"filename"`
	Data        []byte `json:"data"`
	ContentType string `json:"content_type,omitempty"`
}

func (r *Request[T]) SetFormField(name, value string) {
	if r.Multipart == nil {
		r.Multipart = &MultipartForm{
			Fields: make(map[string]string),
			Files:  make(map[string]FileData),
		}
	}
	r.Multipart.Fields[name] = value
}

func (r *Request[T]) AddFormFile(fieldName, fileName string, data []byte, contentType string) {
	if r.Multipart == nil {
		r.Multipart = &MultipartForm{
			Fields: make(map[string]string),
			Files:  make(map[string]FileData),
		}
	}
	r.Multipart.Files[fieldName] = FileData{
		Filename:    fileName,
		Data:        data,
		ContentType: contentType,
	}
}

func (r *Request[T]) AddQueryParam(name, value string) {
	if r.QueryParams == nil {
		r.QueryParams = make(map[string]string)
	}
	r.QueryParams[name] = value
}

func (r *Request[T]) AddPathParam(name, value string) {
	if r.PathParams == nil {
		r.PathParams = make(map[string]string)
	}
	r.PathParams[name] = value
}

func (r *Request[T]) AddHeader(name, value string) {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers[name] = value
}
