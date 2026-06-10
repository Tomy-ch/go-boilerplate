package driver

import (
	"context"
	"time"

	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/usecase/boundary/tx"

	"github.com/jackc/pgx/v5"
)

const (
	callerSkipCount = 1
	cleanupTimeout  = 5 * time.Second
)

// txManager は、トランザクションの管理を行います。
type txManager struct {
	db     DatabaseDriver
	logger logging.Logger
}

// NewTransactionManager は、トランザクションマネージャを初期化します。
func NewTransactionManager(db DatabaseDriver, logger logging.Logger) tx.Manager {
	return &txManager{
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
		// Commit と同様、txManager 自身が発行する DB 操作のエラーはアプリエラー語彙へ正規化する。
		return pgerror.NormalizeError(err)
	}
	defer func(ctx context.Context) {
		if p := recover(); p != nil {
			t.rollback(ctx, tx, logging.Any("panic", p))
			panic(p)
		}
	}(ctx)

	ctx = withTx(ctx, tx)

	if err := fn(ctx); err != nil {
		t.rollback(ctx, tx, logging.Error(logging.OriginalErrorKey, err))
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return pgerror.NormalizeError(err)
	}

	return nil
}

// rollback はロールバックし、失敗時に fields を併記してログを残す。
// 呼び出し元 ctx がキャンセル済みでも後始末を完了させるため WithoutCancel で切り離す。
func (t *txManager) rollback(ctx context.Context, tx pgx.Tx, fields ...*logging.Field) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()

	if pgErr := tx.Rollback(cleanupCtx); pgErr != nil {
		logFields := append([]*logging.Field{logging.Error(logging.ErrorKey, pgErr)}, fields...)
		t.logger.CallerSkip(callerSkipCount).Named("TransactionManager").Error(
			"Failed to rollback transaction", logFields...,
		)
	}
}

// withTx は、context.Contextにトランザクションを設定します。
func withTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}
