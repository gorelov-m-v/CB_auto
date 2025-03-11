package brand

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/pkg/utils"

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

func (r *Repository) fetchBrand(sCtx provider.StepCtx, filters map[string]interface{}, withRetry bool) (*Brand, error) {
	if err := r.db.Ping(); err != nil {
		log.Printf("Ошибка подключения к БД: %v", err)
	}

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
	var conditions []string
	var args []interface{}

	if len(filters) > 0 {
		for key, value := range filters {
			if !allowedFields[key] {
				log.Printf("Недопустимое поле для фильтрации: %s", key)
				continue
			}
			conditions = append(conditions, fmt.Sprintf("%s = ?", key))
			args = append(args, value)
		}
		if len(conditions) > 0 {
			query += " WHERE " + strings.Join(conditions, " AND ")
		}
	}
	log.Printf("Executing query: %s with args: %v", query, args)
	log.Printf("Using database: %v", r.db.Stats())

	var brand Brand
	var createdAtUnix int64
	var updatedAtUnix sql.NullInt64
	var nodeUUIDStr string
	var localizedNamesRaw []byte

	execQuery := func(ctx context.Context) error {
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
	}

	var err error
	if withRetry {
		err = repository.ExecuteWithRetry(sCtx, r.cfg, execQuery)
	} else {
		err = execQuery(context.Background())
	}
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Нет данных бренда, возвращаем nil: %v", err)
			return nil, nil
		}
		log.Printf("Ошибка при получении данных бренда: %v", err)
		return nil, err
	}

	if err := json.Unmarshal(localizedNamesRaw, &brand.LocalizedNames); err != nil {
		log.Printf("Ошибка декодирования JSON: %v", err)
		return nil, err
	}

	brand.CreatedAt = time.Unix(createdAtUnix, 0)
	if updatedAtUnix.Valid {
		brand.UpdatedAt = time.Unix(updatedAtUnix.Int64, 0)
	}
	brand.NodeUUID = nodeUUIDStr

	sCtx.WithAttachments(allure.NewAttachment("Brand DB Data", allure.JSON, utils.CreatePrettyJSON(brand)))
	return &brand, nil
}

func (r *Repository) GetBrandWithRetry(sCtx provider.StepCtx, filters map[string]interface{}) *Brand {
	brand, err := r.fetchBrand(sCtx, filters, true)
	if err != nil {
		log.Printf("Ошибка при получении данных бренда: %v", err)
	}
	return brand
}

func (r *Repository) GetBrand(sCtx provider.StepCtx, filters map[string]interface{}) (*Brand, error) {
	brand, err := r.fetchBrand(sCtx, filters, false)
	if err != nil {
		return nil, err
	}
	return brand, nil
}
