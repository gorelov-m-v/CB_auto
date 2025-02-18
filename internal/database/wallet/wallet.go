package wallet

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"CB_auto/internal/config"
	"CB_auto/internal/database"

	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/shopspring/decimal"
)

type Wallet struct {
	UUID                string          `db:"uuid"`
	PlayerUUID          string          `db:"player_uuid"`
	Currency            string          `db:"currency"`
	WalletStatus        int             `db:"wallet_status"`
	Balance             decimal.Decimal `db:"balance"`
	CreatedAt           int             `db:"created_at"`
	UpdatedAt           sql.NullInt64   `db:"updated_at"`
	IsDefault           bool            `db:"is_default"`
	IsBasic             bool            `db:"is_basic"`
	IsBlocked           bool            `db:"is_blocked"`
	WalletType          int             `db:"wallet_type"`
	Seq                 int             `db:"seq"`
	IsGamblingActive    bool            `db:"is_gambling_active"`
	IsBettingActive     bool            `db:"is_betting_active"`
	DepositAmount       decimal.Decimal `db:"deposit_amount"`
	ProfitAmount        decimal.Decimal `db:"profit_amount"`
	NodeUUID            sql.NullString  `db:"node_uuid"`
	IsSumsubVerified    bool            `db:"is_sumsub_verified"`
	AvailableWithdrawal decimal.Decimal `db:"available_withdrawal"`
	IsKycVerified       bool            `db:"is_kyc_verified"`
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
	"uuid":          true,
	"player_uuid":   true,
	"currency":      true,
	"wallet_status": true,
	"balance":       true,
}

func (r *Repository) GetWallet(t provider.T, filters map[string]interface{}) *Wallet {
	if err := r.db.Ping(); err != nil {
		t.Fatalf("Ошибка подключения к БД: %v", err)
	}

	var conditions []string
	var args []interface{}

	query := `SELECT 
		uuid,
		player_uuid,
		currency,
		wallet_status,
		balance,
		created_at,
		updated_at,
		is_default,
		is_basic,
		is_blocked,
		wallet_type,
		seq,
		is_gambling_active,
		is_betting_active,
		deposit_amount,
		profit_amount,
		node_uuid,
		is_sumsub_verified,
		available_withdrawal,
		is_kyc_verified
	FROM wallet`

	if len(filters) > 0 {
		for key, value := range filters {
			if !allowedFields[key] {
				t.Fatalf("Недопустимое поле для фильтрации: %s", key)
			}
			conditions = append(conditions, fmt.Sprintf("%s = ?", key))
			args = append(args, value)
		}
		query += ` WHERE ` + strings.Join(conditions, " AND ")
	}

	log.Printf("Executing query: %s with args: %v", query, args)
	log.Printf("Using database: %v", r.db.Stats())

	var wallet Wallet
	err := database.ExecuteWithRetry(context.Background(), r.cfg, func(ctx context.Context) error {
		return r.db.QueryRowContext(ctx, query, args...).Scan(
			&wallet.UUID,
			&wallet.PlayerUUID,
			&wallet.Currency,
			&wallet.WalletStatus,
			&wallet.Balance,
			&wallet.CreatedAt,
			&wallet.UpdatedAt,
			&wallet.IsDefault,
			&wallet.IsBasic,
			&wallet.IsBlocked,
			&wallet.WalletType,
			&wallet.Seq,
			&wallet.IsGamblingActive,
			&wallet.IsBettingActive,
			&wallet.DepositAmount,
			&wallet.ProfitAmount,
			&wallet.NodeUUID,
			&wallet.IsSumsubVerified,
			&wallet.AvailableWithdrawal,
			&wallet.IsKycVerified,
		)
	})
	if err != nil {
		t.Fatalf("Ошибка при получении данных кошелька: %v", err)
	}

	return &wallet
}
