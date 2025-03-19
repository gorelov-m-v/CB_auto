package kafka

import (
	"encoding/json"
	"errors"
)

type LimitEventType string
type LimitType string
type LimitIntervalType string
type ProjectionEventType string
type PlayerEventType string
type TransactionStatus int
type TransactionDirection string
type PaymentMethodAlias string
type PaymentMethod string

const (
	// Event Types
	LimitEventCreated LimitEventType = "created"
	LimitEventUpdated LimitEventType = "updated"
	LimitEventDeleted LimitEventType = "deleted"

	// Limit Types
	LimitTypeSingleBet     LimitType = "single-bet"
	LimitTypeCasinoLoss    LimitType = "casino-loss"
	LimitTypeTurnoverFunds LimitType = "turnover-of-funds"

	// Limit Interval Types
	IntervalTypeDaily   LimitIntervalType = "daily"
	IntervalTypeWeekly  LimitIntervalType = "weekly"
	IntervalTypeMonthly LimitIntervalType = "monthly"

	// Projection Event Types
	ProjectionEventBalanceAdjusted    ProjectionEventType = "balance_adjusted"
	ProjectionEventLimitChanged       ProjectionEventType = "limit_changed_v2"
	ProjectionEventBlockAmountStarted ProjectionEventType = "block_amount_started"
	ProjectionEventBlockAmountRevoked ProjectionEventType = "block_amount_revoked"
	ProjectionEventDepositedMoney     ProjectionEventType = "deposited_money"

	// Player Event Types
	PlayerEventSignUpFast        PlayerEventType = "player.signUpFast"
	PlayerEventConfirmationPhone PlayerEventType = "player.confirmationPhone"
	PlayerEventConfirmationEmail PlayerEventType = "player.confirmationEmail"

	// Transaction Direction Types
	TransactionDirectionDeposit TransactionDirection = "deposit"

	// Transaction Status Types
	TransactionStatusSuccess TransactionStatus = 4

	// Payment Method Alias Types
	PaymentMethodAliasFake PaymentMethodAlias = "Fake"

	// Payment Method Types
	PaymentMethodFake PaymentMethod = "Fake1001"
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
		CreatedAt      int               `json:"created_at"`
	} `json:"brand"`
}

type PlayerMessage struct {
	Message struct {
		EventType      PlayerEventType `json:"eventType"`
		EventCreatedAt int64           `json:"eventCreatedAt"`
	} `json:"message"`
	Player struct {
		ID             int    `json:"id,omitempty"`
		NodeID         string `json:"nodeId,omitempty"`
		ProjectGroupID string `json:"projectGroupId,omitempty"`
		ExternalID     string `json:"externalId,omitempty"`
		AccountID      string `json:"accountId,omitempty"`
		Country        string `json:"country,omitempty"`
		Currency       string `json:"currency,omitempty"`
		Phone          string `json:"phone,omitempty"`
		Email          string `json:"email,omitempty"`
		Locale         string `json:"locale,omitempty"`
		CreatedAt      int    `json:"createdAt,omitempty"`
	} `json:"player"`
	Context json.RawMessage `json:"context"`
}

type LimitMessage struct {
	IntervalType LimitIntervalType `json:"intervalType"`
	LimitType    LimitType         `json:"limitType"`
	Amount       string            `json:"amount"`
	Spent        string            `json:"spent"`
	Rest         string            `json:"rest"`
	CurrencyCode string            `json:"currencyCode"`
	ID           string            `json:"id"`
	PlayerID     string            `json:"playerId"`
	Status       bool              `json:"status"`
	StartedAt    int               `json:"startedAt"`
	ExpiresAt    int               `json:"expiresAt"`
	EventType    LimitEventType    `json:"eventType"`
}

type ProjectionSourceMessage struct {
	Type              ProjectionEventType `json:"type"`
	SeqNumber         int                 `json:"seq_number"`
	WalletUUID        string              `json:"wallet_uuid"`
	PlayerUUID        string              `json:"player_uuid"`
	NodeUUID          string              `json:"node_uuid"`
	Payload           string              `json:"payload"`
	Currency          string              `json:"currency"`
	Timestamp         int                 `json:"timestamp"`
	SeqNumberNodeUUID string              `json:"seq_number_node_uuid"`
}

type ProjectionPayloadLimits struct {
	EventType LimitEventType    `json:"event_type"`
	Limits    []ProjectionLimit `json:"limits"`
}

type ProjectionLimit struct {
	ExternalID   string            `json:"external_id"`
	LimitType    LimitType         `json:"limit_type"`
	IntervalType LimitIntervalType `json:"interval_type"`
	Amount       string            `json:"amount"`
	CurrencyCode string            `json:"currency_code"`
	StartedAt    int               `json:"started_at"`
	ExpiresAt    int               `json:"expires_at"`
	Status       bool              `json:"status"`
}

