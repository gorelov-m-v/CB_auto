package models

type LimitPeriodType string

const (
	// LimitPeriodType определяет периоды лимитов
	LimitPeriodDaily   LimitPeriodType = "daily"
	LimitPeriodWeekly  LimitPeriodType = "weekly"
	LimitPeriodMonthly LimitPeriodType = "monthly"
)

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
	ExpireType string `json:"expireType"`
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
