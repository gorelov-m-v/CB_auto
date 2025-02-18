package brand

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"CB_auto/internal/database"

	"github.com/google/uuid"
	"github.com/ozontech/allure-go/pkg/framework/provider"
)

type Brand struct {
	UUID           uuid.UUID `db:"uuid"`
	Alias          string    `db:"alias"`
	LocalizedNames []byte    `db:"localized_names"`
	Description    string    `db:"description"`
	NodeUUID       uuid.UUID `db:"node_uuid"`
	Status         int       `db:"status"`
	Sort           int       `db:"sort"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

type Repository struct {
	database.Repository
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		Repository: database.NewRepository(db),
	}
}

func (r *Repository) GetBrand(t provider.T, filters map[string]interface{}) *Brand {
	conditions := []string{}
	args := []interface{}{}

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
			updated_at
		FROM brand
	`

	if len(filters) > 0 {
		for key, value := range filters {
			conditions = append(conditions, fmt.Sprintf("%s = ?", key))
			args = append(args, value)
		}
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	log.Printf("Executing query: %s with args: %v", query, args)

	var brand Brand
	var createdAtUnix int64
	var updatedAtUnix sql.NullInt64
	var nodeUUIDStr string
	var localizedNamesRaw []byte

	err := r.ExecuteWithRetry(context.Background(), func(ctx context.Context) error {
		return r.DB().QueryRowContext(ctx, query, args...).Scan(
			&brand.UUID,
			&brand.Alias,
			&localizedNamesRaw,
			&brand.Description,
			&nodeUUIDStr,
			&brand.Status,
			&brand.Sort,
			&createdAtUnix,
			&updatedAtUnix,
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
