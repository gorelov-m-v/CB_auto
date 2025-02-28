package category

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

	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
)

type Category struct {
	ID               int               `db:"id"`
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

func (r *Repository) GetCategory(sCtx provider.StepCtx, filters map[string]interface{}) *Category {
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
		localized_names,
		type,
		cms
	FROM game_category`

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

	var category Category
	var localizedNamesRaw []byte

	err := repository.ExecuteWithRetry(context.Background(), r.cfg, func(ctx context.Context) error {
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
		log.Printf("Ошибка при получении данных категории: %v", err)
		return nil
	}

	if err := json.Unmarshal(localizedNamesRaw, &category.LocalizedNames); err != nil {
		log.Printf("Ошибка декодирования JSON: %v", err)
		return nil
	}

	sCtx.WithAttachments(allure.NewAttachment("Category DB Data", allure.JSON, utils.CreatePrettyJSON(category)))
	return &category
}
