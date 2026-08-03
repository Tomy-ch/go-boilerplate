package userpurge

import (
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/user"
	mock_user "go-boilerplate/internal/usecase/user/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)
	log := logging.NewTestLogger(t)
	purge := mock_user.NewMockPurgeUsecase(ctrl)

	job := New(log, tf, purge)
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
			got, err := parseArgs([]string{"--older-than=720h"})
			require.NoError(t, err)
			assert.Equal(t, options{retention: 720 * time.Hour}, got)
		})

		t.Run("--batch-size=Nを解釈する", func(t *testing.T) {
			t.Parallel()
			got, err := parseArgs([]string{"--batch-size=100"})
			require.NoError(t, err)
			assert.Equal(t, options{batchSize: 100}, got)
		})

		t.Run("--dry-runは値なしのブールフラグとして解釈する", func(t *testing.T) {
			t.Parallel()
			got, err := parseArgs([]string{"--dry-run"})
			require.NoError(t, err)
			assert.Equal(t, options{dryRun: true}, got)
		})

		t.Run("3フラグの併用をすべて解釈する", func(t *testing.T) {
			t.Parallel()
			got, err := parseArgs([]string{"--dry-run", "--older-than=1h30m", "--batch-size=5"})
			require.NoError(t, err)
			assert.Equal(t, options{retention: 90 * time.Minute, batchSize: 5, dryRun: true}, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未知のフラグはエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseArgs([]string{"--unknown"})
			require.ErrorIs(t, err, errUnknownFlag)
		})

		t.Run("--older-thanの重複はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseArgs([]string{"--older-than=1h", "--older-than=2h"})
			require.ErrorIs(t, err, errDuplicateFlag)
		})

		t.Run("--batch-sizeの重複はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseArgs([]string{"--batch-size=1", "--batch-size=2"})
			require.ErrorIs(t, err, errDuplicateFlag)
		})

		t.Run("--dry-runの重複はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseArgs([]string{"--dry-run", "--dry-run"})
			require.ErrorIs(t, err, errDuplicateFlag)
		})

		t.Run("--older-thanの解析失敗はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseArgs([]string{"--older-than=30d"})
			require.ErrorIs(t, err, errInvalidOlderThan)
		})

		t.Run("--older-thanの0はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseArgs([]string{"--older-than=0s"})
			require.ErrorIs(t, err, errInvalidOlderThan)
		})

		t.Run("--older-thanの負値はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseArgs([]string{"--older-than=-1h"})
			require.ErrorIs(t, err, errInvalidOlderThan)
		})

		t.Run("--batch-sizeの0はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseArgs([]string{"--batch-size=0"})
			require.ErrorIs(t, err, errInvalidBatchSize)
		})

		t.Run("--batch-sizeの負値はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseArgs([]string{"--batch-size=-1"})
			require.ErrorIs(t, err, errInvalidBatchSize)
		})

		t.Run("--batch-sizeの非数値はエラー", func(t *testing.T) {
			t.Parallel()
			_, err := parseArgs([]string{"--batch-size=abc"})
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
			purge := mock_user.NewMockPurgeUsecase(ctrl)
			purge.EXPECT().PurgeDeleted(gomock.Any(), time.Duration(0), int32(0), false).
				Return(user.PurgeResult{}, nil)

			job := &jobImpl{logging: logging.NewTestLogger(t), tracer: tf.Controller(), purge: purge}
			require.NoError(t, job.Execute(t.Context(), nil))
		})

		t.Run("3フラグの併用をそのまま usecase へ渡す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			purge := mock_user.NewMockPurgeUsecase(ctrl)
			purge.EXPECT().PurgeDeleted(gomock.Any(), 720*time.Hour, int32(100), true).
				Return(user.PurgeResult{}, nil)

			job := &jobImpl{logging: logging.NewTestLogger(t), tracer: tf.Controller(), purge: purge}
			require.NoError(t, job.Execute(t.Context(), []string{"--older-than=720h", "--batch-size=100", "--dry-run"}))
		})

		t.Run("削除件数とスキップ件数を結果ログのキーへ出力する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			purge := mock_user.NewMockPurgeUsecase(ctrl)
			purge.EXPECT().PurgeDeleted(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(user.PurgeResult{Purged: 3, SkippedWithPurchases: 2}, nil)

			log, logs := logging.NewObservedTestLogger(t)
			job := &jobImpl{logging: log, tracer: tf.Controller(), purge: purge}
			require.NoError(t, job.Execute(t.Context(), nil))

			entries := logs.All()
			require.Len(t, entries, 1)
			assert.Equal(t, int64(3), entries[0].ContextMap()[logging.JobResultKey])
			assert.Equal(t, int64(2), entries[0].ContextMap()[logging.JobSkippedKey])
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不正フラグは削除せずエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			purge := mock_user.NewMockPurgeUsecase(ctrl)

			job := &jobImpl{logging: logging.NewTestLogger(t), tracer: tf.Controller(), purge: purge}
			require.ErrorIs(t, job.Execute(t.Context(), []string{"--bad"}), errUnknownFlag)
		})

		t.Run("削除の失敗はエラーを伝播する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			purge := mock_user.NewMockPurgeUsecase(ctrl)
			wantErr := apperror.ErrUnavailable
			purge.EXPECT().PurgeDeleted(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(user.PurgeResult{}, wantErr)

			job := &jobImpl{logging: logging.NewTestLogger(t), tracer: tf.Controller(), purge: purge}
			require.ErrorIs(t, job.Execute(t.Context(), nil), wantErr)
		})

		t.Run("削除が失敗しても中断までに確定した件数はログへ出力する", func(t *testing.T) {
			t.Parallel()

			// コミット済みの物理削除は取り消せない。エラーだけを返して件数を落とすと、
			// 実際に消えたユーザーの数が運用者から見えなくなる。
			ctrl := gomock.NewController(t)
			purge := mock_user.NewMockPurgeUsecase(ctrl)
			purge.EXPECT().PurgeDeleted(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(user.PurgeResult{Purged: 5, SkippedWithPurchases: 1}, apperror.ErrUnavailable)

			log, logs := logging.NewObservedTestLogger(t)
			job := &jobImpl{logging: log, tracer: tf.Controller(), purge: purge}
			require.Error(t, job.Execute(t.Context(), nil))

			entries := logs.All()
			require.Len(t, entries, 1)
			assert.Equal(t, int64(5), entries[0].ContextMap()[logging.JobResultKey])
			assert.Equal(t, int64(1), entries[0].ContextMap()[logging.JobSkippedKey])
		})
	})
}
