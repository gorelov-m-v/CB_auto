package models

type VerificationStatus int

const (
	// Статусы верификации
	VerificationStatusApproved VerificationStatus = 2

	// Заголовок "Platform-Locale" по умолчанию
	DefaultLocale = "en"
)

type UpdateVerificationStatusRequestBody struct {
	Note   string             `json:"note"`
	Reason string             `json:"reason"`
	Status VerificationStatus `json:"status"`
}
