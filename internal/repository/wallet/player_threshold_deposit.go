package wallet

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

// PlayerThresholdDeposit представляет запись в таблице player_threshold_deposit
type PlayerThresholdDeposit struct {
	PlayerUUID string `db:"player_uuid"`
	Amount     string `db:"amount"`
	UpdatedAt  int64  `db:"updated_at"`
}

type PlayerThresholdDepositRepository struct {
	db  *sql.DB
	cfg *config.MySQLConfig
}

func NewPlayerThresholdDepositRepository(db *sql.DB, mysqlConfig *config.MySQLConfig) *PlayerThresholdDepositRepository {
	return &PlayerThresholdDepositRepository{
		db:  db,
		cfg: mysqlConfig,
	}
}

var playerThresholdDepositAllowedFields = map[string]bool{
	"player_uuid": true,
	"amount":      true,
	"updated_at":  true,
}

func (r *PlayerThresholdDepositRepository) fetchPlayerThresholdDeposit(sCtx provider.StepCtx, filters map[string]interface{}) (*PlayerThresholdDeposit, error) {
	if err := r.db.Ping(); err != nil {
		log.Printf("Ошибка подключения к БД: %v", err)
	}

	query := `SELECT 
		player_uuid,
		IF(amount LIKE '%.000000000000000000', 
		   SUBSTRING_INDEX(amount, '.', 1), 
		   amount) as amount,
		updated_at
	FROM player_threshold_deposit`
	var conditions []string
	var args []interface{}
	if len(filters) > 0 {
		for key, value := range filters {
			if !playerThresholdDepositAllowedFields[key] {
				log.Printf("Недопустимое поле для фильтрации: %s", key)
				continue
			}
			conditions = append(conditions, fmt.Sprintf("%s = ?", key))
			args = append(args, value)
		}
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	log.Printf("Executing query: %s with args: %v", query, args)

	var record PlayerThresholdDeposit
	err := repository.ExecuteWithRetry(sCtx, r.cfg, func(ctx context.Context) error {
		return r.db.QueryRowContext(ctx, query, args...).Scan(
			&record.PlayerUUID,
			&record.Amount,
			&record.UpdatedAt,
		)
	})
	if err != nil {
		return nil, err
	}
	sCtx.WithAttachments(allure.NewAttachment("PlayerThresholdDeposit DB Data", allure.JSON, utils.CreatePrettyJSON(record)))
	return &record, nil
}

func (r *PlayerThresholdDepositRepository) GetPlayerThresholdDepositWithRetry(sCtx provider.StepCtx, filters map[string]interface{}) *PlayerThresholdDeposit {
	record, err := r.fetchPlayerThresholdDeposit(sCtx, filters)
	if err != nil {
		log.Printf("Ошибка при получении данных порога депозита игрока: %v", err)
	}
	return record
}

func (r *PlayerThresholdDepositRepository) GetPlayerThresholdDeposit(sCtx provider.StepCtx, filters map[string]interface{}) (*PlayerThresholdDeposit, error) {
	record, err := r.fetchPlayerThresholdDeposit(sCtx, filters)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Нет данных порога депозита игрока, возвращаем nil: %v", err)
			return nil, nil
		}
		log.Printf("Ошибка при получении данных порога депозита игрока: %v", err)
		return nil, err
	}
	return record, nil
}
