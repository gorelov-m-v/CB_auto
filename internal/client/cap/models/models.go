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
