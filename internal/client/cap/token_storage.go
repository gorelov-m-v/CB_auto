package cap

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	httpClient "CB_auto/internal/client"
	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/config"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

type TokenStorage struct {
	mu sync.RWMutex

	token     string
	refToken  string
	expiresAt time.Time

	client   CapAPI
	config   *config.Config
	provider provider.T
}

type jwtClaims struct {
	ExpiresAt int64 `json:"exp"`
}

func parseJWT(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to decode payload: %v", err)
	}

	var claims jwtClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, fmt.Errorf("failed to parse claims: %v", err)
	}

	return time.Unix(claims.ExpiresAt, 0), nil
}

func NewTokenStorage(t provider.T, cfg *config.Config, client CapAPI) *TokenStorage {
	storage := &TokenStorage{
		client:   client,
		config:   cfg,
		provider: t,
	}
	storage.refreshToken()
	return storage
}

func (s *TokenStorage) GetToken() string {
	s.mu.RLock()
	if time.Until(s.expiresAt) > 3*time.Minute {
		token := s.token
		s.mu.RUnlock()
		return token
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if time.Until(s.expiresAt) > 3*time.Minute {
		return s.token
	}

	s.refreshToken()
	return s.token
}

func (s *TokenStorage) refreshToken() {
	authReq := &httpClient.Request[models.AdminCheckRequestBody]{
		Body: &models.AdminCheckRequestBody{
			UserName: s.config.HTTP.CapUsername,
			Password: s.config.HTTP.CapPassword,
		},
	}

	authResp := s.client.CheckAdmin(authReq)
	s.token = authResp.Body.Token
	s.refToken = authResp.Body.RefreshToken

	expiresAt, err := parseJWT(authResp.Body.Token)
	if err != nil {
		s.provider.Errorf("Failed to parse token expiration: %v", err)
		s.expiresAt = time.Now().Add(30 * time.Minute)
	} else {
		s.expiresAt = expiresAt
	}
}
