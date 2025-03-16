package redis

type LimitPeriodType string
type LimitType string
type WalletsMap map[string]WalletData
type EventType string

const (
	// LimitPeriodType определяет типы периодов для лимитов
	LimitPeriodDaily   LimitPeriodType = "daily"
	LimitPeriodWeekly  LimitPeriodType = "weekly"
	LimitPeriodMonthly LimitPeriodType = "monthly"

	// LimitType определяет типы лимитов
	LimitTypeSingleBet     LimitType = "single-bet"
	LimitTypeCasinoLoss    LimitType = "casino-loss"
	LimitTypeTurnoverFunds LimitType = "turnover-of-funds"

	// Event Types
	WalletCreated      EventType = "wallet_created"
	WalletDisabled     EventType = "wallet_disabled"
	BlockersSetted     EventType = "setting_prevent_gamble_setted"
	BalanceAdjusted    EventType = "balance_adjusted"
	BlockAmountStarted EventType = "block_amount_started"
	BlockAmountRevoked EventType = "block_amount_revoked"
	DepositedMoney     EventType = "deposited_money"
)

type WalletData struct {
	WalletUUID string `json:"wallet_uuid"`
	Currency   string `json:"currency"`
	Type       int    `json:"type"`
	Status     int    `json:"status"`
}

type WalletFullData struct {
	WalletUUID                 string          `json:"WalletUUID"`
	PlayerUUID                 string          `json:"PlayerUUID"`
	PlayerBonusUUID            string          `json:"PlayerBonusUUID"`
	NodeUUID                   string          `json:"NodeUUID"`
	Type                       int             `json:"Type"`
	Status                     int             `json:"Status"`
	Valid                      bool            `json:"Valid"`
	IsGamblingActive           bool            `json:"IsGamblingActive"`
	IsBettingActive            bool            `json:"IsBettingActive"`
	Currency                   string          `json:"Currency"`
	Balance                    string          `json:"Balance"`
	AvailableWithdrawalBalance string          `json:"AvailableWithdrawalBalance"`
	BalanceBefore              string          `json:"BalanceBefore"`
	CreatedAt                  int64           `json:"CreatedAt"`
	UpdatedAt                  int64           `json:"UpdatedAt"`
	BlockDate                  int64           `json:"BlockDate"`
	SumSubBlockDate            int64           `json:"SumSubBlockDate"`
	KYCVerificationUpdateTo    int64           `json:"KYCVerificationUpdateTo"`
	LastSeqNumber              int             `json:"LastSeqNumber"`
	Default                    bool            `json:"Default"`
	Main                       bool            `json:"Main"`
	IsBlocked                  bool            `json:"IsBlocked"`
	IsKYCUnverified            bool            `json:"IsKYCUnverified"`
	IsSumSubVerified           bool            `json:"IsSumSubVerified"`
	BonusInfo                  BonusInfo       `json:"BonusInfo"`
	BonusTransferTransactions  map[string]any  `json:"BonusTransferTransactions"`
	Limits                     []LimitData     `json:"Limits"`
	IFrameRecords              []any           `json:"IFrameRecords"`
	Gambling                   map[string]any  `json:"Gambling"`
	Deposits                   []any           `json:"Deposits"`
	BlockedAmounts             []BlockedAmount `json:"BlockedAmounts"`
}

type BonusInfo struct {
	BonusUUID       string  `json:"BonusUUID"`
	BonusCategory   string  `json:"BonusCategory"`
	PlayerBonusUUID string  `json:"PlayerBonusUUID"`
	NodeUUID        string  `json:"NodeUUID"`
	Wager           string  `json:"Wager"`
	Threshold       *string `json:"Threshold"`
	TransferType    int     `json:"TransferType"`
	TransferValue   string  `json:"TransferValue"`
	RealPercent     int     `json:"RealPercent"`
	BonusPercent    int     `json:"BonusPercent"`
	BetMin          *string `json:"BetMin"`
	BetMax          *string `json:"BetMax"`
}

type LimitData struct {
	ExternalID   string          `json:"ExternalID"`
	LimitType    LimitType       `json:"LimitType"`
	IntervalType LimitPeriodType `json:"IntervalType"`
	Amount       string          `json:"Amount"`
	Spent        string          `json:"Spent"`
	Rest         string          `json:"Rest"`
	CurrencyCode string          `json:"CurrencyCode"`
	StartedAt    int64           `json:"StartedAt"`
	ExpiresAt    int64           `json:"ExpiresAt"`
	Status       bool            `json:"Status"`
}

type BlockedAmount struct {
	UUID                            string `json:"UUID"`
	UserUUID                        string `json:"UserUUID"`
	Type                            int    `json:"Type"`
	Status                          int    `json:"Status"`
	Amount                          string `json:"Amount"`
	DeltaAvailableWithdrawalBalance string `json:"DeltaAvailableWithdrawalBalance"`
	Reason                          string `json:"Reason"`
	UserName                        string `json:"UserName"`
	CreatedAt                       int64  `json:"CreatedAt"`
	ExpiredAt                       int64  `json:"ExpiredAt"`
}

type DepositedMoneyPayload struct {
	UUID         string `json:"uuid"`
	CurrencyCode string `json:"currency_code"`
	Amount       string `json:"amount"`
	Status       int    `json:"status"`
	NodeUUID     string `json:"node_uuid"`
	BonusID      string `json:"bonus_id"`
}
