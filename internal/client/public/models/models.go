package models

type LimitPeriodType string
type VerificationType string
type VerificationStatus int
type DocumentType string
type ContactType string

const (
	// LimitPeriodType определяет периоды лимитов
	LimitPeriodDaily   LimitPeriodType = "daily"
	LimitPeriodWeekly  LimitPeriodType = "weekly"
	LimitPeriodMonthly LimitPeriodType = "monthly"

	// Редиректы пеймента
	DepositRedirectURLFailed  = "https://beta-09.b2bdev.pro/en/account/deposit/failed"
	DepositRedirectURLSuccess = "https://beta-09.b2bdev.pro/en/account/deposit/success"
	DepositRedirectURLPending = "https://beta-09.b2bdev.pro/en/account/deposit/pending"

	// Заголовок "Platform-Locale" по умолчанию
	DefaultLocale = "en"

	// VerificationType определяет типы верификаций
	VerificationTypeAddress  VerificationType = "2"
	VerificationTypeIdentity VerificationType = "4"

	// VerificationStatus определяет статусы верификации
	VerificationStatusPending  VerificationStatus = 0
	VerificationStatusApproved VerificationStatus = 1
	VerificationStatusRejected VerificationStatus = 2

	// DocumentType определяет типы документов
	DocumentTypeIdentity DocumentType = "4"
	DocumentTypeAddress  DocumentType = "2"

	// ContactType определяет типы контактов
	ContactTypePhone ContactType = "PHONE"
	ContactTypeEmail ContactType = "EMAIL"
)

type FastRegistrationRequestBody struct {
	Country  string `json:"country"`
	Currency string `json:"currency"`
}

type FastRegistrationResponseBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokenCheckRequestBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokenCheckResponseBody struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

type WalletData struct {
	ID       string `json:"id"`
	Currency string `json:"currency"`
	Balance  string `json:"balance"`
	Default  bool   `json:"default"`
	Main     bool   `json:"main"`
}

type GetWalletsResponseBody struct {
	Wallets []WalletData `json:"wallets"`
}

type CreateWalletRequestBody struct {
	Currency string `json:"currency"`
}

type CreateWalletResponseBody struct{}

type SwitchWalletRequestBody struct {
	Currency string `json:"currency"`
}

type SetSingleBetLimitRequestBody struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type SetCasinoLossLimitRequestBody struct {
	Amount    string          `json:"amount"`
	Currency  string          `json:"currency"`
	Type      LimitPeriodType `json:"type"`
	StartedAt int64           `json:"startedAt"`
}

type UpcomingChangeData struct {
	ExpiresAt *int64 `json:"expiresAt"`
	StartedAt *int64 `json:"startedAt"`
	Amount    string `json:"amount"`
}

type UpcomingChange struct {
	ApplyAt int64              `json:"applyAt"`
	Data    UpcomingChangeData `json:"data"`
}

type CasinoLossLimit struct {
	ID              string           `json:"id"`
	Type            string           `json:"type"`
	Currency        string           `json:"currency"`
	Amount          string           `json:"amount"`
	Spent           string           `json:"spent"`
	Rest            string           `json:"rest"`
	StartedAt       int64            `json:"startedAt"`
	ExpiresAt       int64            `json:"expiresAt"`
	Status          bool             `json:"status"`
	UpcomingChanges []UpcomingChange `json:"upcomingChanges"`
	DeactivatedAt   *int64           `json:"deactivatedAt"`
}

type GetCasinoLossLimitsResponseBody []CasinoLossLimit

type TurnoverLimit struct {
	ID              string           `json:"id"`
	Type            string           `json:"type"`
	Currency        string           `json:"currency"`
	Status          bool             `json:"status"`
	Amount          string           `json:"amount"`
	Spent           string           `json:"spent"`
	Rest            string           `json:"rest"`
	StartedAt       int64            `json:"startedAt"`
	UpcomingChanges []UpcomingChange `json:"upcomingChanges"`
	ExpiresAt       int64            `json:"expiresAt"`
	DeactivatedAt   *int64           `json:"deactivatedAt"`
	Required        bool             `json:"required"`
}

type GetTurnoverLimitsResponseBody []TurnoverLimit

