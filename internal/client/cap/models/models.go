package models

const (
	StatusEnabled  = 1
	StatusDisabled = 2
	StatusDeleted  = 3
)

type AdminCheckRequestBody struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

type AdminCheckResponseBody struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

type CreateCapBrandRequestBody struct {
	Sort        int               `json:"sort"`
	Alias       string            `json:"alias"`
	Names       map[string]string `json:"names"`
	Description string            `json:"description"`
}

type CreateCapBrandResponseBody struct {
	ID string `json:"id"`
}

type CreateCapBrandRequestHeaders struct {
	Authorization  string
	PlatformNodeID string
}

type GetCapBrandRequestHeaders struct {
	Authorization  string
	PlatformNodeID string
}

type GetCapBrandResponseBody struct {
	ID          string            `json:"id"`
	Names       map[string]string `json:"names"`
	Alias       string            `json:"alias"`
	Description string            `json:"description"`
	GameIDs     []string          `json:"gameIds"`
	Status      int               `json:"status"`
	Sort        int               `json:"sort"`
	NodeID      string            `json:"nodeId"`
	CreatedAt   int64             `json:"createdAt"`
	UpdatedAt   int64             `json:"updatedAt"`
	CreatedBy   string            `json:"createdBy"`
	UpdatedBy   string            `json:"updatedBy"`
	Icon        string            `json:"icon"`
	Logo        string            `json:"logo"`
	ColorLogo   string            `json:"colorLogo"`
}

type BlockersRequestBody struct {
	GamblingEnabled bool `json:"gamblingEnabled"`
	BettingEnabled  bool `json:"bettingEnabled"`
}

type GetBlockersResponseBody struct {
	GamblingEnabled bool `json:"gamblingEnabled"`
	BettingEnabled  bool `json:"bettingEnabled"`
}

type UpdateBrandStatusRequestBody struct {
	Status int `json:"status"`
}

type UpdateCapBrandRequestBody struct {
	Sort        int               `json:"sort"`
	Alias       string            `json:"alias"`
	Names       map[string]string `json:"names"`
	Description string            `json:"description"`
}

type UpdateCapBrandResponseBody struct {
	ID string `json:"id"`
}

type BrandEvent struct {
	Message struct {
		EventType string `json:"eventType"`
	} `json:"message"`
	Brand struct {
		UUID      string `json:"uuid"`
		DeletedAt int64  `json:"deleted_at,omitempty"`
	} `json:"brand"`
}

const (
	LimitTypeSingleBet  = "Single bet"
	LimitTypeTurnover   = "Turnover of funds"
	LimitTypeCasinoLoss = "Casino loss"
)

type GetPlayerLimitsResponseBody struct {
	Data  []PlayerLimit `json:"data"`
	Total int           `json:"total"`
}

type PlayerLimit struct {
	Type          string `json:"type"`
	Status        bool   `json:"status"`
	Period        string `json:"period"`
	Currency      string `json:"currency"`
	Amount        string `json:"amount"`
	Rest          string `json:"rest,omitempty"`
	CreatedAt     int64  `json:"createdAt"`
	DeactivatedAt int64  `json:"deactivatedAt,omitempty"`
	StartedAt     int64  `json:"startedAt"`
	ExpiresAt     int64  `json:"expiresAt,omitempty"`
}

type CreateCapCategoryRequestBody struct {
	Sort    int               `json:"sort"`
	Alias   string            `json:"alias"`
	Names   map[string]string `json:"names"`
	Type    string            `json:"type"`
	GroupID string            `json:"groupId"`
}

type CreateCapCategoryResponseBody struct {
	ID string `json:"id"`
}

type CreateCapCategoryRequestHeaders struct {
	Authorization  string
	PlatformNodeID string
}

type GetCapCategoryRequestHeaders struct {
	Authorization  string
	PlatformNodeID string
}

type GetCapCategoryResponseBody struct {
	ID         string            `json:"id"`
	Names      map[string]string `json:"names"`
	Alias      string            `json:"alias"`
	ProjectId  string            `json:"projectId"`
	GroupID    string            `json:"groupId"`
	GamesCount int               `json:"gamesCount"`
	Status     int               `json:"status"`
	Sort       int               `json:"sort"`
	IsDefault  bool              `json:"isDefault"`
	Type       string            `json:"type"`
	PassToCms  bool              `json:"passToCms"`
}

type CreateLabelRequestBody struct {
	Color       string       `json:"color"`
	Titles      []LabelTitle `json:"titles"`
	Description string       `json:"description"`
}

type LabelTitle struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

type CreateLabelResponseBody struct {
	UUID string `json:"uuid"`
}

type GetLabelResponseBody struct {
	UUID           string       `json:"uuid"`
	Color          string       `json:"color"`
	Node           string       `json:"node"`
	UserID         string       `json:"userId"`
	AuthorCreation string       `json:"authorCreation"`
	AuthorEditing  string       `json:"authorEditing"`
	CreatedAt      string       `json:"createdAt"`
	UpdatedAt      string       `json:"updatedAt"`
	Titles         []LabelTitle `json:"titles"`
	Description    string       `json:"description"`
}
