package models

type VerificationType string
type VerificationStatus int
type DocumentType string
type ContactType string

const (
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

// Модели для методов работы с игроками

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

type ConfirmContactRequestBody struct {
	Contact string      `json:"contact"`
	Type    ContactType `json:"type"`
	Code    string      `json:"code"`
}
