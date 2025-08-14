package rdb

import (
	"context"
	"database/sql"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/usecase/transaction"
)

// txManager は、トランザクションの管理を行います。
type txManager struct {
	cfg *config.Config
	db  *sql.DB
}

// NewTransactionManager は、トランザクションマネージャを初期化します。
func NewTransactionManager(cfg *config.Config, db *sql.DB) transaction.Manager {
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

// ResolveConn は、context.Contextからトランザクションを取得します。
func ResolveConn(ctx context.Context, db *sql.DB) DBTX {
	tx, ok := ctx.Value(txKey{}).(*sql.Tx)
	if ok {
		return tx
	}
	return db
}

// withTx は、context.Contextにトランザクションを設定します。
func withTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}
