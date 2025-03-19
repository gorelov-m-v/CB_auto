package models

type StatusTypeDemoGame string
type StatusTypeGame string

const (

	// Статусы
	StatusDemoEnabled  StatusTypeDemoGame = "true"
	StatusDemoDisabled StatusTypeDemoGame = "false"
)

type Localized struct {
	Ru string `json:"ru"`
	En string `json:"en"`
}

type Type struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Localized Localized `json:"localized"`
}

type GetCapGamesResponseBody struct {
	ID                      string   `json:"id"`
	ProjectID               string   `json:"projectId"`
	GroupID                 string   `json:"groupId"`
	Name                    string   `json:"name"`
	Alias                   string   `json:"alias"`
	OriginalName            string   `json:"originalName"`
	Image                   string   `json:"image"`
	OriginalImage           string   `json:"originalImage"`
	ProviderId              string   `json:"providerId"`
	GameDevice              int      `json:"gameDevice"`
	HasDemo                 bool     `json:"hasDemo"`
	IsDemoDisabled          bool     `json:"isDemoDisabled"`
	Type                    Type     `json:"type"`
	CreatedAt               int64    `json:"createdAt"`
	UpdatedAt               int64    `json:"updatedAt"`
	CategoryIds             []string `json:"categoryIds"`
	StatusOnProject         string   `json:"statusOnProject"`
	StatusOnGroup           string   `json:"statusOnGroup"`
	ProviderStatusOnProject string   `json:"providerStatusOnProject"`
	ProviderStatusOnGroup   string   `json:"providerStatusOnGroup"`
	RuleResource            string   `json:"ruleResource"`
}

type UpdateGamesStatusRequestBody struct {
	Status StatusTypeGame `json:"status"`
}

type UpdateGamesDemoStatusRequestBody struct {
	Status StatusTypeDemoGame `json:"isDemoDisabled"`
}

type UpdateCapGamesRequestBody struct {
	Alias        string `json:"alias"`
	Image        string `json:"image,omitempty"`
	Name         string `json:"name,omitempty"`
	RuleResource string `json:"ruleResource,omitempty"`
}

type UpdateCapGamesResponseBody struct {
	ID string `json:"id"`
}
