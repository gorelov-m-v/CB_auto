package redis

type WalletsMap map[string]WalletData

type WalletData struct {
	WalletUUID string `json:"wallet_uuid"`
	Currency   string `json:"currency"`
	Type       int    `json:"type"`
	Status     int    `json:"status"`
}

type WalletFullData struct {
	WalletUUID                 string         `json:"WalletUUID"`
	PlayerUUID                 string         `json:"PlayerUUID"`
	PlayerBonusUUID            string         `json:"PlayerBonusUUID"`
	NodeUUID                   string         `json:"NodeUUID"`
	Type                       int            `json:"Type"`
	Status                     int            `json:"Status"`
	Valid                      bool           `json:"Valid"`
	IsGamblingActive           bool           `json:"IsGamblingActive"`
	IsBettingActive            bool           `json:"IsBettingActive"`
	Currency                   string         `json:"Currency"`
	Balance                    string         `json:"Balance"`
	AvailableWithdrawalBalance string         `json:"AvailableWithdrawalBalance"`
	BalanceBefore              string         `json:"BalanceBefore"`
	CreatedAt                  int64          `json:"CreatedAt"`
	UpdatedAt                  int64          `json:"UpdatedAt"`
	BlockDate                  int64          `json:"BlockDate"`
	SumSubBlockDate            int64          `json:"SumSubBlockDate"`
	KYCVerificationUpdateTo    int64          `json:"KYCVerificationUpdateTo"`
	LastSeqNumber              int            `json:"LastSeqNumber"`
	Default                    bool           `json:"Default"`
	Main                       bool           `json:"Main"`
	IsBlocked                  bool           `json:"IsBlocked"`
	IsKYCUnverified            bool           `json:"IsKYCUnverified"`
	IsSumSubVerified           bool           `json:"IsSumSubVerified"`
	BonusInfo                  BonusInfo      `json:"BonusInfo"`
	BonusTransferTransactions  map[string]any `json:"BonusTransferTransactions"`
	Limits                     []LimitData    `json:"Limits"`
	IFrameRecords              []any          `json:"IFrameRecords"`
	Gambling                   map[string]any `json:"Gambling"`
	Deposits                   []any          `json:"Deposits"`
	BlockedAmounts             []any          `json:"BlockedAmounts"`
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
	ExternalID   string `json:"ExternalID"`
	LimitType    string `json:"LimitType"`
	IntervalType string `json:"IntervalType"`
	Amount       string `json:"Amount"`
	Spent        string `json:"Spent"`
	Rest         string `json:"Rest"`
	CurrencyCode string `json:"CurrencyCode"`
	StartedAt    int64  `json:"StartedAt"`
	ExpiresAt    int64  `json:"ExpiresAt"`
	Status       bool   `json:"Status"`
}
