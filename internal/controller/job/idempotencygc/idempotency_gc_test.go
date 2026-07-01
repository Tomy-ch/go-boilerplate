package idempotencygc

import (
	"testing"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	idempotencybndry "go-boilerplate/internal/usecase/boundary/idempotency"
	mock_idempotency "go-boilerplate/internal/usecase/idempotency/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)
	log := logging.NewTestLogger(t)
	gc := mock_idempotency.NewMockGCUsecase(ctrl)

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

		t.Run("負数はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseBatchSize([]string{"--batch-size=-1"})
			require.Error(t, err)
		})
	})
}

func Test_jobImpl_Execute(t *testing.T) {
	t.Parallel()

	tf := observability.NewNoopTracerFactory(t)
	log := logging.NewTestLogger(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("フラグ無しは既定バッチサイズ(0)で usecase へ委譲する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			gc := mock_idempotency.NewMockGCUsecase(ctrl)
			gc.EXPECT().SweepExpired(gomock.Any(), int32(0)).Return(int64(0), nil)

			job := &jobImpl{logging: log, tracer: tf.Controller(), gc: gc}
			require.NoError(t, job.Execute(t.Context(), nil))
		})

		t.Run("--batch-size 指定はそのバッチサイズで usecase へ委譲する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			gc := mock_idempotency.NewMockGCUsecase(ctrl)
			gc.EXPECT().SweepExpired(gomock.Any(), int32(100)).Return(int64(3), nil)

			job := &jobImpl{logging: log, tracer: tf.Controller(), gc: gc}
			require.NoError(t, job.Execute(t.Context(), []string{"--batch-size=100"}))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不正フラグは掃除せずエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			gc := mock_idempotency.NewMockGCUsecase(ctrl)

			job := &jobImpl{logging: log, tracer: tf.Controller(), gc: gc}
			require.Error(t, job.Execute(t.Context(), []string{"--bad"}))
		})

		t.Run("掃除の失敗はエラーを伝播する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			gc := mock_idempotency.NewMockGCUsecase(ctrl)
			gc.EXPECT().
				SweepExpired(gomock.Any(), int32(0)).
				Return(int64(0), idempotencybndry.ErrLockTimeout)

			job := &jobImpl{logging: log, tracer: tf.Controller(), gc: gc}
			require.ErrorIs(t, job.Execute(t.Context(), nil), idempotencybndry.ErrLockTimeout)
		})
	})
}
