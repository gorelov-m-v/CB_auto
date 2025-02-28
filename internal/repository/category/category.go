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

type Category struct {
	ID               string            `db:"id"`
	Alias            string            `db:"alias"`
	CreatedAt        time.Time         `db:"created_at"`
	UpdatedAt        time.Time         `db:"updated_at"`
	UUID             string            `db:"uuid"`
	ProjectGroupUUID string            `db:"project_group_uuid"`
	ProjectUUID      string            `db:"project_uuid"`
	StatusID         int               `db:"status_id"`
	Sort             int               `db:"sort"`
	IsDefault        int               `db:"is_default"`
	LocalizedNames   map[string]string `db:"localized_names"`
	Type             string            `db:"type"`
	PassToCms        int               `db:"cms"`
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

func (r *Repository) GetCategory(sCtx provider.StepCtx, filters map[string]interface{}) *Category {
	if err := r.db.Ping(); err != nil {
		log.Printf("Ошибка подключения к БД: %v", err)
	}

	conditions := []string{}
	args := []interface{}{}

	query := `SELECT 
		id,
		alias,
		created_at,
		updated_at,
		uuid,
		project_group_uuid,
		project_uuid,
		status_id,
		sort,
		is_default,
		type,
		cms
	FROM game_category`

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

	var category Category
	var createdAtUnix int64
	var updatedAtUnix sql.NullInt64
	var nodeUUIDStr string
	var localizedNamesRaw []byte

	err := repository.ExecuteWithRetry(context.Background(), r.cfg, func(ctx context.Context) error {
		return r.db.QueryRowContext(ctx, query, args...).Scan(
			&category.ID,
			&category.Alias,
			&createdAtUnix,
			&updatedAtUnix,
			&category.UUID,
			&category.ProjectGroupUUID,
			&localizedNamesRaw,
			&category.ProjectUUID,
			&category.StatusID,
			&category.Sort,
			&category.IsDefault,
			&category.Type,
			&category.PassToCms,
		)
	})
	if err != nil {
		log.Printf("Ошибка при получении данных категории: %v", err)
		return nil
	}

	if err := json.Unmarshal(localizedNamesRaw, &category.LocalizedNames); err != nil {
		log.Printf("Ошибка декодирования JSON: %v", err)
		return nil
	}

	category.CreatedAt = time.Unix(createdAtUnix, 0)
	if updatedAtUnix.Valid {
		category.UpdatedAt = time.Unix(updatedAtUnix.Int64, 0)
	}

	category.ProjectUUID = nodeUUIDStr

	sCtx.WithAttachments(allure.NewAttachment("Category DB Data", allure.JSON, utils.CreatePrettyJSON(category)))
	return &category
}
