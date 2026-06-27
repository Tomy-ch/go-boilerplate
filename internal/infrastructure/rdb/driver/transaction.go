package driver

import (
	"context"
	"errors"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/pkg/backoff"
	"go-boilerplate/pkg/retry"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	callerSkipCount = 1
	cleanupTimeout  = 5 * time.Second

	// defaultTxMaxAttempts は、tx リトライの最大試行回数の既定値です。
	defaultTxMaxAttempts = 3
	// defaultTxBackoffInitial / defaultTxBackoffMax は、tx リトライ backoff の既定値です。
	defaultTxBackoffInitial = 5 * time.Millisecond
	defaultTxBackoffMax     = 100 * time.Millisecond
	// txBackoffMultiplier は、tx リトライ backoff の倍率です。
	txBackoffMultiplier = 2
)

// txManager は、トランザクションの管理を行います。
type txManager struct {
	db          DatabaseDriver
	logger      logging.Logger
	sleeper     clock.Sleeper
	maxAttempts int
	backoff     backoff.Exponential
}

// NewTransactionManager は、トランザクションマネージャを初期化します。
//
// serialization failure / deadlock 検出時に fn を有限回まで再試行します。
// リトライ上限・backoff は config（DB_TX_MAX_RETRIES / DB_TX_RETRY_BASE_BACKOFF /
// DB_TX_RETRY_MAX_BACKOFF）から取得します。0 以下の場合は既定値にフォールバックします。
func NewTransactionManager(
	db DatabaseDriver, dbCfg *config.DatabaseConfig, logger logging.Logger, sleeper clock.Sleeper,
) tx.Manager {
	maxAttempts := dbCfg.TxMaxRetries()
	if maxAttempts <= 0 {
		maxAttempts = defaultTxMaxAttempts
	}
	initialBackoff := dbCfg.TxRetryBaseBackoff()
	if initialBackoff <= 0 {
		initialBackoff = defaultTxBackoffInitial
	}
	maxBackoff := dbCfg.TxRetryMaxBackoff()
	if maxBackoff <= 0 {
		maxBackoff = defaultTxBackoffMax
	}

	return &txManager{
		db:          db,
		logger:      logger,
		sleeper:     sleeper,
		maxAttempts: maxAttempts,
		backoff: backoff.Exponential{
			Initial:    initialBackoff,
			Max:        maxBackoff,
			Multiplier: txBackoffMultiplier,
		},
	}
}

// Do は、トランザクションを開始し、引数で渡された fn を実行します。
//
// serialization failure / deadlock を検出した場合、有限回（maxAttempts）まで tx 全体を
// 再試行します（指数 backoff + full jitter、pkg/retry）。**fn は最大 N 回再実行されうる**ため、
// DB 副作用以外について冪等であること（呼出側責務）。外部副作用は同一 tx 内で outbox row 化すれば
// rollback と共に巻き戻り retry-safe になります。nested（既存 tx 再利用）経路はリトライ対象外（1 回）。
func (t *txManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx) // nested: savepoint 相当・リトライしない（最外の Do が正規化する）
	}

	err := retry.Do(ctx, t.sleeper,
		retry.Policy{MaxAttempts: t.maxAttempts, Backoff: t.backoff.Duration},
		pgerror.IsRetryableTxError,
		func(c context.Context) error { return t.doOnce(c, fn) },
	)
	return normalizeTxResult(err)
}

// normalizeTxResult は、リトライ後の最終エラーを apperror へ正規化します。
// begin / commit 由来の DB エラー（生 pg / 接続）および context キャンセル・期限超過を
// apperror へ写像します。fn が返したエラー（apperror や rollback sentinel 等）は
// そのまま通し、呼出側のエラー判定に干渉しません。
func normalizeTxResult(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) || pgerror.IsUnavailable(err) {
		return pgerror.NormalizeError(err)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return pgerror.NormalizeError(err)
	}
	return err
}

// doOnce は、1 回分のトランザクション（begin → fn → commit / rollback）を実行します。
//
// エラーは正規化せず**生のまま**返します。serialization failure / deadlock の判定（IsRetryableTxError）は
// 生 SQLSTATE を要するため、正規化（PgError を文字列化して捨てる）は呼出元 Do がリトライ後に 1 度だけ行います。
func (t *txManager) doOnce(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := t.db.Begin(ctx)
	if err != nil {
		return err
	}

	completed := false
	defer func(ctx context.Context) {
		if completed {
			return
		}
		if p := recover(); p != nil {
			t.rollback(ctx, tx, logging.Any("panic", p))
			panic(p)
		}
		// fn が runtime.Goexit（testify の FailNow 等）で中断した場合の後始末。
		t.rollback(ctx, tx)
	}(ctx)

	ctx = withTx(ctx, tx)

	if err := fn(ctx); err != nil {
		t.rollback(ctx, tx, logging.Error(logging.OriginalErrorKey, err))
		completed = true
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		completed = true
		return err
	}

	completed = true
	return nil
}

// rollback はロールバックし、失敗時に fields を併記してログを残します。
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
