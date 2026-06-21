package idempotency

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	idempotencybndry "go-boilerplate/internal/usecase/boundary/idempotency"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newFingerprint(b byte) []byte {
	fp := make([]byte, 32)
	for i := range fp {
		fp[i] = b
	}
	return fp
}

func Test_isLockNotAvailable(t *testing.T) {
	t.Parallel()

	t.Run("55P03(lock_not_available)はtrue", func(t *testing.T) {
		t.Parallel()
		assert.True(t, isLockNotAvailable(&pgconn.PgError{Code: pgLockNotAvailable}))
	})

	t.Run("別のSQLSTATEはfalse", func(t *testing.T) {
		t.Parallel()
		assert.False(t, isLockNotAvailable(&pgconn.PgError{Code: "23505"}))
	})

	t.Run("PgError以外はfalse", func(t *testing.T) {
		t.Parallel()
		assert.False(t, isLockNotAvailable(errors.New("plain error")))
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	loggingDB := testkit.NewTestLoggingProvider(t)
	tf := observability.NewNoopTracerFactory(t)
	expected := &store{
		tracer: tf.Infra(),
		db:     loggingDB,
	}

	assert.Equal(t, expected, New(loggingDB, tf))
}

func Test_store_lifecycle(t *testing.T) {
	t.Parallel()

	loggingDB := testkit.NewTestLoggingProvider(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	s := &store{tracer: lt, db: loggingDB}

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

	loggingDB := testkit.NewTestLoggingProvider(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	s := &store{tracer: lt, db: loggingDB}

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

	loggingDB := testkit.NewTestLoggingProvider(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	s := &store{tracer: lt, db: loggingDB}

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないキーへのcompleteはエラーになる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				err := s.Complete(ctx, idempotencybndry.CompleteParams{
					Scope: "missing-scope", Key: "missing-key",
					ResponseStatus: 201, ResponsePayload: []byte(`{}`),
				})
				require.Error(t, err)
			})
		})

		t.Run("キャンセル済みコンテキストでは各操作がエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			params := idempotencybndry.ClaimParams{
				Scope: "ctx-cancel", Key: "key-1", Method: "POST", Path: "/v1/users",
				Fingerprint: newFingerprint(0x03), ExpiresAt: time.Now().Add(time.Hour),
			}
			_, claimErr := s.Claim(ctx, params)
			require.Error(t, claimErr)

			_, getErr := s.Get(ctx, "ctx-cancel", "key-1")
			require.Error(t, getErr)

			completeErr := s.Complete(ctx, idempotencybndry.CompleteParams{
				Scope: "ctx-cancel", Key: "key-1", ResponseStatus: 201, ResponsePayload: []byte(`{}`),
			})
			require.Error(t, completeErr)

			_, delErr := s.DeleteExpired(ctx, time.Now(), 100)
			require.Error(t, delErr)
		})
	})
}
