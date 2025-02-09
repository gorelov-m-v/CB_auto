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

type Brand struct {
	UUID           string              `db:"uuid"            json:"UUID"`
	LocalizedNames json.RawMessage     `db:"localized_names" json:"LocalizedNames"`
	Alias          string              `db:"alias"           json:"Alias"`
	Description    database.NullString `db:"description"     json:"Description,omitempty"`
	NodeUUID       string              `db:"node_uuid"       json:"NodeUUID"`
	Status         int16               `db:"status"          json:"Status"`
	Sort           int                 `db:"sort"            json:"Sort"`
	CreatedAt      int64               `db:"created_at"      json:"CreatedAt"`
	CreatedBy      string              `db:"created_by"      json:"CreatedBy"`
	UpdatedAt      database.NullInt64  `db:"updated_at"      json:"UpdatedAt,omitempty"`
	UpdatedBy      database.NullString `db:"updated_by"      json:"UpdatedBy,omitempty"`
	DeletedAt      database.NullInt64  `db:"deleted_at"      json:"DeletedAt,omitempty"`
	DeletedBy      database.NullString `db:"deleted_by"      json:"DeletedBy,omitempty"`
	AliasForIndex  database.NullString `db:"alias_for_index" json:"AliasForIndex,omitempty"`
	Icon           database.NullString `db:"icon"            json:"Icon,omitempty"`
	Logo           database.NullString `db:"logo"            json:"Logo,omitempty"`
}

type Repository struct {
	connector *database.Connector
}

func NewRepository(connector *database.Connector) *Repository {
	return &Repository{connector: connector}
}

func (r *Repository) GetBrand(ctx context.Context, brandUUID uuid.UUID) (*Brand, error) {
	q := `SELECT uuid, localized_names, alias, description, node_uuid, status, sort, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by, alias_for_index, icon, logo 
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
		&result.DeletedAt,
		&result.DeletedBy,
		&result.AliasForIndex,
		&result.Icon,
		&result.Logo,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &result, nil
}
