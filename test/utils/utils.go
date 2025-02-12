package utils

import (
	"CB_auto/test/transport/http"
	"encoding/json"
	"fmt"
	"strings"
)

func CreateHttpAttachRequest[T any](req *http.Request[T]) []byte {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Method: %s\n", req.Method))
	sb.WriteString(fmt.Sprintf("Path: %s\n", req.Path))
	if len(req.PathParams) > 0 {
		sb.WriteString("PathParams:\n")
		for k, v := range req.PathParams {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}
	if len(req.QueryParams) > 0 {
		sb.WriteString("QueryParams:\n")
		for k, v := range req.QueryParams {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}
	if len(req.Headers) > 0 {
		sb.WriteString("Headers:\n")
		for k, v := range req.Headers {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}
	if req.Body != nil {
		b, err := json.MarshalIndent(req.Body, "", "  ")
		if err != nil {
			sb.WriteString(fmt.Sprintf("Body: %+v\n", req.Body))
		} else {
			sb.WriteString("Body: " + string(b) + "\n")
		}
	}
	return []byte(sb.String())
}

func CreateHttpAttachResponse[V any](resp *http.Response[V]) []byte {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("StatusCode: %d\n", resp.StatusCode))
	if len(resp.Headers) > 0 {
		sb.WriteString("Headers:\n")
		for k, v := range resp.Headers {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, strings.Join(v, ", ")))
		}
	}
	if resp.Error != nil {
		sb.WriteString(fmt.Sprintf("Error: %s\n", resp.Error.Body))
	} else {
		b, err := json.MarshalIndent(resp.Body, "", "  ")
		if err != nil {
			sb.WriteString(fmt.Sprintf("Body: %+v\n", resp.Body))
		} else {
			sb.WriteString("Body: " + string(b) + "\n")
		}
	}
	return []byte(sb.String())
}

func CreatePrettyJSON[T any](v T) []byte {
	prettyJSON, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		return []byte(fmt.Sprintf("Ошибка при форматировании JSON: %v", err))
	}
	return prettyJSON
}
