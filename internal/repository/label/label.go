package label

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"CB_auto/internal/config"
	"CB_auto/internal/repository"
	"CB_auto/pkg/utils"

	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
)

type Label struct {
	ID             int    `db:"id"`
	UUID           string `db:"uuid"`
	Color          string `db:"color"`
	Node           string `db:"node"`
	UserID         string `db:"user_id"`
	UpdatedAt      int64  `db:"updated_at"`
	CreatedAt      int64  `db:"created_at"`
	AuthorCreation string `db:"author_creation"`
	AuthorEditing  string `db:"author_editing"`
	Description    string `db:"description"`
}

type Repository struct {
	db  *sql.DB
	cfg *config.MySQLConfig
}

var allowedFields = map[string]bool{
	"uuid":            true,
	"color":           true,
	"node":            true,
	"description":     true,
	"author_creation": true,
}

func NewRepository(db *sql.DB, mysqlConfig *config.MySQLConfig) *Repository {
	return &Repository{
		db:  db,
		cfg: mysqlConfig,
	}
}

func (r *Repository) GetLabel(sCtx provider.StepCtx, filters map[string]interface{}) *Label {
	if err := r.db.Ping(); err != nil {
		log.Printf("Ошибка подключения к БД: %v", err)
	}

	var conditions []string
	var args []interface{}

	query := `SELECT 
		id,
		uuid,
		color,
		node,
		user_id,
		UNIX_TIMESTAMP(updated_at) as updated_at,
		UNIX_TIMESTAMP(created_at) as created_at,
		author_creation,
		author_editing,
		description
	FROM label`

	if len(filters) > 0 {
		for key, value := range filters {
			if !allowedFields[key] {
				log.Printf("Недопустимое поле для фильтрации: %s", key)
				continue
			}
			conditions = append(conditions, fmt.Sprintf("%s = ?", key))
			args = append(args, value)
		}
		query += ` WHERE ` + strings.Join(conditions, " AND ")
	}

	var label Label
	err := repository.ExecuteWithRetry(sCtx, r.cfg, func(ctx context.Context) error {
		return r.db.QueryRowContext(ctx, query, args...).Scan(
			&label.ID,
			&label.UUID,
			&label.Color,
			&label.Node,
			&label.UserID,
			&label.UpdatedAt,
			&label.CreatedAt,
			&label.AuthorCreation,
			&label.AuthorEditing,
			&label.Description,
		)
	})

	if err != nil {
		log.Printf("Ошибка при получении данных лейбла: %v", err)
		return nil
	}

	sCtx.WithAttachments(allure.NewAttachment("Label DB Data", allure.JSON, utils.CreatePrettyJSON(label)))

	return &label
}
