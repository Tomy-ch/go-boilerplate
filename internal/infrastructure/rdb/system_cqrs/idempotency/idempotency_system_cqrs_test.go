package idempotency

import (
	"context"
	"sync"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
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

// errSubjectRollback は、lock_timeout 復元テストで業務側 tx を行を残さず終えるための sentinel です。
var errSubjectRollback = xerrors.New("rollback subject tx for test")

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
				// DeleteExpired は scope 非限定でテーブル全体を対象とするため、共有DB上では他の失効行も
				// 含まれ得る。厳密件数に依存せず「1件以上削除された」ことのみ検証し、対象行の削除は
				// 後続の Get==nil で担保する。
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

func Test_store_Claim_lockTimeoutScope(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	s := &store{tracer: lt, db: testDB}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("claim後の業務クエリは復元済みlock_timeoutで動き3s超のロック待ちでも55P03にならない", func(t *testing.T) {
			t.Parallel()

			// claim 時の SET LOCAL lock_timeout(3s) が業務フェーズへ波及していないことを、
			// 保持側 tx の advisory lock を claim の 3s 上限を超えて待たせても 55P03 にならない、
			// という振る舞いで検証する。WithinTx は txLock で直列化するため使わず、同一
			// TransactionManager の Do を 2 本並行させる（Do ごとに別コネクションの tx を張る）。
			claimTimeout, err := time.ParseDuration(claimLockTimeout)
			require.NoError(t, err)

			dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
			txm := driver.NewTransactionManager(testDB, dbCfg, logging.NewTestLogger(t), system.NewSleeper())

			const advisoryLockKey = int64(732100000000000091)

			holderLocked := make(chan struct{})
			releaseHolder := make(chan struct{})
			holderDone := make(chan error, 1)

			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(releaseHolder) }) }
			t.Cleanup(release) // 失敗時も保持側 goroutine をリークさせない

			// 保持側 tx: advisory xact lock を取得し、claim の lock_timeout(3s)を超えて保持する。
			go func() {
				holderDone <- txm.Do(context.Background(), func(ctx context.Context) error {
					if _, err := driver.New(ctx, testDB).Exec(ctx, "SELECT pg_advisory_xact_lock($1)", advisoryLockKey); err != nil {
						return err
					}
					close(holderLocked)
					<-releaseHolder
					return errHolderRollback // rollback で advisory lock を解放する。
				})
			}()

			select {
			case <-holderLocked:
			case err := <-holderDone:
				require.NoError(t, err, "保持側 tx がロック取得前に失敗した")
				return
			}

			// 業務側 tx: claim（ここで lock_timeout=3s がセットされ直後に既定へ戻る）後、
			// 保持側が握る advisory lock を待つ。ロック待ちがブロックするため goroutine で実行する。
			subjectDone := make(chan struct{})
			var subjectTxErr, bizPhaseErr error
			go func() {
				defer close(subjectDone)
				subjectTxErr = txm.Do(context.Background(), func(ctx context.Context) error {
					claimed, err := s.Claim(ctx, idempotencybndry.ClaimParams{
						Scope: "lock-timeout-restore-scope", Key: "key-1", Method: "POST", Path: "/v1/users",
						Fingerprint: newFingerprint(0x06), ExpiresAt: time.Now().Add(time.Hour),
					})
					if err != nil {
						return err
					}
					if !claimed {
						return xerrors.New("subject: expected fresh claim to succeed")
					}
					// 業務フェーズのロック待ち。lock_timeout が claim に閉じていれば 55P03 にならない。
					_, bizPhaseErr = driver.New(ctx, testDB).Exec(ctx, "SELECT pg_advisory_xact_lock($1)", advisoryLockKey)
					return errSubjectRollback // 業務クエリの成否は bizPhaseErr で見て、行は残さない。
				})
			}()

			// 業務側が実際に advisory lock 待ちへ入るまで待つ。goroutine 起動直後に保持タイマーを
			// 始めると、CI 遅延でロック待ち突入前に解放してしまい偽陽性で通る窓があるため。
			waitForAdvisoryLockWait(t, testDB, advisoryLockKey)

			// ロック待ちを確認できたので、claim の 3s 上限を確実に超える時間だけ保持してから解放する。
			// lock_timeout が未復元なら業務側は待機中に 3s で 55P03 になっている。
			time.Sleep(claimTimeout + time.Second)
			release()

			<-subjectDone
			require.NoError(t, bizPhaseErr)
			require.False(t, pgerror.IsLockNotAvailable(bizPhaseErr))
			require.ErrorIs(t, subjectTxErr, errSubjectRollback)
			require.ErrorIs(t, <-holderDone, errHolderRollback)
		})
	})
}

// waitForAdvisoryLockWait は、指定キーの advisory lock 待ち（granted=false）が pg_locks に
// 現れるまで待ちます。bigint 版 advisory lock は classid=上位32bit / objid=下位32bit /
// objsubid=1 に分解されるため、キーを再構成して照合します。
func waitForAdvisoryLockWait(t *testing.T, db driver.DatabaseDriver, key int64) {
	t.Helper()
	require.Eventually(t, func() bool {
		var waiting bool
		if err := driver.New(context.Background(), db).QueryRow(context.Background(),
			`SELECT EXISTS (SELECT 1 FROM pg_locks WHERE locktype = 'advisory' AND NOT granted
				AND (classid::bigint << 32) + objid::bigint = $1 AND objsubid = 1)`,
			key).Scan(&waiting); err != nil {
			return false
		}
		return waiting
	}, 10*time.Second, 50*time.Millisecond, "advisory lock 待ちに入らなかった")
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

		t.Run("claim挿入が汎用DBエラーで失敗した場合はErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				// NUL バイトを含む scope は PostgreSQL の TEXT に格納できず、ロック競合でも既存行でもない
				// 汎用エラーとなるため、Claim の default 分岐（ErrInternal 正規化）へ落ちる。
				params := idempotencybndry.ClaimParams{
					Scope: "bad\x00scope", Key: "key-1", Method: "POST", Path: "/v1/users",
					Fingerprint: newFingerprint(0x05), ExpiresAt: time.Now().Add(time.Hour),
				}
				_, err := s.Claim(ctx, params)
				require.ErrorIs(t, err, apperror.ErrInternal)
			})
		})

		t.Run("キャンセル済みコンテキストでは各操作がErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			params := idempotencybndry.ClaimParams{
				Scope: "ctx-cancel", Key: "key-1", Method: "POST", Path: "/v1/users",
				Fingerprint: newFingerprint(0x03), ExpiresAt: time.Now().Add(time.Hour),
			}
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
