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
	"github.com/shopspring/decimal"
)

type LimitType string
type IntervalType string

const (
	// Типы лимитов
	LimitTypeSingleBet     LimitType = "single-bet"
	LimitTypeCasinoLoss    LimitType = "casino-loss"
	LimitTypeTurnoverFunds LimitType = "turnover-of-funds"

	// Типы интервалов
	IntervalTypeDaily   IntervalType = "daily"
	IntervalTypeWeekly  IntervalType = "weekly"
	IntervalTypeMonthly IntervalType = "monthly"
)

type LimitRecord struct {
	ExternalUUID string          `db:"external_uuid"`
	PlayerUUID   string          `db:"player_uuid"`
	LimitType    LimitType       `db:"limit_type"`
	IntervalType IntervalType    `db:"interval_type"`
	Amount       decimal.Decimal `db:"amount"`
	Spent        decimal.Decimal `db:"spent"`
	Rest         decimal.Decimal `db:"rest"`
	CurrencyCode string          `db:"currency_code"`
	StartedAt    int             `db:"started_at"`
	ExpiresAt    int             `db:"expires_at"`
	LimitStatus  bool            `db:"limit_status"`
}

type LimitRecordRepository struct {
	db  *sql.DB
	cfg *config.MySQLConfig
}

func NewLimitRecordRepository(db *sql.DB, mysqlConfig *config.MySQLConfig) *LimitRecordRepository {
	return &LimitRecordRepository{
		db:  db,
		cfg: mysqlConfig,
	}
}

var allowedFields = map[string]bool{
	"external_uuid": true,
	"player_uuid":   true,
	"limit_type":    true,
	"interval_type": true,
	"currency_code": true,
	"limit_status":  true,
}

func (r *LimitRecordRepository) fetchLimitRecord(sCtx provider.StepCtx, filters map[string]interface{}) (*LimitRecord, error) {
	if err := r.db.Ping(); err != nil {
		log.Printf("Ошибка подключения к БД: %v", err)
	}

	query := `SELECT 
		external_uuid,
		player_uuid,
		limit_type,
		interval_type,
		amount,
		spent,
		rest,
		currency_code,
		started_at,
		expires_at,
		limit_status
	FROM limit_record`

	var conditions []string
	var args []interface{}
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

	var limitRecord LimitRecord
	err := repository.ExecuteWithRetry(sCtx, r.cfg, func(ctx context.Context) error {
		return r.db.QueryRowContext(ctx, query, args...).Scan(
			&limitRecord.ExternalUUID,
			&limitRecord.PlayerUUID,
			&limitRecord.LimitType,
			&limitRecord.IntervalType,
			&limitRecord.Amount,
			&limitRecord.Spent,
			&limitRecord.Rest,
			&limitRecord.CurrencyCode,
			&limitRecord.StartedAt,
			&limitRecord.ExpiresAt,
			&limitRecord.LimitStatus,
		)
	})
	if err != nil {
		return nil, err
	}
	sCtx.WithAttachments(allure.NewAttachment("Limit Record DB Data", allure.JSON, utils.CreatePrettyJSON(limitRecord)))
	return &limitRecord, nil
}

func (r *LimitRecordRepository) GetLimitRecordWithRetry(sCtx provider.StepCtx, filters map[string]interface{}) *LimitRecord {
	limitRecord, err := r.fetchLimitRecord(sCtx, filters)
	if err != nil {
		log.Printf("Ошибка при получении данных лимита: %v", err)
	}
	return limitRecord
}

func (r *LimitRecordRepository) GetLimitRecord(sCtx provider.StepCtx, filters map[string]interface{}) (*LimitRecord, error) {
	limitRecord, err := r.fetchLimitRecord(sCtx, filters)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Нет данных лимита, возвращаем nil: %v", err)
			return nil, nil
		}
		log.Printf("Ошибка при получении данных лимита: %v", err)
		return nil, err
	}
	return limitRecord, nil
}
