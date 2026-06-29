package idempotency

import (
	"context"
	"sync"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/infrastructure/system"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	idempotencybndry "go-boilerplate/internal/usecase/boundary/idempotency"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errHolderRollback は、ロック競合テストで保持側 tx を最後にロールバックさせるための sentinel です。
var errHolderRollback = xerrors.New("rollback holder tx for test")

func newFingerprint(b byte) []byte {
	fp := make([]byte, 32)
	for i := range fp {
		fp[i] = b
	}
	return fp
}

func TestNew(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	tf := observability.NewNoopTracerFactory(t)
	expected := &store{
		tracer: tf.Infra(),
		db:     testDB,
	}

	assert.Equal(t, expected, New(testDB, tf))
}

func Test_store_lifecycle(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	s := &store{tracer: lt, db: testDB}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("claim→get(claimed)→complete→get(completed) の一連が成立する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				const scope, key = "test-scope-lifecycle", "key-1"
				fp := newFingerprint(0xAB)

				// 未claimは nil, nil。
				before, err := s.Get(ctx, scope, key)
				require.NoError(t, err)
				assert.Nil(t, before)

				// 新規claim → true。
				claimed, err := s.Claim(ctx, idempotencybndry.ClaimParams{
					Scope: scope, Key: key, Method: "POST", Path: "/v1/users",
					Fingerprint: fp, ExpiresAt: time.Now().Add(24 * time.Hour),
				})
				require.NoError(t, err)
				assert.True(t, claimed)

				// claim直後は claimed 状態・結果未保存。
				rec, err := s.Get(ctx, scope, key)
				require.NoError(t, err)
				require.NotNil(t, rec)
				assert.Equal(t, idempotencybndry.StatusClaimed, rec.Status)
				assert.Nil(t, rec.ResponseStatus)
				assert.Equal(t, fp, rec.Fingerprint)

				// 同一(scope,key)の再claimは false(一意制約で衝突)。
				again, err := s.Claim(ctx, idempotencybndry.ClaimParams{
					Scope: scope, Key: key, Method: "POST", Path: "/v1/users",
					Fingerprint: fp, ExpiresAt: time.Now().Add(24 * time.Hour),
				})
				require.NoError(t, err)
				assert.False(t, again)

				// complete で結果保存。
				payload := []byte(`{"v":1}`)
				require.NoError(t, s.Complete(ctx, idempotencybndry.CompleteParams{
					Scope: scope, Key: key, ResponseStatus: 201, ResponsePayload: payload,
				}))

				// completed 状態・結果が読める。
				done, err := s.Get(ctx, scope, key)
				require.NoError(t, err)
				require.NotNil(t, done)
				assert.Equal(t, idempotencybndry.StatusCompleted, done.Status)
				require.NotNil(t, done.ResponseStatus)
				assert.Equal(t, int32(201), *done.ResponseStatus)
				assert.Equal(t, payload, done.ResponsePayload)
			})
		})
	})
}

func Test_store_DeleteExpired(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	s := &store{tracer: lt, db: testDB}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("cutoffより古い失効行を削除する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				const scope, key = "test-scope-gc", "key-1"

				// expires_at を過去にして claim（= 失効済み）。
				claimed, err := s.Claim(ctx, idempotencybndry.ClaimParams{
					Scope: scope, Key: key, Method: "POST", Path: "/v1/users",
					Fingerprint: newFingerprint(0x01), ExpiresAt: time.Now().Add(-1 * time.Hour),
				})
				require.NoError(t, err)
				require.True(t, claimed)

				// cutoff=now で失効行が削除される。
				deleted, err := s.DeleteExpired(ctx, time.Now(), 100)
				require.NoError(t, err)
				assert.GreaterOrEqual(t, deleted, int64(1))

				// 削除後は取得できない。
				rec, err := s.Get(ctx, scope, key)
				require.NoError(t, err)
				assert.Nil(t, rec)
			})
		})

		t.Run("失効行が無ければ0件", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				const scope, key = "test-scope-gc-fresh", "key-1"

				// 未失効(将来失効)で claim。
				claimed, err := s.Claim(ctx, idempotencybndry.ClaimParams{
					Scope: scope, Key: key, Method: "POST", Path: "/v1/users",
					Fingerprint: newFingerprint(0x02), ExpiresAt: time.Now().Add(24 * time.Hour),
				})
				require.NoError(t, err)
				require.True(t, claimed)

				// cutoff を過去に置けば、この行は対象外。
				deleted, err := s.DeleteExpired(ctx, time.Now().Add(-48*time.Hour), 100)
				require.NoError(t, err)
				assert.Equal(t, int64(0), deleted)
			})
		})
	})
}

