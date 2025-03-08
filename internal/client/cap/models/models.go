package models

type StatusType int
type LimitPeriodType string
type LimitType string
type DirectionType string
type ReasonType string
type OperationType string
type VerificationStatus int
type CategoryType string
type CategoryStatus int

const (

	// Type определяет типы категорий
	TypeVertical   CategoryType = "vertical"
	TypeHorizontal CategoryType = "horizontal"
	TypeAllGames   CategoryType = "allGames"

	// CategoryStatus определяет статусы категорий
	CategoryStatusDeleted CategoryStatus = 2

	// DefaultLocale определяет язык по умолчанию
	DefaultLocale = "en"

	// StatusType определяет типы статусов
	StatusEnabled  StatusType = 1
	StatusDisabled StatusType = 2
	StatusDeleted  StatusType = 3

	// LimitPeriodType определяет периоды лимитов
	LimitPeriodDaily   LimitPeriodType = "Daily"
	LimitPeriodWeekly  LimitPeriodType = "Weekly"
	LimitPeriodMonthly LimitPeriodType = "Monthly"

	// LimitType определяет типы лимитов
	LimitTypeSingleBet  LimitType = "Single bet"
	LimitTypeTurnover   LimitType = "Turnover of funds"
	LimitTypeCasinoLoss LimitType = "Casino loss"

	// DirectionType определяет направление корректировок баланса
	DirectionIncrease DirectionType = "INCREASE"
	DirectionDecrease DirectionType = "DECREASE"

	// ReasonType определяет причины корректировок баланса
	ReasonMalfunction        ReasonType = "MALFUNCTION"
	ReasonOperationalMistake ReasonType = "OPERATIONAL_MISTAKE"
	ReasonBalanceCorrection  ReasonType = "BALANCE_CORRECTION"

	// OperationType определяет типы корректировок баланса
	OperationTypeCorrection         OperationType = "CORRECTION"
	OperationTypeDeposit            OperationType = "DEPOSIT"
	OperationTypeWithdrawal         OperationType = "WITHDRAWAL"
	OperationTypeGift               OperationType = "GIFT"
	OperationTypeReferralCommission OperationType = "REFERRAL_COMMISSION"
	OperationTypeCashback           OperationType = "CASHBACK"
	OperationTypeTournamentPrize    OperationType = "TOURNAMENT_PRIZE"
	OperationTypeJackpot            OperationType = "JACKPOT"

	//VerificationStatus определяет статусы верификации
	VerificationStatusApproved VerificationStatus = 2
)

type AdminCheckRequestBody struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

type AdminCheckResponseBody struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

type CreateCapBrandRequestBody struct {
	Sort        int               `json:"sort"`
	Alias       string            `json:"alias"`
	Names       map[string]string `json:"names"`
	Description string            `json:"description"`
}

type CreateCapBrandResponseBody struct {
	ID string `json:"id"`
}

type CreateCapBrandRequestHeaders struct {
	Authorization  string
	PlatformNodeID string
}

type GetCapBrandRequestHeaders struct {
	Authorization  string
	PlatformNodeID string
}

type GetCapBrandResponseBody struct {
	ID          string            `json:"id"`
	Names       map[string]string `json:"names"`
	Alias       string            `json:"alias"`
	Description string            `json:"description"`
	GameIDs     []string          `json:"gameIds"`
	Status      StatusType        `json:"status"`
	Sort        int               `json:"sort"`
	NodeID      string            `json:"nodeId"`
	CreatedAt   int64             `json:"createdAt"`
	UpdatedAt   int64             `json:"updatedAt"`
	CreatedBy   string            `json:"createdBy"`
	UpdatedBy   string            `json:"updatedBy"`
	Icon        string            `json:"icon"`
	Logo        string            `json:"logo"`
	ColorLogo   string            `json:"colorLogo"`
}

type BlockersRequestBody struct {
	GamblingEnabled bool `json:"gamblingEnabled"`
	BettingEnabled  bool `json:"bettingEnabled"`
}

type GetBlockersResponseBody struct {
	GamblingEnabled bool `json:"gamblingEnabled"`
	BettingEnabled  bool `json:"bettingEnabled"`
}

type UpdateBrandStatusRequestBody struct {
	Status StatusType `json:"status"`
}

type UpdateCapBrandRequestBody struct {
	Sort        int               `json:"sort"`
	Alias       string            `json:"alias"`
	Names       map[string]string `json:"names"`
	Description string            `json:"description"`
}

type UpdateCapBrandResponseBody struct {
	ID string `json:"id"`
}

type BrandEvent struct {
	Message struct {
		EventType string `json:"eventType"`
	} `json:"message"`
	Brand struct {
		UUID      string `json:"uuid"`
		DeletedAt int64  `json:"deleted_at,omitempty"`
	} `json:"brand"`
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
	StartedAt     int64           `json:"startedAt"`
	ExpiresAt     int64           `json:"expiresAt,omitempty"`
}

type CreateCapCategoryRequestBody struct {
	Sort      int               `json:"sort"`
	Alias     string            `json:"alias"`
	Names     map[string]string `json:"names"`
	Type      CategoryType      `json:"type"`
	GroupID   string            `json:"groupId"`
	ProjectID string            `json:"projectId"`
}

type CreateCapCategoryResponseBody struct {
	ID string `json:"id"`
}

type CreateCapCategoryRequestHeaders struct {
	Authorization  string
	PlatformNodeID string
}

type GetCapCategoryRequestHeaders struct {
	Authorization  string
	PlatformNodeID string
}

type GetCapCategoryResponseBody struct {
	ID         string            `json:"id"`
	Names      map[string]string `json:"names"`
	Alias      string            `json:"alias"`
	ProjectId  string            `json:"projectId"`
	GroupID    string            `json:"groupId"`
	GamesCount int               `json:"gamesCount"`
	Status     StatusType        `json:"status"`
	Sort       int               `json:"sort"`
	IsDefault  bool              `json:"isDefault"`
	Type       CategoryType      `json:"type"`
	PassToCms  bool              `json:"passToCms"`
}

type CreateLabelRequestBody struct {
	Color       string       `json:"color"`
	Titles      []LabelTitle `json:"titles"`
	Description string       `json:"description"`
}

type LabelTitle struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

type CreateLabelResponseBody struct {
	UUID string `json:"uuid"`
}

type GetLabelResponseBody struct {
	UUID           string       `json:"uuid"`
	Color          string       `json:"color"`
	Node           string       `json:"node"`
	UserID         string       `json:"userId"`
	AuthorCreation string       `json:"authorCreation"`
	AuthorEditing  string       `json:"authorEditing"`
	CreatedAt      string       `json:"createdAt"`
	UpdatedAt      string       `json:"updatedAt"`
	Titles         []LabelTitle `json:"titles"`
	Description    string       `json:"description"`
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

type UpdateCapCategoryRequestBody struct {
	Alias string            `json:"alias"`
	Names map[string]string `json:"names"`
	Sort  int               `json:"sort"`
	Type  CategoryType      `json:"type"`
}

type UpdateCapCategoryResponseBody struct {
	ID    string `json:"id"`
	Alias string `json:"alias"`
}

type UpdateVerificationStatusRequestBody struct {
	Note   string             `json:"note"`
	Reason string             `json:"reason"`
	Status VerificationStatus `json:"status"`
}
