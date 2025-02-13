package models

type FastRegistrationRequestBody struct {
	Country  string `json:"country"`
	Currency string `json:"currency"`
}

type FastRegistrationResponseBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
