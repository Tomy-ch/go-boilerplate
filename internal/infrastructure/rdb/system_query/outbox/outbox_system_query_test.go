package outbox

import (
	"context"
	"testing"
	"time"

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

		t.Run("Insert→claim→publish の一連が成立し published 後は claim されない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				msgID, err := s.Insert(ctx, emitParams())
				require.NoError(t, err)
				assert.False(t, msgID.IsNil())

				// pending を claim すると挿入行が返る。
				msgs, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				require.Len(t, msgs, 1)
				assert.Equal(t, msgID, msgs[0].MessageID)
				assert.Equal(t, "purchase.created.v1", msgs[0].EventType)
				assert.JSONEq(t, `{"v":1}`, string(msgs[0].Payload))
				assert.Equal(t, int32(0), msgs[0].Attempts)

				// published へ遷移すると再 claim では返らない。
				require.NoError(t, s.MarkPublished(ctx, msgs[0].ID))
				after, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				assert.Empty(t, after)
			})
		})

		t.Run("MarkFailed は attempts を加算し加算後の値を返す", func(t *testing.T) {
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

		t.Run("MarkDead→ReplayDead で dead が pending へ戻る", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				_, err := s.Insert(ctx, emitParams())
				require.NoError(t, err)
				msgs, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				require.Len(t, msgs, 1)

				require.NoError(t, s.MarkDead(ctx, msgs[0].ID))
				// dead 中は claim されない。
				none, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				assert.Empty(t, none)

				// replay で pending へ復帰し再び claim される。
				replayed, err := s.ReplayDead(ctx, nil)
				require.NoError(t, err)
				assert.GreaterOrEqual(t, replayed, int64(1))

				back, err := s.ClaimPending(ctx, 10)
				require.NoError(t, err)
				assert.Len(t, back, 1)
			})
		})

		t.Run("DeletePublished は cutoff より古い published 行を削除する", func(t *testing.T) {
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

		t.Run("OldestPendingCreatedAt は pending 有無で ok を返す", func(t *testing.T) {
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

		t.Run("MarkFailed は pending でない id に対しては 0 を返す", func(t *testing.T) {
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

		t.Run("コンテキストキャンセル時は各操作が DB エラーを返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			_, err := s.Insert(ctx, emitParams())
			require.Error(t, err)

			_, err = s.ClaimPending(ctx, 10)
			require.Error(t, err)

			require.Error(t, s.MarkPublished(ctx, 1))

			_, err = s.MarkFailed(ctx, 1, "boom")
			require.Error(t, err)

			require.Error(t, s.MarkDead(ctx, 1))

			_, err = s.ReplayDead(ctx, nil)
			require.Error(t, err)

			_, err = s.DeletePublished(ctx, time.Now(), 1)
			require.Error(t, err)

			_, _, err = s.OldestPendingCreatedAt(ctx)
			require.Error(t, err)
		})
	})
}
