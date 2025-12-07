package driver

import (
	"context"
	"database/sql"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/usecase/tx"
)

// txManager は、トランザクションの管理を行います。
type txManager struct {
	cfg *config.Config
	db  DatabaseDriver
}

// NewTransactionManager は、トランザクションマネージャを初期化します。
func NewTransactionManager(cfg *config.Config, db DatabaseDriver) tx.Manager {
	return &txManager{
		cfg: cfg,
		db:  db,
	}
}

// Do は、トランザクションを開始し、引数で渡されたfnを実行します。
func (t *txManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return fn(ctx)
	}

	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	ctx = withTx(ctx, tx)

	if err := fn(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// withTx は、context.Contextにトランザクションを設定します。
func withTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}
