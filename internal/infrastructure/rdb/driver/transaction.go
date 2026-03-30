package driver

import (
	"context"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/infrastructure/rdb/postgres/pgerror"
	"boilerplate-go/internal/logging"
	"boilerplate-go/internal/usecase/boundary/tx"

	"github.com/jackc/pgx/v5"
)

const callerSkipCount = 1

// txManager は、トランザクションの管理を行います。
type txManager struct {
	cfg    *config.Config
	db     DatabaseDriver
	logger logging.Logger
}

// NewTransactionManager は、トランザクションマネージャを初期化します。
func NewTransactionManager(cfg *config.Config, db DatabaseDriver, logger logging.Logger) tx.Manager {
	return &txManager{
		cfg:    cfg,
		db:     db,
		logger: logger,
	}
}

// Do は、トランザクションを開始し、引数で渡されたfnを実行します。
func (t *txManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	tx, err := t.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			if pgErr := tx.Rollback(ctx); pgErr != nil {
				t.logger.CallerSkip(callerSkipCount).Named("TransactionManager").Error(
					"Failed to rollback transaction on panic", logging.Error("rollback transaction", pgErr),
				)
			}
			panic(p)
		}
	}()

	ctx = withTx(ctx, tx)

	if err := fn(ctx); err != nil {
		if pgErr := tx.Rollback(ctx); pgErr != nil {
			t.logger.CallerSkip(callerSkipCount).Named("TransactionManager").Error(
				"Failed to rollback transaction",
				logging.Error("rollback transaction", pgErr),
				logging.Error("original error", err),
			)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return pgerror.NormalizeError(err)
	}

	return nil
}

// withTx は、context.Contextにトランザクションを設定します。
func withTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}
