package idempotencygc

import (
	"testing"
	"time"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	idempotencybndry "go-boilerplate/internal/usecase/boundary/idempotency"
	mock_idempotency "go-boilerplate/internal/usecase/boundary/idempotency/mock"
	"go-boilerplate/internal/usecase/idempotency"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newGC(ctrl *gomock.Controller, store idempotencybndry.Store, now time.Time) idempotency.GCUsecase {
	clk := mock_clock.NewMockClock(ctrl)
	clk.EXPECT().Now().Return(now).AnyTimes()
	return idempotency.NewGC(store, clk)
}

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)
	log := logging.NewTestLogger(t)
	gc := newGC(ctrl, mock_idempotency.NewMockStore(ctrl), time.Now())

	job := New(log, tf, gc)
	require.NotNil(t, job)
}

func Test_jobImpl_Name(t *testing.T) {
	t.Parallel()
	job := &jobImpl{}
	assert.Equal(t, jobName, job.Name())
}

func Test_parseBatchSize(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("フラグ無しは0(usecase既定値)を返す", func(t *testing.T) {
			t.Parallel()
			got, err := parseBatchSize(nil)
			require.NoError(t, err)
			assert.Equal(t, int32(0), got)
		})

		t.Run("--batch-size=N を解釈する", func(t *testing.T) {
			t.Parallel()
			got, err := parseBatchSize([]string{"--batch-size=500"})
			require.NoError(t, err)
			assert.Equal(t, int32(500), got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未知のフラグはエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseBatchSize([]string{"--unknown"})
			require.Error(t, err)
		})

		t.Run("重複指定はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseBatchSize([]string{"--batch-size=1", "--batch-size=2"})
			require.Error(t, err)
		})

		t.Run("0以下はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseBatchSize([]string{"--batch-size=0"})
			require.Error(t, err)
		})

		t.Run("非数値はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseBatchSize([]string{"--batch-size=abc"})
			require.Error(t, err)
		})
	})
}

func Test_jobImpl_Execute(t *testing.T) {
	t.Parallel()

	tf := observability.NewNoopTracerFactory(t)
	log := logging.NewTestLogger(t)
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("フラグ無しは既定バッチサイズで掃除する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			store.EXPECT().
				DeleteExpired(gomock.Any(), now, idempotency.DefaultGCBatchSize).
				Return(int64(0), nil)

			job := &jobImpl{logging: log, tracer: tf.Controller(), gc: newGC(ctrl, store, now)}
			require.NoError(t, job.Execute(t.Context(), nil))
		})

		t.Run("--batch-size 指定はそのバッチサイズで掃除する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			store.EXPECT().
				DeleteExpired(gomock.Any(), now, int32(100)).
				Return(int64(3), nil)

			job := &jobImpl{logging: log, tracer: tf.Controller(), gc: newGC(ctrl, store, now)}
			require.NoError(t, job.Execute(t.Context(), []string{"--batch-size=100"}))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不正フラグは掃除せずエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)

			job := &jobImpl{logging: log, tracer: tf.Controller(), gc: newGC(ctrl, store, now)}
			require.Error(t, job.Execute(t.Context(), []string{"--bad"}))
		})

		t.Run("掃除の失敗はエラーを伝播する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			store.EXPECT().
				DeleteExpired(gomock.Any(), now, idempotency.DefaultGCBatchSize).
				Return(int64(0), idempotencybndry.ErrLockTimeout)

			job := &jobImpl{logging: log, tracer: tf.Controller(), gc: newGC(ctrl, store, now)}
			require.ErrorIs(t, job.Execute(t.Context(), nil), idempotencybndry.ErrLockTimeout)
		})
	})
}
