package models

// ErrorResponse структура для ответа с ошибкой
type ErrorResponse struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors"`
}
