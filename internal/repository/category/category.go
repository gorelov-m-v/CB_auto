package category

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
)

type Category struct {
	ID               int64             `db:"id"`
	Alias            string            `db:"alias"`
	CreatedAt        int64             `db:"created_at"`
	UpdatedAt        int64             `db:"updated_at"`
	UUID             string            `db:"uuid"`
	ProjectGroupUUID string            `db:"project_group_uuid"`
	ProjectUUID      string            `db:"project_uuid"`
	StatusID         int16             `db:"status_id"`
	Sort             uint32            `db:"sort"`
	IsDefault        bool              `db:"is_default"`
	LocalizedNames   map[string]string `db:"localized_names"`
	Type             string            `db:"type"`
	CMS              bool              `db:"cms"`
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
	"uuid":      true,
	"alias":     true,
	"status_id": true,
	"type":      true,
}

func (r *Repository) fetchCategory(sCtx provider.StepCtx, filters map[string]interface{}) (*Category, error) {
	if err := r.db.Ping(); err != nil {
		log.Printf("Ошибка подключения к БД: %v", err)
	}

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
		localized_names,
		type,
		cms
	FROM game_category`
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
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	log.Printf("Executing query: %s with args: %v", query, args)
	log.Printf("Using database: %v", r.db.Stats())

	var category Category
	var localizedNamesRaw []byte

	err := repository.ExecuteWithRetry(sCtx, r.cfg, func(ctx context.Context) error {
		return r.db.QueryRowContext(ctx, query, args...).Scan(
			&category.ID,
			&category.Alias,
			&category.CreatedAt,
			&category.UpdatedAt,
			&category.UUID,
			&category.ProjectGroupUUID,
			&category.ProjectUUID,
			&category.StatusID,
			&category.Sort,
			&category.IsDefault,
			&localizedNamesRaw,
			&category.Type,
			&category.CMS,
		)
	})
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(localizedNamesRaw, &category.LocalizedNames); err != nil {
		log.Printf("Ошибка декодирования JSON: %v", err)
		return nil, err
	}

	sCtx.WithAttachments(allure.NewAttachment("Category DB Data", allure.JSON, utils.CreatePrettyJSON(category)))
	return &category, nil
}

func (r *Repository) GetCategoryWithRetry(sCtx provider.StepCtx, filters map[string]interface{}) *Category {
	category, err := r.fetchCategory(sCtx, filters)
	if err != nil {
		log.Printf("Ошибка при получении данных категории: %v", err)
	}
	return category
}

func (r *Repository) GetCategory(sCtx provider.StepCtx, filters map[string]interface{}) (*Category, error) {
	category, err := r.fetchCategory(sCtx, filters)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Нет данных категории, возвращаем nil: %v", err)
			return nil, nil
		}
		log.Printf("Ошибка при получении данных категории: %v", err)
		return nil, err
	}
	return category, nil
}
