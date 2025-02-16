package models

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
