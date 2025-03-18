package models

type LimitPeriodType string
type LimitType string
type DirectionType string
type ReasonType string
type OperationType string

const (
	// Периоды лимитов
	LimitPeriodDaily   LimitPeriodType = "Daily"
	LimitPeriodWeekly  LimitPeriodType = "Weekly"
	LimitPeriodMonthly LimitPeriodType = "Monthly"

	// Типы лимитов
	LimitTypeSingleBet  LimitType = "Single bet"
	LimitTypeTurnover   LimitType = "Turnover of funds"
	LimitTypeCasinoLoss LimitType = "Casino loss"

	// Направления корректировок
	DirectionIncrease DirectionType = "INCREASE"
	DirectionDecrease DirectionType = "DECREASE"

	// Причины корректировок
	ReasonMalfunction        ReasonType = "MALFUNCTION"
	ReasonOperationalMistake ReasonType = "OPERATIONAL_MISTAKE"
	ReasonBalanceCorrection  ReasonType = "BALANCE_CORRECTION"

	// Типы корректировок
	OperationTypeCorrection         OperationType = "CORRECTION"
	OperationTypeDeposit            OperationType = "DEPOSIT"
	OperationTypeWithdrawal         OperationType = "WITHDRAWAL"
	OperationTypeGift               OperationType = "GIFT"
	OperationTypeReferralCommission OperationType = "REFERRAL_COMMISSION"
	OperationTypeCashback           OperationType = "CASHBACK"
	OperationTypeTournamentPrize    OperationType = "TOURNAMENT_PRIZE"
	OperationTypeJackpot            OperationType = "JACKPOT"
)

type BlockersRequestBody struct {
	GamblingEnabled bool `json:"gamblingEnabled"`
	BettingEnabled  bool `json:"bettingEnabled"`
}

type GetBlockersResponseBody struct {
	GamblingEnabled bool `json:"gamblingEnabled"`
	BettingEnabled  bool `json:"bettingEnabled"`
}

type GetPlayerLimitsResponseBody struct {
	Data  []PlayerLimit `json:"data"`
	Total int           `json:"total"`
}

type PlayerLimit struct {
	Type          LimitType       `json:"type"`
	Status        bool            `json:"status"`
	Period        LimitPeriodType `json:"period"`
	Currency      string          `json:"currency"`
	Amount        string          `json:"amount"`
	Rest          string          `json:"rest,omitempty"`
	CreatedAt     int64           `json:"createdAt"`
	DeactivatedAt int64           `json:"deactivatedAt,omitempty"`
	StartedAt     int             `json:"startedAt"`
	ExpiresAt     int             `json:"expiresAt,omitempty"`
}

type CreateBalanceAdjustmentRequestBody struct {
	Currency      string        `json:"currency"`
	Amount        float64       `json:"amount"`
	Reason        ReasonType    `json:"reason"`
	OperationType OperationType `json:"operationType"`
	Direction     DirectionType `json:"direction"`
	Comment       string        `json:"comment"`
}

type CreateBlockAmountRequestBody struct {
	Reason   string `json:"reason"`
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type CreateBlockAmountResponseBody struct {
	TransactionID string `json:"transactionId"`
	Currency      string `json:"currency"`
	Amount        string `json:"amount"`
	Reason        string `json:"reason"`
	UserID        string `json:"userId"`
	UserName      string `json:"userName"`
	CreatedAt     int64  `json:"createdAt"`
}

type BlockAmountListItem struct {
	TransactionID string `json:"transactionId"`
	Currency      string `json:"currency"`
	Amount        string `json:"amount"`
	Reason        string `json:"reason"`
	UserID        string `json:"userId"`
	UserName      string `json:"userName"`
	CreatedAt     int64  `json:"createdAt"`
	WalletID      string `json:"walletId"`
	PlayerID      string `json:"playerId"`
}

type BlockAmountListResponseBody struct {
	Items []BlockAmountListItem `json:"items"`
}

type GetWalletListWallet struct {
	Currency            string `json:"currency"`
	Balance             string `json:"balance"`
	PaymentBlockAmount  string `json:"paymentBlockAmount"`
	BlockAmount         string `json:"blockAmount"`
	ActualBalance       string `json:"actualBalance"`
	AvailableWithdrawal string `json:"availableWithdrawal"`
}

type GetWalletListResponseBody struct {
	Wallets []GetWalletListWallet `json:"wallets"`
}
