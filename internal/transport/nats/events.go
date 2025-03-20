package nats

import (
	"github.com/google/uuid"
)

type EventType string
type TransactionStatus int
type LimitEventType string

const (
	// Event Types
	WalletCreatedType      EventType = "wallet_created"
	WalletDisabledType     EventType = "wallet_disabled"
	BlockersSettedType     EventType = "setting_prevent_gamble_setted"
	BalanceAdjustedType    EventType = "balance_adjusted"
	BlockAmountStartedType EventType = "block_amount_started"
	BlockAmountRevokedType EventType = "block_amount_revoked"
	DepositedMoneyType     EventType = "deposited_money"
	LimitChangedV2Type     EventType = "limit_changed_v2"

	TransactionStatusSuccess TransactionStatus = 4
	LimitEventAmountUpdated  LimitEventType    = "amount_updated"
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
	CreatedAt       int          `json:"created_at"`
	UpdatedAt       int          `json:"updated_at"`
	IsDefault       bool         `json:"is_default"`
	IsBasic         bool         `json:"is_basic"`
}

type SetDefaultStartedPayload struct {
	UUID      string `json:"uuid"`
	CreatedAt int    `json:"created_at"`
}

type DefaultUnsettedPayload struct {
	UUID      string `json:"uuid"`
	CreatedAt int    `json:"created_at"`
}

type DefaultSettedPayload struct {
	UUID      string `json:"uuid"`
	CreatedAt int    `json:"created_at"`
}

type WalletDisabledPayload struct {
	CreatedAt int `json:"created_at"`
}

type BlockersSettedPayload struct {
	IsGamblingActive bool `json:"is_gambling_active"`
	IsBettingActive  bool `json:"is_betting_active"`
	CreatedAt        int  `json:"created_at"`
}

type LimitChangedV2 struct {
	EventType string `json:"event_type"`
	Limits    []struct {
		ExternalID   string `json:"external_id"`
		LimitType    string `json:"limit_type"`
		IntervalType string `json:"interval_type"`
		Amount       string `json:"amount"`
		CurrencyCode string `json:"currency_code"`
		StartedAt    int    `json:"started_at"`
		ExpiresAt    int    `json:"expires_at"`
		Status       bool   `json:"status"`
	} `json:"limits"`
}

const (
	// Event Types
	EventTypeCreated = "created"
	EventTypeUpdated = "amount_updated"
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

type BalanceAdjustedPayload struct {
	Currenc       string `json:"currenc"` // Полный провал. Такая вот опечатка у нас :]
	UUID          string `json:"uuid"`
	Amount        string `json:"amount"`
	OperationType int    `json:"operation_type"`
	Direction     int    `json:"direction"`
	Reason        int    `json:"reason"`
	Comment       string `json:"comment"`
	UserUUID      string `json:"user_uuid"`
	UserName      string `json:"user_name"`
}

type BlockAmountStartedPayload struct {
	UUID      string `json:"uuid"`
	Status    int    `json:"status"`
	Amount    string `json:"amount"`
	Reason    string `json:"reason"`
	Type      int    `json:"type"`
	ExpiredAt int64  `json:"expired_at"`
	UserUUID  string `json:"user_uuid"`
	UserName  string `json:"user_name"`
	CreatedAt int64  `json:"created_at"`
}

type BlockAmountRevokedPayload struct {
	UUID     string `json:"uuid"`
	UserUUID string `json:"user_uuid"`
	UserName string `json:"user_name"`
	NodeUUID string `json:"node_uuid"`
}

type DepositedMoneyPayload struct {
	UUID         string            `json:"uuid"`
	CurrencyCode string            `json:"currency_code"`
	Amount       string            `json:"amount"`
	Status       TransactionStatus `json:"status"`
	NodeUUID     string            `json:"node_uuid"`
	BonusID      string            `json:"bonus_id"`
}