type SingleBetLimit struct {
	ID              string           `json:"id"`
	Currency        string           `json:"currency"`
	Status          bool             `json:"status"`
	Amount          string           `json:"amount"`
	UpcomingChanges []UpcomingChange `json:"upcomingChanges"`
	DeactivatedAt   *int64           `json:"deactivatedAt"`
	Required        bool             `json:"required"`
}

type GetSingleBetLimitsResponseBody []SingleBetLimit

type SetRestrictionRequestBody struct {
	ExpireType string `json:"expireType"` // day, week, month
}

type SetRestrictionResponseBody struct {
	ID              string           `json:"id"`
	StartedAt       int64            `json:"startedAt"`
	UpcomingChanges []UpcomingChange `json:"upcomingChanges"`
	ExpiresAt       int64            `json:"expiresAt"`
	DeactivatedAt   *int64           `json:"deactivatedAt"`
}

type SetTurnoverLimitRequestBody struct {
	Amount    string          `json:"amount"`
	Currency  string          `json:"currency"`
	Type      LimitPeriodType `json:"type"`
	StartedAt int64           `json:"startedAt"`
}

type DepositRedirectURLs struct {
	Failed  string `json:"failed"`
	Success string `json:"success"`
	Pending string `json:"pending"`
}

type DepositRequestBody struct {
	Amount          string              `json:"amount"`
	PaymentMethodID int                 `json:"paymentMethodId"`
	Currency        string              `json:"currency"`
	Country         string              `json:"country"`
	Redirect        DepositRedirectURLs `json:"redirect"`
}

type UpdatePlayerRequestBody struct {
	FirstName        string `json:"firstName"`
	LastName         string `json:"lastName"`
	Gender           int    `json:"gender"`
	City             string `json:"city"`
	Postcode         string `json:"postcode"`
	PermanentAddress string `json:"permanentAddress"`
	PersonalID       string `json:"personalId"`
	Profession       string `json:"profession"`
	IBAN             string `json:"iban"`
	Birthday         string `json:"birthday"`
	Country          string `json:"country"`
}

type UpdatePlayerResponseBody struct {
	ID                       string  `json:"id"`
	AccountID                string  `json:"accountId"`
	Email                    *string `json:"email"`
	Phone                    *string `json:"phone"`
	NodeID                   string  `json:"nodeId"`
	FirstName                string  `json:"firstName"`
	MiddleName               *string `json:"middleName"`
	LastName                 string  `json:"lastName"`
	Gender                   int     `json:"gender"`
	PermanentAddress         string  `json:"permanentAddress"`
	City                     string  `json:"city"`
	Region                   *string `json:"region"`
	Country                  string  `json:"country"`
	Postcode                 string  `json:"postcode"`
	Birthday                 string  `json:"birthday"`
	PersonalID               string  `json:"personalId"`
	RegSource                string  `json:"regSource"`
	Locale                   string  `json:"locale"`
	IBAN                     string  `json:"iban"`
	Profession               string  `json:"profession"`
	Status                   int     `json:"status"`
	IsPoliticallyInvolved    *bool   `json:"isPoliticallyInvolved"`
	PlaceOfWork              *string `json:"placeOfWork"`
	JobAlias                 *string `json:"jobAlias"`
	JobInput                 *string `json:"jobInput"`
	AvgMonthlySalaryEURAlias *string `json:"avgMonthlySalaryEURAlias"`
	AvgMonthlySalaryEURInput *string `json:"avgMonthlySalaryEURInput"`
	ActivitySectorAlias      *string `json:"activitySectorAlias"`
	ActivitySectorInput      *string `json:"activitySectorInput"`
}

type VerifyIdentityRequestBody struct {
	Number     string           `json:"number"`
	Type       VerificationType `json:"type"`
	IssuedDate string           `json:"issuedDate,omitempty"`
	ExpiryDate string           `json:"expiryDate,omitempty"`
}

type VerifyIdentityResponseBody struct {
}

type VerificationStatusResponseItem struct {
	Status         VerificationStatus `json:"status"`
	DocumentID     string             `json:"documentId"`
	Reason         any                `json:"reason,omitempty"`
	Type           VerificationType   `json:"type"`
	DocumentType   DocumentType       `json:"documentType"`
	DocumentNumber string             `json:"documentNumber,omitempty"`
	ExpireDate     int64              `json:"expireDate,omitempty"`
}

type RequestVerificationRequestBody struct {
	Contact string      `json:"contact"`
	Type    ContactType `json:"type"`
}

type RequestVerificationResponseBody struct {
}