func Test_store_errors(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	s := &store{tracer: lt, db: testDB}

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないキーへのcompleteはエラーになる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				err := s.Complete(ctx, idempotencybndry.CompleteParams{
					Scope: "missing-scope", Key: "missing-key",
					ResponseStatus: 201, ResponsePayload: []byte(`{}`),
				})
				require.ErrorIs(t, err, apperror.ErrInternal)
			})
		})

		t.Run("ロック競合タイムアウト時はErrLockTimeoutを返す", func(t *testing.T) {
			t.Parallel()

			// Claim の 55P03(lock_not_available) 分岐は、同一 (scope,key) を SET LOCAL
			// lock_timeout 付きで claim する 2 本の業務 tx が並行に競合して初めて発生する。
			// testkit の WithinTx は txLock で直列化するため再現できないので、ここでは
			// TransactionManager を 2 本（= 2 コネクション）並行で走らせて再現する。
			dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
			txm := driver.NewTransactionManager(testDB, dbCfg, logging.NewTestLogger(t), system.NewSleeper())

			params := idempotencybndry.ClaimParams{
				Scope: "lock-timeout-scope", Key: "key-1", Method: "POST", Path: "/v1/users",
				Fingerprint: newFingerprint(0x04), ExpiresAt: time.Now().Add(time.Hour),
			}

			holderClaimed := make(chan struct{})
			releaseHolder := make(chan struct{})
			holderDone := make(chan error, 1)

			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(releaseHolder) }) }
			t.Cleanup(release) // 失敗時も保持側 goroutine をリークさせない

			// 保持側 tx: 同一キーを claim し、競合側がタイムアウトするまで未コミットで保持する。
			go func() {
				holderDone <- txm.Do(context.Background(), func(ctx context.Context) error {
					claimed, err := s.Claim(ctx, params)
					if err != nil {
						return err
					}
					if !claimed {
						return xerrors.New("holder: expected fresh claim to succeed")
					}
					close(holderClaimed)
					<-releaseHolder
					// 行を残さないようエラー返却で rollback させる。
					return errHolderRollback
				})
			}()

			// 保持側が claim 済み（行ロック確立）になるまで待つ。失敗時は holderDone で検知。
			select {
			case <-holderClaimed:
			case err := <-holderDone:
				require.NoError(t, err, "保持側 tx が claim 前に失敗した")
				return
			}

			// 競合側 tx: 同一キーの claim はロック待ち→ lock_timeout で 55P03 → ErrLockTimeout。
			contenderErr := txm.Do(context.Background(), func(ctx context.Context) error {
				_, err := s.Claim(ctx, params)
				return err
			})
			require.ErrorIs(t, contenderErr, idempotencybndry.ErrLockTimeout)

			release()
			require.ErrorIs(t, <-holderDone, errHolderRollback)
		})

		t.Run("キャンセル済みコンテキストでは各操作がErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			params := idempotencybndry.ClaimParams{
				Scope: "ctx-cancel", Key: "key-1", Method: "POST", Path: "/v1/users",
				Fingerprint: newFingerprint(0x03), ExpiresAt: time.Now().Add(time.Hour),
			}
			// context.Canceled は pgerror.NormalizeError で apperror.ErrCanceled へ写像される。
			_, claimErr := s.Claim(ctx, params)
			require.ErrorIs(t, claimErr, apperror.ErrCanceled)

			_, getErr := s.Get(ctx, "ctx-cancel", "key-1")
			require.ErrorIs(t, getErr, apperror.ErrCanceled)

			completeErr := s.Complete(ctx, idempotencybndry.CompleteParams{
				Scope: "ctx-cancel", Key: "key-1", ResponseStatus: 201, ResponsePayload: []byte(`{}`),
			})
			require.ErrorIs(t, completeErr, apperror.ErrCanceled)

			_, delErr := s.DeleteExpired(ctx, time.Now(), 100)
			require.ErrorIs(t, delErr, apperror.ErrCanceled)
		})
	})
}
