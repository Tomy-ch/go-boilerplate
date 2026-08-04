package outboxgc

import (
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_outbox "go-boilerplate/internal/usecase/outbox/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)
	log := logging.NewTestLogger(t)
	gc := mock_outbox.NewMockGCUsecase(ctrl)

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
			require.ErrorIs(t, err, errUnknownFlag)
		})

		t.Run("重複指定はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseBatchSize([]string{"--batch-size=1", "--batch-size=2"})
			require.ErrorIs(t, err, errDuplicateFlag)
		})

		t.Run("0以下はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseBatchSize([]string{"--batch-size=0"})
			require.ErrorIs(t, err, errInvalidBatchSize)
		})

		t.Run("非数値はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseBatchSize([]string{"--batch-size=abc"})
			require.ErrorIs(t, err, errInvalidBatchSize)
		})

		t.Run("負数はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseBatchSize([]string{"--batch-size=-1"})
			require.ErrorIs(t, err, errInvalidBatchSize)
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
			gc := mock_outbox.NewMockGCUsecase(ctrl)
			gc.EXPECT().SweepPublished(gomock.Any(), int32(0)).Return(int64(0), nil)

			job := &jobImpl{logging: log, tracer: tf.Controller(), gc: gc}
			require.NoError(t, job.Execute(t.Context(), nil))
		})

		t.Run("--batch-size 指定はそのバッチサイズで usecase へ委譲する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			gc := mock_outbox.NewMockGCUsecase(ctrl)
			gc.EXPECT().SweepPublished(gomock.Any(), int32(100)).Return(int64(3), nil)

			job := &jobImpl{logging: log, tracer: tf.Controller(), gc: gc}
			require.NoError(t, job.Execute(t.Context(), []string{"--batch-size=100"}))
		})

		t.Run("削除件数を結果ログのキーへ出力する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			gc := mock_outbox.NewMockGCUsecase(ctrl)
			gc.EXPECT().SweepPublished(gomock.Any(), int32(0)).Return(int64(7), nil)

			observedLog, logs := logging.NewObservedTestLogger(t)
			job := &jobImpl{logging: observedLog, tracer: tf.Controller(), gc: gc}
			require.NoError(t, job.Execute(t.Context(), nil))

			entries := logs.All()
			require.Len(t, entries, 1)
			assert.Equal(t, "info", entries[0].Level.String())
			assert.Equal(t, resultMessage, entries[0].Message)
			assert.Equal(t, int64(7), entries[0].ContextMap()[logging.JobResultKey])
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不正フラグは掃除せずエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			gc := mock_outbox.NewMockGCUsecase(ctrl)

			job := &jobImpl{logging: log, tracer: tf.Controller(), gc: gc}
			require.Error(t, job.Execute(t.Context(), []string{"--bad"}))
		})

		t.Run("掃除の失敗はエラーを伝播する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			gc := mock_outbox.NewMockGCUsecase(ctrl)
			wantErr := apperror.ErrUnavailable
			gc.EXPECT().SweepPublished(gomock.Any(), int32(0)).Return(int64(0), wantErr)

			job := &jobImpl{logging: log, tracer: tf.Controller(), gc: gc}
			require.ErrorIs(t, job.Execute(t.Context(), nil), wantErr)
		})

		t.Run("掃除が失敗しても中断までに確定した件数はログへ出力する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gc := mock_outbox.NewMockGCUsecase(ctrl)
			gc.EXPECT().SweepPublished(gomock.Any(), int32(0)).Return(int64(5), apperror.ErrUnavailable)

			observedLog, logs := logging.NewObservedTestLogger(t)
			job := &jobImpl{logging: observedLog, tracer: tf.Controller(), gc: gc}
			require.Error(t, job.Execute(t.Context(), nil))

			entries := logs.All()
			require.Len(t, entries, 1)
			assert.Equal(t, "warn", entries[0].Level.String())
			assert.Equal(t, abortedMessage, entries[0].Message)
			assert.Equal(t, int64(5), entries[0].ContextMap()[logging.JobResultKey])
		})
	})
}
