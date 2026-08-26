package productimagegc

import (
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/product"
	mock_product "go-boilerplate/internal/usecase/product/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)
	log := logging.NewTestLogger(t)
	gc := mock_product.NewMockImageGCUsecase(ctrl)

	job := New(log, tf, gc)
	require.NotNil(t, job)
}

func Test_jobImpl_Name(t *testing.T) {
	t.Parallel()
	job := &jobImpl{}
	assert.Equal(t, jobName, job.Name())
}

func Test_parseArgs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("フラグ無しは既定値委譲(0)と実削除を表す", func(t *testing.T) {
			t.Parallel()
			got, err := parseArgs(nil)
			require.NoError(t, err)
			assert.Equal(t, options{}, got)
		})

		t.Run("--older-than=はduration文字列として解釈する", func(t *testing.T) {
			t.Parallel()
			got, err := parseArgs([]string{"--older-than=48h"})
			require.NoError(t, err)
			assert.Equal(t, 48*time.Hour, got.grace)
		})

		t.Run("--batch-size=Nを解釈する", func(t *testing.T) {
			t.Parallel()
			got, err := parseArgs([]string{"--batch-size=250"})
			require.NoError(t, err)
			assert.Equal(t, int32(250), got.batchSize)
		})

		t.Run("--dry-runを解釈する", func(t *testing.T) {
			t.Parallel()
			got, err := parseArgs([]string{dryRunFlag})
			require.NoError(t, err)
			assert.True(t, got.dryRun)
		})

		t.Run("3フラグの併用を順不同で解釈する", func(t *testing.T) {
			t.Parallel()
			got, err := parseArgs([]string{dryRunFlag, "--batch-size=10", "--older-than=1h"})
			require.NoError(t, err)
			assert.Equal(t, options{grace: time.Hour, batchSize: 10, dryRun: true}, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未知のフラグはエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseArgs([]string{"--unknown"})
			require.ErrorIs(t, err, errUnknownFlag)
		})

		t.Run("--older-thanの重複指定はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseArgs([]string{"--older-than=1h", "--older-than=2h"})
			require.ErrorIs(t, err, errDuplicateFlag)
		})

		t.Run("--batch-sizeの重複指定はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseArgs([]string{"--batch-size=1", "--batch-size=2"})
			require.ErrorIs(t, err, errDuplicateFlag)
		})

		t.Run("--dry-runの重複指定はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseArgs([]string{dryRunFlag, dryRunFlag})
			require.ErrorIs(t, err, errDuplicateFlag)
		})

		t.Run("--older-thanが解釈できない値はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseArgs([]string{"--older-than=30d"})
			require.ErrorIs(t, err, errInvalidOlderThan)
		})

		t.Run("--older-thanが0以下はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseArgs([]string{"--older-than=0s"})
			require.ErrorIs(t, err, errInvalidOlderThan)
		})

		t.Run("--batch-sizeが非数値はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseArgs([]string{"--batch-size=abc"})
			require.ErrorIs(t, err, errInvalidBatchSize)
		})

		t.Run("--batch-sizeが0以下はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseArgs([]string{"--batch-size=0"})
			require.ErrorIs(t, err, errInvalidBatchSize)
		})
	})
}

func Test_resultMessage(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("dry-runでは削除していないことを明示する", func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, resultMessage(true), "dry-run")
		})

		t.Run("実削除ではdry-runの断りを含めない", func(t *testing.T) {
			t.Parallel()
			assert.NotContains(t, resultMessage(false), "dry-run")
		})
	})
}

func Test_abortedMessage(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("dry-runでは削除していないことを明示する", func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, abortedMessage(true), "dry-run")
		})

		t.Run("実削除では併記した件数が削除済みであることを明示する", func(t *testing.T) {
			t.Parallel()
			assert.NotContains(t, abortedMessage(false), "dry-run")
			assert.Contains(t, abortedMessage(false), "already deleted")
		})
	})
}

func Test_jobImpl_Execute(t *testing.T) {
	t.Parallel()

	tf := observability.NewNoopTracerFactory(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("フラグ無しは既定値委譲(0)と実削除で usecase へ委譲する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			gc := mock_product.NewMockImageGCUsecase(ctrl)
			gc.EXPECT().SweepOrphans(gomock.Any(), time.Duration(0), int32(0), false).
				Return(product.ImageGCResult{}, nil)

			job := &jobImpl{logging: logging.NewTestLogger(t), tracer: tf.Controller(), gc: gc}
			require.NoError(t, job.Execute(t.Context(), nil))
		})

		t.Run("3フラグの併用をそのまま usecase へ渡す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			gc := mock_product.NewMockImageGCUsecase(ctrl)
			gc.EXPECT().SweepOrphans(gomock.Any(), 48*time.Hour, int32(100), true).
				Return(product.ImageGCResult{}, nil)

			job := &jobImpl{logging: logging.NewTestLogger(t), tracer: tf.Controller(), gc: gc}
			require.NoError(t, job.Execute(t.Context(), []string{"--older-than=48h", "--batch-size=100", dryRunFlag}))
		})

		t.Run("削除件数と検査件数を結果ログのキーへ出力する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			gc := mock_product.NewMockImageGCUsecase(ctrl)
			gc.EXPECT().SweepOrphans(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(product.ImageGCResult{Deleted: 3, Scanned: 7}, nil)

			log, logs := logging.NewObservedTestLogger(t)
			job := &jobImpl{logging: log, tracer: tf.Controller(), gc: gc}
			require.NoError(t, job.Execute(t.Context(), nil))

			entries := logs.All()
			require.Len(t, entries, 1)
			assert.Equal(t, "info", entries[0].Level.String())
			assert.Equal(t, resultMessage(false), entries[0].Message)
			assert.Equal(t, int64(3), entries[0].ContextMap()[logging.JobResultKey])
			assert.Equal(t, int64(7), entries[0].ContextMap()[logging.JobScannedKey])
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不正フラグは回収せずエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			gc := mock_product.NewMockImageGCUsecase(ctrl)

			job := &jobImpl{logging: logging.NewTestLogger(t), tracer: tf.Controller(), gc: gc}
			require.ErrorIs(t, job.Execute(t.Context(), []string{"--bad"}), errUnknownFlag)
		})

		t.Run("回収の失敗はエラーを伝播する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			gc := mock_product.NewMockImageGCUsecase(ctrl)
			wantErr := apperror.ErrUnavailable
			gc.EXPECT().SweepOrphans(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(product.ImageGCResult{}, wantErr)

			job := &jobImpl{logging: logging.NewTestLogger(t), tracer: tf.Controller(), gc: gc}
			require.ErrorIs(t, job.Execute(t.Context(), nil), wantErr)
		})

		t.Run("回収が失敗しても中断までに確定した件数はログへ出力する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			gc := mock_product.NewMockImageGCUsecase(ctrl)
			gc.EXPECT().SweepOrphans(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(product.ImageGCResult{Deleted: 5, Scanned: 9}, apperror.ErrUnavailable)

			log, logs := logging.NewObservedTestLogger(t)
			job := &jobImpl{logging: log, tracer: tf.Controller(), gc: gc}
			require.Error(t, job.Execute(t.Context(), nil))

			entries := logs.All()
			require.Len(t, entries, 1)
			assert.Equal(t, "warn", entries[0].Level.String())
			assert.Equal(t, abortedMessage(false), entries[0].Message)
			assert.Equal(t, int64(5), entries[0].ContextMap()[logging.JobResultKey])
			assert.Equal(t, int64(9), entries[0].ContextMap()[logging.JobScannedKey])
		})
	})
}
