package requests

import (
	httpClient "CB_auto/test/transport/http"
	"fmt"
	"net/http"
)

type CapGameListRequest struct {
	QueryParams CapGameListQueryParams
	Headers     CapGameListHeaders
}

type CapGameListQueryParams struct {
	GroupID       string
	ProjectID     string
	SortField     string
	SortDirection int
	Page          int
	PerPage       int
}

type CapGameListHeaders struct {
	Authorization  string
	PlatformLocale string
}

type GetCapGameListResponse struct {
	Total int       `json:"total"`
	Items []CapGame `json:"items"`
}

type CapGame struct {
	ID                      string   `json:"id"`
	ProjectID               string   `json:"projectId"`
	GroupID                 string   `json:"groupId"`
	Name                    string   `json:"name"`
	Alias                   string   `json:"alias"`
	OriginalName            string   `json:"originalName"`
	Image                   string   `json:"image"`
	OriginalImage           string   `json:"originalImage"`
	ProviderID              string   `json:"providerId"`
	GameDevice              int      `json:"gameDevice"`
	HasDemo                 bool     `json:"hasDemo"`
	IsDemoDisabled          bool     `json:"isDemoDisabled"`
	Type                    GameType `json:"type"`
	CreatedAt               int64    `json:"createdAt"`
	UpdatedAt               int64    `json:"updatedAt"`
	CategoryIDs             []string `json:"categoryIds"`
	StatusOnProject         string   `json:"statusOnProject"`
	StatusOnGroup           string   `json:"statusOnGroup"`
	ProviderStatusOnProject string   `json:"providerStatusOnProject"`
	ProviderStatusOnGroup   string   `json:"providerStatusOnGroup"`
	RuleResource            string   `json:"ruleResource"`
}

type GameType struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Localized map[string]string `json:"localized"`
}

func (p CapGameListRequest) GetPath() string {
	return "/_cap/api/v1/games"
}

func (p CapGameListRequest) GetQueryParams() map[string]string {
	return map[string]string{
		"groupId":       p.QueryParams.GroupID,
		"projectId":     p.QueryParams.ProjectID,
		"sortField":     p.QueryParams.SortField,
		"sortDirection": fmt.Sprintf("%d", p.QueryParams.SortDirection),
		"page":          fmt.Sprintf("%d", p.QueryParams.Page),
		"perPage":       fmt.Sprintf("%d", p.QueryParams.PerPage),
	}
}

func (p CapGameListRequest) GetBody() []byte {
	return nil
}

func (p CapGameListRequest) GetQueryHeaders() map[string]string {
	return map[string]string{
		"Authorization":   p.Headers.Authorization,
		"platform-locale": p.Headers.PlatformLocale,
	}
}

func (p CapGameListRequest) GetPathParams() map[string]string {
	return nil
}

func GetCapGameList(client *httpClient.Client, request CapGameListRequest) (*GetCapGameListResponse, error) {
	response, err := httpClient.DoRequest[CapGameListRequest, GetCapGameListResponse](client, http.MethodGet, request)
	if err != nil {
		return nil, fmt.Errorf("GetCapGameList failed: %v", err)
	}

	return response, nil
}
