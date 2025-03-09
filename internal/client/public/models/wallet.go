package models

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
