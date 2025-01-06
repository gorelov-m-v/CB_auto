package utils

import (
	"CB_auto/test/config"
	"CB_auto/test/transport/http"
	"CB_auto/test/transport/http/requests"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func CheckTokenExpiry(response *requests.AdminCheckResponse) bool {
	if response == nil || response.Token == "" {
		return false
	}

	parts := strings.Split(response.Token, ".")
	if len(parts) != 3 {
		fmt.Println("Invalid token format")
		return false
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		fmt.Printf("Failed to decode token payload: %v\n", err)
		return false
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		fmt.Printf("Failed to unmarshal token payload: %v\n", err)
		return false
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		fmt.Println("Token does not contain an expiration time")
		return false
	}

	expirationTime := time.Unix(int64(exp), 0)
	if time.Until(expirationTime) <= 0 {
		fmt.Println("Token has expired")
		return false
	}

	fmt.Printf("Token is valid until: %v\n", expirationTime)
	return true
}

func FormatRequest(request http.Params, config *config.Config) (string, error) {
	bodyJSON := request.GetBody()
	headers := request.GetQueryHeaders()
	var headerStrings []string
	for key, value := range headers {
		headerStrings = append(headerStrings, fmt.Sprintf("%s: %s", key, value))
	}

	result := fmt.Sprintf("Method:\nPOST %s%s\n\nHeaders:\n%s\n\nBody:\n%s",
		config.BaseURL, request.GetPath(), strings.Join(headerStrings, "\n"), string(bodyJSON))

	return result, nil
}
