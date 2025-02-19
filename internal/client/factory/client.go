package factory

import (
	"net/http"
	"time"

	"CB_auto/internal/client/cap"
	"CB_auto/internal/client/public"
	"CB_auto/internal/client/types"
	"CB_auto/internal/config"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

func InitClient[T any](t provider.T, cfg *config.Config, clientType types.ClientType) T {
	baseClient := &types.Client{
		HttpClient: &http.Client{
			Timeout: time.Duration(cfg.HTTP.Timeout) * time.Second,
		},
	}

	switch clientType {
	case types.Cap:
		baseClient.ServiceURL = cfg.HTTP.CapURL
		return cap.NewClient(t, cfg, baseClient).(T)
	case types.Public:
		baseClient.ServiceURL = cfg.HTTP.PublicURL
		return public.NewClient(baseClient).(T)
	default:
		t.Fatalf("Неизвестный тип клиента: %s", clientType)
		return *new(T)
	}
}
