package kafka

import (
	"encoding/json"
)

type Brand struct {
	Message struct {
		EventType string `json:"eventType"`
	} `json:"message"`
	Brand struct {
		UUID           string            `json:"uuid"`
		LocalizedNames map[string]string `json:"localized_names"`
		Alias          string            `json:"alias"`
		ProjectID      string            `json:"project_id"`
		StatusEnabled  bool              `json:"status_enabled"`
		CreatedAt      int64             `json:"created_at"`
	} `json:"brand"`
}

type PlayerMessage struct {
	Message struct {
		EventType      string `json:"eventType"`
		EventCreatedAt int64  `json:"eventCreatedAt"`
	} `json:"message"`
	Player struct {
		ID             int    `json:"id,omitempty"`
		NodeID         string `json:"nodeId,omitempty"`
		ProjectGroupID string `json:"projectGroupId,omitempty"`
		ExternalID     string `json:"externalId,omitempty"`
		AccountID      string `json:"accountId,omitempty"`
		Country        string `json:"country,omitempty"`
		Currency       string `json:"currency,omitempty"`
		CreatedAt      int64  `json:"createdAt,omitempty"`
	} `json:"player"`
	Context json.RawMessage `json:"context"`
}
