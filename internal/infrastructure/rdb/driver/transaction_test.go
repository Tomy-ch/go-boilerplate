package driver

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/system"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"
)

// stubTx は、Commit / Rollback の結果と呼び出し回数を観測する pgx.Tx のテストダブルです。
// 埋め込みは nil のため、テストが想定しないメソッドが呼ばれれば panic で顕在化します。
type stubTx struct {
	pgx.Tx

	commitErr   error
	rollbackErr error

	commits   int
	rollbacks int
	// rollbackCtxErr は、Rollback 呼び出し時点で渡された ctx が生きていたかを表します
	// （呼び出し元は defer cancel するため、戻ってから ctx を見ても常にキャンセル済みになる）。
	rollbackCtxErr error
	rollbackDone   chan struct{}
}

// stubDriver は、Begin の戻り値を注入する DatabaseDriver のテストダブルです。
// 生成 mock は同じ mock パッケージ内の mock_query_metric.go.gen.go が driver を import するため、
// 内部テストパッケージからは import cycle になり使えません（外部テストパッケージの
// transaction_retry_test.go は生成 mock を使えますが、doOnce / rollback は非公開のため到達できません）。
type stubDriver struct {
	DatabaseDriver

	tx       pgx.Tx
	beginErr error
}

func (s *stubTx) Commit(context.Context) error {
	s.commits++
	return s.commitErr
}

func (s *stubTx) Rollback(ctx context.Context) error {
	s.rollbacks++
	s.rollbackCtxErr = ctx.Err()
	if s.rollbackDone != nil {
		close(s.rollbackDone)
	}
	return s.rollbackErr
}

func (s *stubDriver) Begin(context.Context) (pgx.Tx, error) { return s.tx, s.beginErr }

func TestNewTransactionManager(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)
	testLogger := logging.NewTestLogger(t)

	db, err := NewDB(dbCfg, osCfg, dbConnCfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定値からリトライ上限とbackoffが構成される", func(t *testing.T) {
			t.Parallel()

			manager := NewTransactionManager(db, dbCfg, testLogger, system.NewSleeper())
			require.NotNil(t, manager)

			txm, ok := manager.(*txManager)
			require.True(t, ok)
			assert.Equal(t, dbCfg.TxMaxRetries(), txm.maxAttempts)
			assert.Equal(t, dbCfg.TxRetryBaseBackoff(), txm.backoff.Initial)
			assert.Equal(t, dbCfg.TxRetryMaxBackoff(), txm.backoff.Max)
		})

		t.Run("設定値が0以下の場合は既定値へフォールバックする", func(t *testing.T) {
			t.Parallel()

			// ゼロ値の DatabaseConfig は各リトライ設定が 0 のため、既定値へフォールバックする。
			manager := NewTransactionManager(db, &config.DatabaseConfig{}, testLogger, system.NewSleeper())
			require.NotNil(t, manager)

			txm, ok := manager.(*txManager)
			require.True(t, ok)
			assert.Equal(t, defaultTxMaxAttempts, txm.maxAttempts)
			assert.Equal(t, defaultTxBackoffInitial, txm.backoff.Initial)
			assert.Equal(t, defaultTxBackoffMax, txm.backoff.Max)
		})
	})
}

func Test_normalizeTxResult(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilはnilを返す", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, normalizeTxResult(nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("PgErrorはapperrorへ正規化される", func(t *testing.T) {
			t.Parallel()
			got := normalizeTxResult(&pgconn.PgError{Code: "23505"})
			require.ErrorIs(t, got, apperror.ErrConflict)
		})

		t.Run("接続不可エラー(08xxx)はapperrorへ正規化される", func(t *testing.T) {
			t.Parallel()
			got := normalizeTxResult(&pgconn.PgError{Code: "08006"})
			require.ErrorIs(t, got, apperror.ErrUnavailable)
		})

		t.Run("context.Canceledはapperrorへ正規化される", func(t *testing.T) {
			t.Parallel()
			got := normalizeTxResult(context.Canceled)
			require.ErrorIs(t, got, apperror.ErrCanceled)
		})

		t.Run("context.DeadlineExceededはapperrorへ正規化される", func(t *testing.T) {
			t.Parallel()
			got := normalizeTxResult(context.DeadlineExceeded)
			require.ErrorIs(t, got, apperror.ErrUnavailable)
		})

		t.Run("fnが返した非DBエラーは正規化せず素通しする", func(t *testing.T) {
			t.Parallel()
			appErr := xerrors.Wrap(apperror.ErrValidation, "boom")
			got := normalizeTxResult(appErr)
			require.ErrorIs(t, got, apperror.ErrValidation)
		})
	})
}

