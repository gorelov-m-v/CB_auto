package redis

type LimitPeriodType string
type LimitType string
type WalletsMap map[string]WalletData
type EventType string
type TransactionStatus int

const (
	// LimitPeriodType определяет типы периодов для лимитов
	LimitPeriodDaily   LimitPeriodType = "daily"
	LimitPeriodWeekly  LimitPeriodType = "weekly"
	LimitPeriodMonthly LimitPeriodType = "monthly"

	// LimitType определяет типы лимитов
	LimitTypeSingleBet     LimitType = "single-bet"
	LimitTypeCasinoLoss    LimitType = "casino-loss"
	LimitTypeTurnoverFunds LimitType = "turnover-of-funds"

	TransactionStatusSuccess TransactionStatus = 4
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
	CreatedAt                  int             `json:"CreatedAt"`
	UpdatedAt                  int             `json:"UpdatedAt"`
	BlockDate                  int             `json:"BlockDate"`
	SumSubBlockDate            int             `json:"SumSubBlockDate"`
	KYCVerificationUpdateTo    int             `json:"KYCVerificationUpdateTo"`
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
	Deposits                   []DepositData   `json:"Deposits"`
	BlockedAmounts             []BlockedAmount `json:"BlockedAmounts"`
}

type DepositData struct {
	UUID           string            `json:"UUID"`
	NodeUUID       string            `json:"NodeUUID"`
	BonusID        string            `json:"BonusID"`
	CurrencyCode   string            `json:"CurrencyCode"`
	Status         TransactionStatus `json:"Status"`
	Amount         string            `json:"Amount"`
	WageringAmount string            `json:"WageringAmount"`
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
	StartedAt    int             `json:"StartedAt"`
	ExpiresAt    int             `json:"ExpiresAt"`
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
	CreatedAt                       int    `json:"CreatedAt"`
	ExpiredAt                       int    `json:"ExpiredAt"`
}
