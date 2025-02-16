package wallet

import (
	"context"
	"database/sql"

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
	database.Repository
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		Repository: database.NewRepository(db),
	}
}

func (r *Repository) GetWalletByUUID(t provider.T, uuid string) *Wallet {
	query := `
        SELECT * FROM wallet 
        WHERE uuid = ?
    `

	var wallet Wallet
	err := r.ExecuteWithRetry(context.Background(), func(ctx context.Context) error {
		return r.DB().QueryRowContext(ctx, query, uuid).Scan(
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