type ProjectionPayloadAdjustment struct {
	Amount        string `json:"amount"`
	Comment       string `json:"comment"`
	Currenc       string `json:"currenc"`
	Direction     int    `json:"direction"`
	OperationType int    `json:"operation_type"`
	Reason        int    `json:"reason"`
	UserName      string `json:"user_name"`
	UserUUID      string `json:"user_uuid"`
	UUID          string `json:"uuid"`
}

type ProjectionPayloadBlockAmount struct {
	UUID      string `json:"uuid"`
	Status    int    `json:"status"`
	Amount    string `json:"amount"`
	Reason    string `json:"reason"`
	Type      int    `json:"type"`
	ExpiredAt int    `json:"expired_at"`
	UserUUID  string `json:"user_uuid"`
	UserName  string `json:"user_name"`
	CreatedAt int    `json:"created_at"`
}

type ProjectionPayloadBlockAmountRevoked struct {
	UUID     string `json:"uuid"`
	NodeUUID string `json:"node_uuid"`
	Amount   string `json:"amount"`
}

type ProjectionPayloadDepositedMoney struct {
	UUID         string `json:"uuid"`
	CurrencyCode string `json:"currency_code"`
	Amount       string `json:"amount"`
	Status       int    `json:"status"`
	NodeUUID     string `json:"node_uuid"`
	BonusID      string `json:"bonus_id"`
}

func (m *ProjectionSourceMessage) UnmarshalPayloadTo(payload interface{}) error {
	return json.Unmarshal([]byte(m.Payload), payload)
}

type ConfirmationContext struct {
	ConfirmationCode string `json:"confirmationCode"`
}

func (m *PlayerMessage) GetConfirmationContext() (*ConfirmationContext, error) {
	if len(m.Context) == 0 {
		return nil, errors.New("empty context")
	}

	var ctx ConfirmationContext
	err := json.Unmarshal(m.Context, &ctx)
	if err != nil {
		return nil, err
	}

	return &ctx, nil
}

type GameMessage struct {
	ID                   string            `json:"id"`
	ExternalUUID         string            `json:"external_uuid"`
	Name                 string            `json:"name"`
	OriginalName         string            `json:"original_name"`
	Alias                string            `json:"alias"`
	Image                string            `json:"image"`
	ProviderName         string            `json:"provider_name"`
	ProviderID           string            `json:"provider_id"`
	ProviderExternalUUID string            `json:"provider_external_uuid"`
	GameTypeLocalized    map[string]string `json:"game_type_localized"`
	Status               string            `json:"status"`
	GameTypeUUID         string            `json:"game_type_uuid"`
	BrandUUID            string            `json:"brand_uuid"`
}

type TransactionMessage struct {
	PlayerID    string `json:"playerId"`
	NodeID      string `json:"nodeId"`
	Transaction struct {
		TransactionID          string               `json:"transactionId"`
		CurrencyCode           string               `json:"currencyCode"`
		Direction              TransactionDirection `json:"direction"`
		PaymentMethod          PaymentMethod        `json:"paymentMethod"`
		Amount                 string               `json:"amount"`
		Status                 TransactionStatus    `json:"status"`
		CreatedAt              int                  `json:"createdAt"`
		UpdatedAt              int                  `json:"updatedAt"`
		PlayerAccountID        string               `json:"playerAccountId"`
		StatusNumber           int                  `json:"statusNumber"`
		DefaultCurrencyCode    string               `json:"defaultCurrencyCode"`
		DefaultAmount          string               `json:"defaultAmount"`
		ProcessingCurrencyCode string               `json:"processingCurrencyCode"`
		ProcessingAmount       string               `json:"processingAmount"`
		PaymentMethodAlias     PaymentMethodAlias   `json:"paymentMethodAlias"`
		MethodID               int                  `json:"methodId"`
	} `json:"transaction"`
	Meta struct {
		FirstDep        bool   `json:"firstDep"`
		FirstDepAmount  string `json:"firstDepAmount"`
		Gateway         string `json:"gateway"`
		InternalID      int    `json:"internalId"`
		CustomAdminName string `json:"customAdminName"`
		Fee             struct {
			Currency            string `json:"currency"`
			BalanceSide         string `json:"balanceSide"`
			AmountInFeeCurrency string `json:"amountInFeeCurrency"`
			TransactionFees     []struct {
				Amount string `json:"amount"`
				Type   string `json:"type"`
			} `json:"transactionFees"`
		} `json:"fee"`
	} `json:"meta"`
}
