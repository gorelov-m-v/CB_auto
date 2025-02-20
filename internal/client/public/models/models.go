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

type SetSingleBetLimitRequestBody struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type SetCasinoLossLimitRequestBody struct {
	Currency  string `json:"currency"`
	Type      string `json:"type"` // daily, weekly, monthly
	Amount    string `json:"amount"`
	StartedAt int64  `json:"startedAt"`
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
	Amount    string `json:"amount"`
	Currency  string `json:"currency"`
	Type      string `json:"type"` // daily, weekly, monthly
	StartedAt int64  `json:"startedAt"`
}
