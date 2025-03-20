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
	StartedAt int             `json:"startedAt"`
}

type UpcomingChangeData struct {
	ExpiresAt *int   `json:"expiresAt"`
	StartedAt *int   `json:"startedAt"`
	Amount    string `json:"amount"`
}

type UpcomingChange struct {
	ApplyAt int                `json:"applyAt"`
	Data    UpcomingChangeData `json:"data"`
}

type CasinoLossLimit struct {
	ID              string           `json:"id"`
	Type            LimitPeriodType  `json:"type"`
	Currency        string           `json:"currency"`
	Amount          string           `json:"amount"`
	Spent           string           `json:"spent"`
	Rest            string           `json:"rest"`
	StartedAt       int              `json:"startedAt"`
	ExpiresAt       int              `json:"expiresAt"`
	Status          bool             `json:"status"`
	UpcomingChanges []UpcomingChange `json:"upcomingChanges"`
	DeactivatedAt   *int             `json:"deactivatedAt"`
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
	StartedAt       int              `json:"startedAt"`
	UpcomingChanges []UpcomingChange `json:"upcomingChanges"`
	ExpiresAt       int              `json:"expiresAt"`
	DeactivatedAt   *int             `json:"deactivatedAt"`
	Required        bool             `json:"required"`
}

type GetTurnoverLimitsResponseBody []TurnoverLimit

type SingleBetLimit struct {
	ID              string           `json:"id"`
	Currency        string           `json:"currency"`
	Status          bool             `json:"status"`
	Amount          string           `json:"amount"`
	UpcomingChanges []UpcomingChange `json:"upcomingChanges"`
	DeactivatedAt   *int             `json:"deactivatedAt"`
	Required        bool             `json:"required"`
}

type GetSingleBetLimitsResponseBody []SingleBetLimit

type SetRestrictionRequestBody struct {
	ExpireType string `json:"expireType"`
}

type SetRestrictionResponseBody struct {
	ID              string           `json:"id"`
	StartedAt       int              `json:"startedAt"`
	UpcomingChanges []UpcomingChange `json:"upcomingChanges"`
	ExpiresAt       int              `json:"expiresAt"`
	DeactivatedAt   *int             `json:"deactivatedAt"`
}

type SetTurnoverLimitRequestBody struct {
	Amount    string          `json:"amount"`
	Currency  string          `json:"currency"`
	Type      LimitPeriodType `json:"type"`
	StartedAt int             `json:"startedAt"`
}

type UpdateSingleBetLimitRequestBody struct {
	Amount string `json:"amount"`
}

type UpdateRecalculatedLimitRequestBody struct {
	Amount string `json:"amount"`
}
