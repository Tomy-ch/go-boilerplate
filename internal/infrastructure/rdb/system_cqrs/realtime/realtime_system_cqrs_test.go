package realtime

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	realtimebndry "go-boilerplate/internal/usecase/boundary/realtime"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestAllocator は、共有テストDB上の allocator と tx 直列化ランナーを組み立てるテストヘルパーです。
func newTestAllocator(t *testing.T) (*allocator, testkit.TransactionRunner) {
	t.Helper()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	return &allocator{tracer: lt, db: testDB}, txm
}

// canceledContext は、キャンセル済みの context を返すテストヘルパーです。
func canceledContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestNewSequenceAllocator(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	tf := observability.NewNoopTracerFactory(t)
	expected := &allocator{
		tracer: tf.Infra(),
		db:     testDB,
	}

	assert.Equal(t, expected, NewSequenceAllocator(testDB, tf))
}

func Test_allocator_Allocate(t *testing.T) {
	t.Parallel()

	a, txm := newTestAllocator(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未採番のストリームは1から始まる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				seq, err := a.Allocate(ctx, "stream-allocate-first")
				require.NoError(t, err)
				assert.Equal(t, realtimebndry.Sequence(1), seq)
			})
		})

		t.Run("同一ストリームでは gap なく 1 ずつ進む", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				const streamID = realtimebndry.StreamID("stream-allocate-increment")
				for want := range 3 {
					seq, err := a.Allocate(ctx, streamID)
					require.NoError(t, err)
					assert.Equal(t, realtimebndry.Sequence(want+1), seq)
				}
			})
		})

		t.Run("別ストリームの採番は互いに影響しない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				first, err := a.Allocate(ctx, "stream-allocate-a")
				require.NoError(t, err)
				second, err := a.Allocate(ctx, "stream-allocate-b")
				require.NoError(t, err)

				assert.Equal(t, realtimebndry.Sequence(1), first)
				assert.Equal(t, realtimebndry.Sequence(1), second)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			_, err := a.Allocate(canceledContext(t), "stream-allocate-canceled")
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

func Test_allocator_Current(t *testing.T) {
	t.Parallel()

	a, txm := newTestAllocator(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("採番済みのストリームは現在位置とok=trueを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				const streamID = realtimebndry.StreamID("stream-current-allocated")
				_, err := a.Allocate(ctx, streamID)
				require.NoError(t, err)
				_, err = a.Allocate(ctx, streamID)
				require.NoError(t, err)

				seq, ok, err := a.Current(ctx, streamID)
				require.NoError(t, err)
				assert.True(t, ok)
				assert.Equal(t, realtimebndry.Sequence(2), seq)
			})
		})

		t.Run("未採番のストリームはok=falseを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				seq, ok, err := a.Current(ctx, "stream-current-unallocated")
				require.NoError(t, err)
				assert.False(t, ok)
				assert.Equal(t, realtimebndry.Sequence(0), seq)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			_, _, err := a.Current(canceledContext(t), "stream-current-canceled")
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}
