package cap

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"CB_auto/internal/client/cap/models"
	"CB_auto/internal/client/types"
	"CB_auto/internal/config"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

type TokenStorage struct {
	mu        sync.RWMutex
	token     string
	refToken  string
	expiresAt time.Time
	client    CapAPI
	config    *config.Config
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

func NewTokenStorage(sCtx provider.StepCtx, cfg *config.Config, client CapAPI) *TokenStorage {
	storage := &TokenStorage{
		client: client,
		config: cfg,
	}
	storage.refreshToken(sCtx)
	return storage
}

func (s *TokenStorage) GetToken(sCtx provider.StepCtx) string {
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

	s.refreshToken(sCtx)
	return s.token
}

func (s *TokenStorage) refreshToken(sCtx provider.StepCtx) {
	req := &types.Request[models.AdminCheckRequestBody]{
		Body: &models.AdminCheckRequestBody{
			UserName: s.config.HTTP.CapUsername,
			Password: s.config.HTTP.CapPassword,
		},
	}

	res := s.client.CheckAdmin(sCtx, req)
	s.token = res.Body.Token
	s.refToken = res.Body.RefreshToken

	expiresAt, err := parseJWT(res.Body.Token)
	if err != nil {
		sCtx.Errorf("Failed to parse token expiration: %v", err)
		s.expiresAt = time.Now().Add(30 * time.Minute)
	} else {
		s.expiresAt = expiresAt
	}
}
