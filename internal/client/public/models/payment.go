package models

const (
	DepositRedirectURLFailed  = "https://beta-09.b2bdev.pro/en/account/deposit/failed"
	DepositRedirectURLSuccess = "https://beta-09.b2bdev.pro/en/account/deposit/success"
	DepositRedirectURLPending = "https://beta-09.b2bdev.pro/en/account/deposit/pending"
	DefaultLocale             = "en"
)

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
