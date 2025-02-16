package nats

import (
	"github.com/google/uuid"
)

type EventType string

const (
	WalletCreated EventType = "wallet_created"
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
	Currency        string       `json:"currency"`
	WalletType      WalletType   `json:"wallet_type"`
	WalletStatus    WalletStatus `json:"wallet_status"`
	Balance         string       `json:"balance"`
	CreatedAt       int64        `json:"created_at"`
	UpdatedAt       int64        `json:"updated_at"`
	IsDefault       bool         `json:"is_default"`
	IsBasic         bool         `json:"is_basic"`
}
