package nats

import (
	"github.com/google/uuid"
)

type EventType string

const (
	WalletCreated  EventType = "wallet_created"
	WalletDisabled EventType = "wallet_disabled"
	BlockersSetted EventType = "setting_prevent_gamble_setted"
)

const (
	StatusEnabled  WalletStatus = 1
	StatusDisabled WalletStatus = 2

	TypeReal  WalletType = 1
	TypeBonus WalletType = 2
)

type EventPayload struct {
	Type EventType `json:"type"`
	Data any       `json:"data"`
}

type Event struct {
	PlayerUUID uuid.UUID
	WalletUUID uuid.UUID
	SeqNumber  uint64
	Payload    EventPayload
}

type (
	WalletStatus int8
	WalletType   int8
)

type WalletCreatedPayload struct {
	WalletUUID      string       `json:"wallet_uuid"`
	PlayerUUID      string       `json:"player_uuid"`
	PlayerBonusUUID string       `json:"player_bonus_uuid"`
	NodeUUID        string       `json:"node_uuid"`
	BonusCategory   string       `json:"bonus_category"`
	Currency        string       `json:"currency"`
	WalletType      WalletType   `json:"wallet_type"`
	WalletStatus    WalletStatus `json:"wallet_status"`
	Balance         string       `json:"balance"`
	CreatedAt       int64        `json:"created_at"`
	UpdatedAt       int64        `json:"updated_at"`
	IsDefault       bool         `json:"is_default"`
	IsBasic         bool         `json:"is_basic"`
}

type SetDefaultStartedPayload struct {
	UUID      string `json:"uuid"`
	CreatedAt int64  `json:"created_at"`
}

type DefaultUnsettedPayload struct {
	UUID      string `json:"uuid"`
	CreatedAt int64  `json:"created_at"`
}

type DefaultSettedPayload struct {
	UUID      string `json:"uuid"`
	CreatedAt int64  `json:"created_at"`
}

type WalletDisabledPayload struct {
	CreatedAt int64 `json:"created_at"`
}

type BlockersSettedPayload struct {
	IsGamblingActive bool  `json:"is_gambling_active"`
	IsBettingActive  bool  `json:"is_betting_active"`
	CreatedAt        int64 `json:"created_at"`
}

type LimitChangedV2 struct {
	EventType string       `json:"event_type"` // created, updated, deleted
	Limits    []LimitEvent `json:"limits"`
}

type LimitEvent struct {
	ID           string `json:"id"`
	PlayerID     string `json:"playerId"`
	LimitType    string `json:"limitType"`
	IntervalType string `json:"intervalType,omitempty"`
	Amount       string `json:"amount"`
	CurrencyCode string `json:"currencyCode"`
	Status       bool   `json:"status"`
	StartedAt    int64  `json:"startedAt"`
	ExpiresAt    *int64 `json:"expiresAt"`
}

const (
	// Event Types
	EventTypeCreated = "created"
	EventTypeUpdated = "updated"
	EventTypeDeleted = "deleted"

	// Limit Types
	LimitTypeSingleBet     = "single-bet"
	LimitTypeCasinoLoss    = "casino-loss"
	LimitTypeTurnoverFunds = "turnover-of-funds"

	// Interval Types
	IntervalTypeDaily   = "daily"
	IntervalTypeWeekly  = "weekly"
	IntervalTypeMonthly = "monthly"
)
