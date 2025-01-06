package http

type Params interface {
	GetPath() string
	GetQueryParams() map[string]string
	GetBody() []byte
	GetQueryHeaders() map[string]string
	GetPathParams() map[string]string
}
