package brand

import (
	"CB_auto/test/transport/database"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/Knetic/go-namedParameterQuery"
	"github.com/google/uuid"
)

type CleanBrand struct {
	UUID           string          `json:"UUID"`
	LocalizedNames json.RawMessage `json:"LocalizedNames"`
	Alias          string          `json:"Alias"`
	Description    string          `json:"Description,omitempty"`
	NodeUUID       string          `json:"NodeUUID"`
	Status         int16           `json:"Status"`
	Sort           int             `json:"Sort"`
	CreatedAt      int64           `json:"CreatedAt"`
	CreatedBy      string          `json:"CreatedBy"`
	UpdatedAt      *int64          `json:"UpdatedAt,omitempty"`
	UpdatedBy      string          `json:"UpdatedBy,omitempty"`
	DeletedAt      *int64          `json:"DeletedAt,omitempty"`
	DeletedBy      string          `json:"DeletedBy,omitempty"`
	AliasForIndex  string          `json:"AliasForIndex,omitempty"`
	Icon           string          `json:"Icon,omitempty"`
	Logo           string          `json:"Logo,omitempty"`
}

type Brand struct {
	UUID           string          `db:"uuid"`
	LocalizedNames json.RawMessage `db:"localized_names"`
	Alias          string          `db:"alias"`
	Description    sql.NullString  `db:"description"`
	NodeUUID       string          `db:"node_uuid"`
	Status         int16           `db:"status"`
	Sort           int             `db:"sort"`
	CreatedAt      int64           `db:"created_at"`
	CreatedBy      string          `db:"created_by"`
	UpdatedAt      sql.NullInt64   `db:"updated_at"`
	UpdatedBy      sql.NullString  `db:"updated_by"`
	DeletedAt      sql.NullInt64   `db:"deleted_at"`
	DeletedBy      sql.NullString  `db:"deleted_by"`
	AliasForIndex  sql.NullString  `db:"alias_for_index"`
	Icon           sql.NullString  `db:"icon"`
	Logo           sql.NullString  `db:"logo"`
}

type Repository struct {
	connector *database.Connector
}

func NewRepository(connector *database.Connector) *Repository {
	return &Repository{connector: connector}
}

// GetBrand извлекает бренд и возвращает CleanBrand
func (r *Repository) GetBrand(ctx context.Context, brandUUID uuid.UUID) (*CleanBrand, error) {
	q := `SELECT uuid, localized_names, alias, description, node_uuid, status, sort, created_at, created_by, updated_at, updated_by, icon, logo 
		FROM brand WHERE uuid = :uuid`
	params := map[string]any{"uuid": brandUUID}

	queryNamed := namedParameterQuery.NewNamedParameterQuery(q)
	queryNamed.SetValuesFromMap(params)

	var result Brand
	if err := r.connector.QueryRowContext(ctx, queryNamed.GetParsedQuery(), queryNamed.GetParsedParameters()...).Scan(
		&result.UUID,
		&result.LocalizedNames,
		&result.Alias,
		&result.Description,
		&result.NodeUUID,
		&result.Status,
		&result.Sort,
		&result.CreatedAt,
		&result.CreatedBy,
		&result.UpdatedAt,
		&result.UpdatedBy,
		&result.Icon,
		&result.Logo,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Возвращаем nil, если запись не найдена
		}
		return nil, err
	}

	cleanBrand := &CleanBrand{
		UUID:           result.UUID,
		LocalizedNames: result.LocalizedNames,
		Alias:          result.Alias,
		Description:    result.Description.String,
		NodeUUID:       result.NodeUUID,
		Status:         result.Status,
		Sort:           result.Sort,
		CreatedAt:      result.CreatedAt,
		CreatedBy:      result.CreatedBy,
		UpdatedAt:      getInt64Pointer(result.UpdatedAt),
		UpdatedBy:      getString(result.UpdatedBy),
		DeletedAt:      getInt64Pointer(result.DeletedAt),
		DeletedBy:      getString(result.DeletedBy),
		AliasForIndex:  getString(result.AliasForIndex),
		Icon:           getString(result.Icon),
		Logo:           getString(result.Logo),
	}

	return cleanBrand, nil
}

// Вспомогательные функции для получения указателя на int64 и строки
func getInt64Pointer(n sql.NullInt64) *int64 {
	if n.Valid {
		return &n.Int64
	}
	return nil
}

func getString(n sql.NullString) string {
	if n.Valid {
		return n.String
	}
	return ""
}
