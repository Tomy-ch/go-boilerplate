package outbox

import (
	"context"
	"fmt"
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
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errRollbackForTest は、SKIP LOCKED 並行テストで保持側 tx を行を残さず終えるための sentinel です。
var errRollbackForTest = xerrors.New("rollback tx for test")

func emitParams() outboxbndry.EmitParams {
	return outboxbndry.EmitParams{
		AggregateType: "Purchase",
		AggregateID:   "p-1",
		EventType:     "purchase.created.v1",
		Payload:       []byte(`{"v":1}`),
	}
}

// newTestStore は、共有テストDB上の store と tx 直列化ランナーを組み立てるテストヘルパーです。
func newTestStore(t *testing.T) (*store, testkit.TransactionRunner) {
	t.Helper()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	return &store{tracer: lt, db: testDB}, txm
}

// canceledContext は、キャンセル済みの context を返すテストヘルパーです。
func canceledContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
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

func Test_store_Insert(t *testing.T) {
	t.Parallel()

	s, txm := newTestStore(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非nilのmessage_idを返しpending行を作成する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				msgID, err := s.Insert(ctx, emitParams())
				require.NoError(t, err)
				assert.False(t, msgID.IsNil())

				// 挿入行は pending として claim できる。
				msgs, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				require.Len(t, msgs, 1)
				assert.Equal(t, msgID, msgs[0].MessageID)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			_, err := s.Insert(canceledContext(t), emitParams())
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

func Test_store_ClaimPending(t *testing.T) {
	t.Parallel()

	s, txm := newTestStore(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("pending行を挿入内容とともに返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				msgID, err := s.Insert(ctx, emitParams())
				require.NoError(t, err)

				msgs, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				require.Len(t, msgs, 1)
				assert.Equal(t, msgID, msgs[0].MessageID)
				assert.Equal(t, "purchase.created.v1", msgs[0].EventType)
				assert.JSONEq(t, `{"v":1}`, string(msgs[0].Payload))
				assert.Equal(t, int32(0), msgs[0].Attempts)
			})
		})

		t.Run("limitを超えるpending行があってもlimit件までしか返さない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				for range 3 {
					_, err := s.Insert(ctx, emitParams())
					require.NoError(t, err)
				}

				msgs, err := s.ClaimPending(ctx, 2)
				require.NoError(t, err)
				require.Len(t, msgs, 2)
			})
		})

		t.Run("pending行が無ければ空を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				msgs, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				assert.Empty(t, msgs)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			_, err := s.ClaimPending(canceledContext(t), 10)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

// Test_store_ClaimPending_concurrentSkipLocked は、ADR-0052 の核である多インスタンス排他
// （FOR UPDATE SKIP LOCKED により別々の並行 tx が同一 pending 行を二重 claim しない）を、
// 2 コネクション（= 2 tx）を並行させて DB レベルで検証します。
//
// testkit の WithinTx は txLock で全 tx を直列化するため、この並行排他は構造的に再現できません。
// そこで idempotency 側の並行テストと同様に driver.NewTransactionManager を直接使い 2 tx を並行走行させます。
// SKIP LOCKED は両 tx から見える committed 行を必要とするため、本テストのみ非並列にし
// （他テストの「pending が空」アサートと committed 行が衝突しないよう sequential フェーズで完結させる）、
// 専用 aggregate_id の行を投入・後始末します。
//
//nolint:paralleltest // committed fixture が並列テストの全体空アサートと衝突するため非並列
func Test_store_ClaimPending_concurrentSkipLocked(t *testing.T) {
	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	s := &store{tracer: lt, db: testDB}

	dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
	txm := driver.NewTransactionManager(testDB, dbCfg, logging.NewTestLogger(t), system.NewSleeper())

	const aggregateID = "skiplocked-concurrency-test"

	// 後始末: 本テストが commit した専用行を必ず削除する（sequential フェーズ内で完結させる）。
	t.Cleanup(func() {
		_ = txm.Do(context.Background(), func(ctx context.Context) error {
			_, err := driver.New(ctx, testDB).Exec(ctx, "DELETE FROM outbox WHERE aggregate_id = $1", aggregateID)
			return err
		})
	})

	// 2 件の pending 行を commit し、両 tx から見えるようにする。
	params := emitParams()
	params.AggregateID = aggregateID
	require.NoError(t, txm.Do(context.Background(), func(ctx context.Context) error {
		for range 2 {
			if _, err := s.Insert(ctx, params); err != nil {
				return err
			}
		}
		return nil
	}))

	// 全 skip: 保持側が 2 行すべてをロックする間、競合側は 0 行（すべて skip される）。
	assertSkipLockedContention(t, txm.Do, s, 10, 2, 0)

	// release 後は行が pending へ戻り、新規 tx から再 claim できる（ロック解放後は取得可能）。
	// SELECT のみのため rollback で pending のまま残し、次の部分 skip フェーズで再利用する。
	reclaimErr := txm.Do(context.Background(), func(ctx context.Context) error {
		claimed, err := s.ClaimPending(ctx, 10)
		if err != nil {
			return err
		}
		assert.Len(t, claimed, 2, "release 後は保持されていた行を再 claim できる")
		return errRollbackForTest
	})
	require.ErrorIs(t, reclaimErr, errRollbackForTest)

	// 部分 skip: 保持側が 1 行だけロックし、競合側は残り 1 行を取得できる（ロック済のみ skip）。
	assertSkipLockedContention(t, txm.Do, s, 1, 1, 1)
}

// assertSkipLockedContention は、保持側 tx が holderLimit 件をロックして保持する間に競合側 tx を走らせ、
// 保持側が wantHolder 件・競合側が wantContender 件を claim することを検証します（SKIP LOCKED の排他/部分 skip）。
// 保持側は行を残さないよう最後に rollback します。do は TransactionManager.Do を渡します。
func assertSkipLockedContention(
	t *testing.T,
	do func(context.Context, func(context.Context) error) error,
	s *store,
	holderLimit int32,
	wantHolder, wantContender int,
) {
	t.Helper()

	holderClaimed := make(chan struct{})
	release := make(chan struct{})
	holderDone := make(chan error, 1)

	var once sync.Once
	rel := func() { once.Do(func() { close(release) }) }
	t.Cleanup(rel) // 失敗時も保持側 goroutine をリークさせない

	go func() {
		holderDone <- do(context.Background(), func(ctx context.Context) error {
			claimed, err := s.ClaimPending(ctx, holderLimit)
			if err != nil {
				return err
			}
			if len(claimed) != wantHolder {
				return xerrors.New(fmt.Sprintf("holder: expected to claim %d rows, got %d", wantHolder, len(claimed)))
			}
			close(holderClaimed)
			<-release
			return errRollbackForTest
		})
	}()

	select {
	case <-holderClaimed:
	case err := <-holderDone:
		require.NoError(t, err, "保持側 tx が claim 前に失敗した")
		return
	}

	contenderErr := do(context.Background(), func(ctx context.Context) error {
		claimed, err := s.ClaimPending(ctx, 10)
		if err != nil {
			return err
		}
		assert.Len(t, claimed, wantContender, "SKIP LOCKED: ロック済行は skip し、未ロック行のみ取得する")
		return nil
	})
	require.NoError(t, contenderErr)

	rel()
	require.ErrorIs(t, <-holderDone, errRollbackForTest)
}

func Test_store_MarkPublished(t *testing.T) {
	t.Parallel()

	s, txm := newTestStore(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("publishedへ遷移し再claimでは返らない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				_, err := s.Insert(ctx, emitParams())
				require.NoError(t, err)
				msgs, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				require.Len(t, msgs, 1)

				require.NoError(t, s.MarkPublished(ctx, msgs[0].ID))

				after, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				assert.Empty(t, after)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, s.MarkPublished(canceledContext(t), 1), apperror.ErrCanceled)
		})
	})
}

