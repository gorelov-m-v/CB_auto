package redis

type WalletData struct {
	WalletUUID string `json:"wallet_uuid"`
	Currency   string `json:"currency"`
	Type       int    `json:"type"`
	Status     int    `json:"status"`
}

type WalletsMap map[string]WalletData
