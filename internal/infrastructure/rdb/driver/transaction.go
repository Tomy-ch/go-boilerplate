package driver

import (
	"context"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/pkg/backoff"
	"go-boilerplate/pkg/retry"
	"go-boilerplate/pkg/xerrors"

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

type txManager struct {
	db          DatabaseDriver
	logger      logging.Logger
	sleeper     clock.Sleeper
	maxAttempts int
	backoff     backoff.Exponential
}

// NewTransactionManager は、リトライ上限と backoff を config から読んで組み立てます。
// 0 以下の値は既定値へ倒します（リトライ方針そのものは ADR-0035）。
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

// Do は、pkg/retry で tx 全体を包み、指数 backoff + full jitter で有限回まで再試行します。
// 何を再試行の対象とするか、fn に課される冪等性契約、nested が 1 回だけであることは
// tx.Manager（internal/usecase/boundary/tx）の doc が述べています。
func (t *txManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx) // nested: 外側の tx をそのまま使い 1 回だけ実行する（ADR-0035）
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
	if xerrors.As(err, &pgErr) || pgerror.IsUnavailable(err) {
		return pgerror.NormalizeError(err)
	}
	if xerrors.Is(err, context.Canceled) || xerrors.Is(err, context.DeadlineExceeded) {
		return pgerror.NormalizeError(err)
	}
	return err
}

// doOnce は、1 回分のトランザクション（begin → fn → commit / rollback）を実行します。
//
// エラーは正規化せず生のまま返します。IsRetryableTxError が生 SQLSTATE を errors.As で
// 参照できる必要があるためで、apperror への写像は呼出元 Do がリトライ後に 1 度だけ行います
// （chain に PgError が残る理由は ADR-0035）。
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
			ctx, "Failed to rollback transaction", logFields...,
		)
	}
}

func withTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}
