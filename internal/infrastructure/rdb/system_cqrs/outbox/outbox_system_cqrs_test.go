package outbox

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
