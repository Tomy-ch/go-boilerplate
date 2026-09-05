package realtime

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/infrastructure/system"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	realtimebndry "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/xerrors"

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

// Test_allocator_Allocate_concurrentSerialization は、独立した 2 つのトランザクションが同一ストリームを
// 同時に採番したとき、行ロックにより直列化され gap も重複も生じないことを検証します。
//
// testkit の WithinTx は全 tx を直列化するため、この競合は構造的に再現できません。
// outbox 側の SKIP LOCKED テストと同様に driver.NewTransactionManager を直接使い 2 tx を並行させます。
// 保持側が commit するため専用 stream_id の行を投入し、後始末で必ず削除します。
//
//nolint:paralleltest // commit した fixture が他テストと衝突しないよう非並列にする
func Test_allocator_Allocate_concurrentSerialization(t *testing.T) {
	testDB := testkit.NewTestDB(t)
	a := &allocator{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}

	dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
	txm := driver.NewTransactionManager(testDB, dbCfg, logging.NewTestLogger(t), system.NewSleeper())

	const streamID = realtimebndry.StreamID("stream-allocate-concurrent")

	t.Cleanup(func() {
		_ = txm.Do(context.Background(), func(ctx context.Context) error {
			_, err := driver.New(ctx, testDB).
				Exec(ctx, "DELETE FROM realtime_stream_sequences WHERE stream_id = $1", streamID.String())
			return err
		})
	})

	allocated := make(chan struct{})
	release := make(chan struct{})
	holderDone := make(chan error, 1)

	var once sync.Once
	rel := func() { once.Do(func() { close(release) }) }
	t.Cleanup(rel) // 失敗時も保持側 goroutine をリークさせない

	// 保持側: 採番して commit まで行ロックを保持する。
	go func() {
		holderDone <- txm.Do(context.Background(), func(ctx context.Context) error {
			seq, err := a.Allocate(ctx, streamID)
			if err != nil {
				return err
			}
			if seq != realtimebndry.Sequence(1) {
				return xerrors.New(fmt.Sprintf("holder: expected sequence 1, got %d", seq))
			}
			close(allocated)
			<-release
			return nil
		})
	}()

	select {
	case <-allocated:
	case err := <-holderDone:
		require.NoError(t, err, "保持側 tx が採番前に失敗した")
		return
	}

	contenderDone := make(chan realtimebndry.Sequence, 1)
	go func() {
		// 保持側の行ロックが解けるまでブロックし、解けた後に続きの位置を得る。
		var seq realtimebndry.Sequence
		if err := txm.Do(context.Background(), func(ctx context.Context) error {
			var aerr error
			seq, aerr = a.Allocate(ctx, streamID)
			return aerr
		}); err != nil {
			seq = 0
		}
		contenderDone <- seq
	}()

	rel()
	require.NoError(t, <-holderDone)

	// 直列化されていれば競合側は 1 を再取得せず 2 を得る（gap も重複も無い）。
	assert.Equal(t, realtimebndry.Sequence(2), <-contenderDone)
}

// Test_allocator_Current_excludesUncommittedAllocation は、History の cursor が完全な prefix である
// ことを固定します。Current が未コミットの採番まで返すと、cursor 以下に未コミットの位置が混じり、
// after=cursor で繋ぎ直した接続がその位置を取りこぼします
// （親 issue の受入基準「History 取得と SSE 接続の間の event を取りこぼさない」の前半）。
//
//nolint:paralleltest // commit した fixture が他テストと衝突しないよう非並列にする
func Test_allocator_Current_excludesUncommittedAllocation(t *testing.T) {
	testDB := testkit.NewTestDB(t)
	a := &allocator{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}

	dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
	txm := driver.NewTransactionManager(testDB, dbCfg, logging.NewTestLogger(t), system.NewSleeper())

	const streamID = realtimebndry.StreamID("stream-current-uncommitted")

	t.Cleanup(func() {
		_ = txm.Do(context.Background(), func(ctx context.Context) error {
			_, err := driver.New(ctx, testDB).
				Exec(ctx, "DELETE FROM realtime_stream_sequences WHERE stream_id = $1", streamID.String())
			return err
		})
	})

	require.NoError(t, txm.Do(context.Background(), func(ctx context.Context) error {
		_, err := a.Allocate(ctx, streamID)
		return err
	}))

	allocated := make(chan struct{})
	release := make(chan struct{})
	holderDone := make(chan error, 1)

	var once sync.Once
	rel := func() { once.Do(func() { close(release) }) }
	t.Cleanup(rel)

	go func() {
		holderDone <- txm.Do(context.Background(), func(ctx context.Context) error {
			seq, err := a.Allocate(ctx, streamID)
			if err != nil {
				return err
			}
			if seq != realtimebndry.Sequence(2) {
				return xerrors.New(fmt.Sprintf("holder: expected sequence 2, got %d", seq))
			}
			close(allocated)
			<-release
			return nil
		})
	}()

	select {
	case <-allocated:
	case err := <-holderDone:
		require.NoError(t, err, "保持側 tx が採番前に失敗した")
		return
	}

	held, ok, err := a.Current(context.Background(), streamID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, realtimebndry.Sequence(1), held, "未コミットの採番が cursor に見えている")

	rel()
	require.NoError(t, <-holderDone)

	after, ok, err := a.Current(context.Background(), streamID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, realtimebndry.Sequence(2), after)
}
