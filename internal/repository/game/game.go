package game

import (
	"context"
	"database/sql"

	"CB_auto/internal/config"
	"CB_auto/internal/repository"

	"github.com/ozontech/allure-go/pkg/framework/provider"
)

type Game struct {
	UUID  string `db:"uuid"`
	Alias string `db:"alias"`
	Name  string `db:"name"`
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

func (r *Repository) GetGame(sCtx provider.StepCtx, filters map[string]interface{}) *Game {
	var game Game
	var whereClause string
	var args []interface{}

	for key, value := range filters {
		if whereClause != "" {
			whereClause += " AND "
		}
		whereClause += key + " = ?"
		args = append(args, value)
	}

	query := "SELECT uuid, alias, name FROM game WHERE " + whereClause

	err := repository.ExecuteWithRetry(sCtx, r.cfg, func(ctx context.Context) error {
		row := r.db.QueryRow(query, args...)
		err := row.Scan(
			&game.UUID,
			&game.Alias,
			&game.Name,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}
		return nil
	})

	if err != nil {
		sCtx.Errorf("Failed to get game: %v", err)
		return nil
	}

	return &game
}