func Test_store_MarkFailed(t *testing.T) {
	t.Parallel()

	s, txm := newTestStore(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("attemptsを加算し加算後の値を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				_, err := s.Insert(ctx, emitParams())
				require.NoError(t, err)
				msgs, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				require.Len(t, msgs, 1)

				attempts, err := s.MarkFailed(ctx, msgs[0].ID, "boom")
				require.NoError(t, err)
				assert.Equal(t, int32(1), attempts)
			})
		})

		t.Run("pendingでないidに対しては0を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				attempts, err := s.MarkFailed(ctx, 1<<60, "boom")
				require.NoError(t, err)
				assert.Equal(t, int32(0), attempts)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			_, err := s.MarkFailed(canceledContext(t), 1, "boom")
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

func Test_store_MarkDead(t *testing.T) {
	t.Parallel()

	s, txm := newTestStore(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("deadへ遷移しclaimされなくなる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				_, err := s.Insert(ctx, emitParams())
				require.NoError(t, err)
				msgs, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				require.Len(t, msgs, 1)

				require.NoError(t, s.MarkDead(ctx, msgs[0].ID))

				none, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				assert.Empty(t, none)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, s.MarkDead(canceledContext(t), 1), apperror.ErrCanceled)
		})
	})
}

func Test_store_ReplayDead(t *testing.T) {
	t.Parallel()

	s, txm := newTestStore(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("dead行がpendingへ戻り再びclaimされる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				_, err := s.Insert(ctx, emitParams())
				require.NoError(t, err)
				msgs, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				require.Len(t, msgs, 1)
				require.NoError(t, s.MarkDead(ctx, msgs[0].ID))

				replayed, err := s.ReplayDead(ctx, nil)
				require.NoError(t, err)
				assert.GreaterOrEqual(t, replayed, int64(1))

				back, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				assert.Len(t, back, 1)
			})
		})

		t.Run("message_idを指定すると当該1件のみpendingへ戻る", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				// 2 行 insert し、両方を dead にする。
				targetMsgID, err := s.Insert(ctx, emitParams())
				require.NoError(t, err)
				otherMsgID, err := s.Insert(ctx, emitParams())
				require.NoError(t, err)

				claimed, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				require.Len(t, claimed, 2)
				for _, m := range claimed {
					require.NoError(t, s.MarkDead(ctx, m.ID))
				}

				// 指定 message_id の 1 件のみが戻る。
				replayed, err := s.ReplayDead(ctx, &targetMsgID)
				require.NoError(t, err)
				assert.Equal(t, int64(1), replayed)

				// 戻ったのは指定行のみ（再 claim で targetMsgID だけが返る）。
				back, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				require.Len(t, back, 1)
				assert.Equal(t, targetMsgID, back[0].MessageID)
				assert.NotEqual(t, otherMsgID, back[0].MessageID)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			_, err := s.ReplayDead(canceledContext(t), nil)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

func Test_store_DeletePublished(t *testing.T) {
	t.Parallel()

	s, txm := newTestStore(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("cutoffより古いpublished行を削除する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				_, err := s.Insert(ctx, emitParams())
				require.NoError(t, err)
				msgs, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				require.Len(t, msgs, 1)
				require.NoError(t, s.MarkPublished(ctx, msgs[0].ID))

				deleted, err := s.DeletePublished(ctx, time.Now().Add(time.Hour), 100)
				require.NoError(t, err)
				assert.GreaterOrEqual(t, deleted, int64(1))
			})
		})

		t.Run("cutoffより新しいpublished行が残る場合は0件", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				_, err := s.Insert(ctx, emitParams())
				require.NoError(t, err)
				msgs, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				require.Len(t, msgs, 1)
				require.NoError(t, s.MarkPublished(ctx, msgs[0].ID))

				// cutoff(1時間前)より後に published にした行は削除対象外なので 0 件。
				deleted, err := s.DeletePublished(ctx, time.Now().Add(-time.Hour), 100)
				require.NoError(t, err)
				assert.Equal(t, int64(0), deleted)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			_, err := s.DeletePublished(canceledContext(t), time.Now(), 1)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

func Test_store_OldestPendingCreatedAt(t *testing.T) {
	t.Parallel()

	s, txm := newTestStore(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("pending有無でokを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				// 空状態では ok=false。
				_, ok, err := s.OldestPendingCreatedAt(ctx)
				require.NoError(t, err)
				assert.False(t, ok)

				_, err = s.Insert(ctx, emitParams())
				require.NoError(t, err)

				createdAt, ok, err := s.OldestPendingCreatedAt(ctx)
				require.NoError(t, err)
				assert.True(t, ok)
				assert.False(t, createdAt.IsZero())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			_, _, err := s.OldestPendingCreatedAt(canceledContext(t))
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}
