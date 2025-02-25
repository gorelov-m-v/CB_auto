package brand

import (
	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/pkg/utils"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
)

type Brand struct {
	UUID           string            `db:"uuid"`
	Alias          string            `db:"alias"`
	LocalizedNames map[string]string `db:"localized_names"`
	Description    string            `db:"description"`
	NodeUUID       string            `db:"node_uuid"`
	Status         int               `db:"status"`
	Sort           int               `db:"sort"`
	CreatedAt      time.Time         `db:"created_at"`
	UpdatedAt      time.Time         `db:"updated_at"`
}

type Repository struct {
	db  *sql.DB
	cfg *config.MySQLConfig
}

func NewRepository(db *sql.DB, mysqlConfig *config.MySQLConfig) *Repository {
	return &Repository{
		db:  db,
		cfg: mysqlConfig,
	}
}

var allowedFields = map[string]bool{
	"uuid":   true,
	"alias":  true,
	"status": true,
}

func (r *Repository) GetBrand(sCtx provider.StepCtx, filters map[string]interface{}) *Brand {
	if err := r.db.Ping(); err != nil {
		log.Printf("Ошибка подключения к БД: %v", err)
	}

	conditions := []string{}
	args := []interface{}{}

	query := `SELECT 
		uuid,
		alias,
		localized_names,
		description,
		node_uuid,
		status,
		sort,
		created_at,
		updated_at
	FROM brand`

	if len(filters) > 0 {
		for key, value := range filters {
			if !allowedFields[key] {
				log.Printf("Недопустимое поле для фильтрации: %s", key)
			}
			conditions = append(conditions, fmt.Sprintf("%s = ?", key))
			args = append(args, value)
		}
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	log.Printf("Executing query: %s with args: %v", query, args)
	log.Printf("Using database: %v", r.db.Stats())

	var brand Brand
	var createdAtUnix int64
	var updatedAtUnix sql.NullInt64
	var nodeUUIDStr string
	var localizedNamesRaw []byte

	err := repository.ExecuteWithRetry(context.Background(), r.cfg, func(ctx context.Context) error {
		return r.db.QueryRowContext(ctx, query, args...).Scan(
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
		log.Printf("Ошибка при получении данных бренда: %v", err)
		return nil
	}

	// Декодирование JSON с обработкой
	if err := json.Unmarshal(localizedNamesRaw, &brand.LocalizedNames); err != nil {
		log.Printf("Ошибка декодирования JSON: %v", err)
		return nil
	}

	brand.CreatedAt = time.Unix(createdAtUnix, 0)
	if updatedAtUnix.Valid {
		brand.UpdatedAt = time.Unix(updatedAtUnix.Int64, 0)
	}

	brand.NodeUUID = nodeUUIDStr

	sCtx.WithAttachments(allure.NewAttachment("Brand DB Data", allure.JSON, utils.CreatePrettyJSON(brand)))
	return &brand
}
