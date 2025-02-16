package brand

import (
	"CB_auto/internal/database"
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/ozontech/allure-go/pkg/framework/provider"
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
	database.Repository
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		Repository: database.NewRepository(db),
	}
}

func (r *Repository) GetBrandByUUID(t provider.T, brandID uuid.UUID) *Brand {
	query := `
		SELECT 
			uuid,
			alias,
			localized_names,
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
		WHERE uuid = ?
	`

	var brand Brand
	var createdBy string
	var updatedBy, deletedBy, aliasForIndex, icon, logo database.NullString
	var deletedAt database.NullInt64
	var localizedNamesRaw []byte
	var createdAtUnix int64
	var updatedAtUnix database.NullInt64
	var nodeUUIDStr string

	err := r.ExecuteWithRetry(context.Background(), func(ctx context.Context) error {
		return r.DB().QueryRowContext(ctx, query, brandID).Scan(
			&brand.UUID,
			&brand.Alias,
			&localizedNamesRaw,
			&brand.Description,
			&nodeUUIDStr,
			&brand.Status,
			&brand.Sort,
			&createdAtUnix,
			&createdBy,
			&updatedAtUnix,
			&updatedBy,
			&deletedAt,
			&deletedBy,
			&aliasForIndex,
			&icon,
			&logo,
		)
	})

	if err != nil {
		t.Fatalf("Ошибка при получении данных бренда: %v", err)
	}

	brand.LocalizedNames = localizedNamesRaw
	brand.CreatedAt = time.Unix(createdAtUnix, 0)
	if updatedAtUnix.Valid {
		brand.UpdatedAt = time.Unix(updatedAtUnix.Int64, 0)
	}

	nodeUUID, err := uuid.Parse(nodeUUIDStr)
	if err != nil {
		t.Fatalf("Ошибка при парсинге node UUID: %v", err)
	}
	brand.NodeUUID = nodeUUID

	return &brand
}