func Test_txManager_doOnce(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("fn が成功した場合はコミットし tx を ctx へ載せる", func(t *testing.T) {
			t.Parallel()

			tx := &stubTx{}
			m := &txManager{db: &stubDriver{tx: tx}, logger: logging.NewTestLogger(t)}

			var seen bool
			err := m.doOnce(context.Background(), func(ctx context.Context) error {
				_, seen = ctx.Value(txKey{}).(pgx.Tx)
				return nil
			})

			require.NoError(t, err)
			assert.True(t, seen)
			assert.Equal(t, 1, tx.commits)
			assert.Equal(t, 0, tx.rollbacks)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Begin が失敗した場合は fn を実行せず生のエラーを返す", func(t *testing.T) {
			t.Parallel()

			beginErr := &pgconn.PgError{Code: "40001"}
			m := &txManager{db: &stubDriver{beginErr: beginErr}, logger: logging.NewTestLogger(t)}

			called := false
			err := m.doOnce(context.Background(), func(context.Context) error {
				called = true
				return nil
			})

			// リトライ判定が生 SQLSTATE を参照できるよう、この層では正規化しない。
			require.ErrorIs(t, err, beginErr)
			assert.False(t, called)
		})

		t.Run("fn が失敗した場合はロールバックしコミットせずそのエラーを返す", func(t *testing.T) {
			t.Parallel()

			tx := &stubTx{}
			m := &txManager{db: &stubDriver{tx: tx}, logger: logging.NewTestLogger(t)}
			fnErr := xerrors.New("fn failed")

			err := m.doOnce(context.Background(), func(context.Context) error { return fnErr })

			require.ErrorIs(t, err, fnErr)
			assert.Equal(t, 1, tx.rollbacks)
			assert.Equal(t, 0, tx.commits)
		})

		t.Run("Commit が失敗した場合はロールバックせず生のエラーを返す", func(t *testing.T) {
			t.Parallel()

			commitErr := &pgconn.PgError{Code: "40P01"}
			tx := &stubTx{commitErr: commitErr}
			m := &txManager{db: &stubDriver{tx: tx}, logger: logging.NewTestLogger(t)}

			err := m.doOnce(context.Background(), func(context.Context) error { return nil })

			require.ErrorIs(t, err, commitErr)
			assert.Equal(t, 0, tx.rollbacks)
		})

		t.Run("fn が panic した場合はロールバックして panic を再送出する", func(t *testing.T) {
			t.Parallel()

			tx := &stubTx{}
			m := &txManager{db: &stubDriver{tx: tx}, logger: logging.NewTestLogger(t)}

			assert.PanicsWithValue(t, "boom", func() {
				_ = m.doOnce(context.Background(), func(context.Context) error { panic("boom") })
			})
			assert.Equal(t, 1, tx.rollbacks)
			assert.Equal(t, 0, tx.commits)
		})

		t.Run("fn が runtime.Goexit で中断した場合もロールバックする", func(t *testing.T) {
			t.Parallel()

			tx := &stubTx{rollbackDone: make(chan struct{})}
			m := &txManager{db: &stubDriver{tx: tx}, logger: logging.NewTestLogger(t)}

			// Goexit は呼び出し goroutine を終了させるため、別 goroutine で実行して後始末だけを観測する。
			go func() {
				_ = m.doOnce(context.Background(), func(context.Context) error {
					runtime.Goexit()
					return nil
				})
			}()

			select {
			case <-tx.rollbackDone:
			case <-time.After(2 * time.Second):
				t.Fatal("Goexit 中断時にロールバックされなかった")
			}
			assert.Equal(t, 0, tx.commits)
		})
	})
}

func Test_txManager_rollback(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ロールバックが成功した場合はログを出さない", func(t *testing.T) {
			t.Parallel()

			logger, observed := logging.NewObservedTestLogger(t)
			tx := &stubTx{}
			m := &txManager{logger: logger}

			m.rollback(context.Background(), tx)

			assert.Equal(t, 1, tx.rollbacks)
			assert.Zero(t, observed.Len())
		})

		t.Run("親 ctx がキャンセル済みでも生きた ctx でロールバックする", func(t *testing.T) {
			t.Parallel()

			tx := &stubTx{}
			m := &txManager{logger: logging.NewTestLogger(t)}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			m.rollback(ctx, tx)

			require.Equal(t, 1, tx.rollbacks)
			assert.NoError(t, tx.rollbackCtxErr) // 後始末は親のキャンセルに巻き込まれない
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ロールバックが失敗した場合は追加フィールドを併記してエラーログを残す", func(t *testing.T) {
			t.Parallel()

			logger, observed := logging.NewObservedTestLogger(t)
			tx := &stubTx{rollbackErr: xerrors.New("rollback failed")}
			m := &txManager{logger: logger}

			m.rollback(context.Background(), tx, logging.String("phase", "commit"))

			entries := observed.FilterMessage("Failed to rollback transaction").All()
			require.Len(t, entries, 1)
			assert.Equal(t, "error", entries[0].Level.String())
			ctxMap := entries[0].ContextMap()
			assert.Contains(t, ctxMap, string(logging.ErrorKey))
			assert.Equal(t, "commit", ctxMap["phase"])
		})
	})
}

func Test_withTx(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
