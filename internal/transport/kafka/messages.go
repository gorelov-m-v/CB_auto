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

type LimitMessage struct {
	IntervalType string `json:"intervalType"`
	LimitType    string `json:"limitType"`
	Amount       string `json:"amount"`
	Spent        string `json:"spent"`
	Rest         string `json:"rest"`
	CurrencyCode string `json:"currencyCode"`
	ID           string `json:"id"`
	PlayerID     string `json:"playerId"`
	Status       bool   `json:"status"`
	StartedAt    int64  `json:"startedAt"`
	ExpiresAt    int64  `json:"expiresAt"`
	EventType    string `json:"eventType"`
}

const (
	// Event Types
	LimitEventCreated = "created"
	LimitEventUpdated = "updated"
	LimitEventDeleted = "deleted"

	// Limit Types
	LimitTypeSingleBet     = "single-bet"
	LimitTypeCasinoLoss    = "casino-loss"
	LimitTypeTurnoverFunds = "turnover-of-funds"

	// Interval Types
	IntervalTypeDaily   = "daily"
	IntervalTypeWeekly  = "weekly"
	IntervalTypeMonthly = "monthly"
)

type ProjectionSourceMessage struct {
	Type              string `json:"type"`
	SeqNumber         int64  `json:"seq_number"`
	WalletUUID        string `json:"wallet_uuid"`
	PlayerUUID        string `json:"player_uuid"`
	NodeUUID          string `json:"node_uuid"`
	Payload           string `json:"payload"`
	Currency          string `json:"currency"`
	Timestamp         int64  `json:"timestamp"`
	SeqNumberNodeUUID string `json:"seq_number_node_uuid"`
}

type ProjectionPayload struct {
	EventType string            `json:"event_type"`
	Limits    []ProjectionLimit `json:"limits"`
}

type ProjectionLimit struct {
	ExternalID   string `json:"external_id"`
	LimitType    string `json:"limit_type"`
	IntervalType string `json:"interval_type"`
	Amount       string `json:"amount"`
	CurrencyCode string `json:"currency_code"`
	StartedAt    int64  `json:"started_at"`
	ExpiresAt    int64  `json:"expires_at"`
	Status       bool   `json:"status"`
}

func (m *ProjectionSourceMessage) UnmarshalPayload() (*ProjectionPayload, error) {
	var payload ProjectionPayload
	err := json.Unmarshal([]byte(m.Payload), &payload)
	return &payload, err
}
