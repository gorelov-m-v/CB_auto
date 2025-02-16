package brand

import (
	"CB_auto/internal/database"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Knetic/go-namedParameterQuery"
	"github.com/google/uuid"
)

type Brand struct {
	UUID           uuid.UUID
	NodeUUID       uuid.UUID
	Alias          string
	LocalizedNames []byte
	Sort           int
	Status         int
	Description    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Repository struct {
	connector *database.Connector
}

func NewRepository(connector *database.Connector) *Repository {
	return &Repository{connector: connector}
}

func (r *Repository) GetBrand(ctx context.Context, brandUUID uuid.UUID) *Brand {
	q := `SELECT 
    	  	uuid, 
    	  	localized_names, 
    	  	alias, 
    	  	description, 
    	  	node_uuid, 
    	  	status, 
    	  	sort, 
    	  	created_at, 
    	  	created_by, 
    	  	updated_at, 
    	  	updated_by, 
    	  	deleted_at, 
    	  	deleted_by, 
    	  	alias_for_index, 
    	  	icon, 
    	  	logo 
		  FROM brand 
		  WHERE uuid = :uuid`

	params := map[string]any{"uuid": brandUUID}

	queryNamed := namedParameterQuery.NewNamedParameterQuery(q)
	queryNamed.SetValuesFromMap(params)

	var result Brand
	var createdBy string
	var updatedBy, deletedBy, aliasForIndex, icon, logo database.NullString
	var deletedAt database.NullInt64
	var localizedNamesRaw []byte
	var createdAtUnix int64
	var updatedAtUnix database.NullInt64
	var nodeUUIDStr string

	if err := r.connector.QueryRowContext(ctx, queryNamed.GetParsedQuery(), queryNamed.GetParsedParameters()...).Scan(
		&result.UUID,
		&localizedNamesRaw,
		&result.Alias,
		&result.Description,
		&nodeUUIDStr,
		&result.Status,
		&result.Sort,
		&createdAtUnix,
		&createdBy,
		&updatedAtUnix,
		&updatedBy,
		&deletedAt,
		&deletedBy,
		&aliasForIndex,
		&icon,
		&logo,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			panic(fmt.Sprintf("бренд не найден: %s", brandUUID))
		}
		panic(fmt.Sprintf("ошибка при получении бренда из БД: %v", err))
	}

	result.LocalizedNames = localizedNamesRaw
	result.CreatedAt = time.Unix(createdAtUnix, 0)
	if updatedAtUnix.Valid {
		result.UpdatedAt = time.Unix(updatedAtUnix.Int64, 0)
	}

	result.NodeUUID = uuid.MustParse(nodeUUIDStr)

	return &result
}
