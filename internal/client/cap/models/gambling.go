package models

type StatusType int
type CategoryType string
type CategoryStatus int

const (

	// Статусы
	StatusEnabled  StatusType = 1
	StatusDisabled StatusType = 2
	StatusDeleted  StatusType = 3

	// Типы категорий
	TypeVertical   CategoryType = "vertical"
	TypeHorizontal CategoryType = "horizontal"
	TypeAllGames   CategoryType = "allGames"

	// Статусы категорий
	CategoryStatusDeleted CategoryStatus = 2
)

// Модели для операций с брендами
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
	Status      StatusType        `json:"status"`
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

type UpdateBrandStatusRequestBody struct {
	Status StatusType `json:"status"`
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

// Модели для операций с категориями
type CreateCapCategoryRequestBody struct {
	Sort      int               `json:"sort"`
	Alias     string            `json:"alias"`
	Names     map[string]string `json:"names"`
	Type      CategoryType      `json:"type"`
	GroupID   string            `json:"groupId"`
	ProjectID string            `json:"projectId"`
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
	Status     StatusType        `json:"status"`
	Sort       int               `json:"sort"`
	IsDefault  bool              `json:"isDefault"`
	Type       CategoryType      `json:"type"`
	PassToCms  bool              `json:"passToCms"`
}

type UpdateCapCategoryRequestBody struct {
	Alias string            `json:"alias"`
	Names map[string]string `json:"names"`
	Sort  int               `json:"sort"`
	Type  CategoryType      `json:"type"`
}

type UpdateCapCategoryResponseBody struct {
	ID    string `json:"id"`
	Alias string `json:"alias"`
}

type UpdateCapCollectionStatusRequestBody struct {
	Status StatusType `json:"status"`
}

type UpdateCapCollectionStatusResponseBody struct {
	Status StatusType `json:"status"`
}

type UpdateCapCategoryStatusRequestBody struct {
	Status StatusType `json:"status"`
}

type UpdateCapCategoryStatusResponseBody struct {
	Status StatusType `json:"status"`
}
