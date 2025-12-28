package job

import (
	"testing"

	"boilerplate-go/internal/usecase/support/job"
	mock_job "boilerplate-go/internal/usecase/support/job/mock"
	"boilerplate-go/pkg/xerrors"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_NewRunner(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	job1 := mock_job.NewMockJob(ctrl)
	job1.EXPECT().Name().Return("job1").AnyTimes()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ジョブが1件登録された場合、ランナーが正常に生成される", func(t *testing.T) {
			t.Parallel()

			runner, err := NewRunner([]job.Job{job1})

			require.NoError(t, err)
			require.Len(t, runner.Names(), 1)
		})

		t.Run("ジョブが複数登録された場合、ランナーが正常に生成される", func(t *testing.T) {
			t.Parallel()

			job2 := mock_job.NewMockJob(ctrl)
			job2.EXPECT().Name().Return("job2").AnyTimes()

			runner, err := NewRunner([]job.Job{job1, job2})

			require.NoError(t, err)
			require.Len(t, runner.Names(), 2)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同一名のジョブが登録された場合、エラーが返される", func(t *testing.T) {
			t.Parallel()

			res, err := NewRunner([]job.Job{job1, job1})

			require.Nil(t, res)
			require.Error(t, err)
		})
	})
}

func Test_runner_Run(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("登録されたジョブ名を指定して実行すると、ジョブが正常に実行される", func(t *testing.T) {
			t.Parallel()

			const jobName = "job"

			args := []string{"arg1", "arg2"}

			job1 := mock_job.NewMockJob(ctrl)
			job1.EXPECT().Name().Return(jobName).AnyTimes()
			job1.EXPECT().Execute(gomock.Any(), args).Return(nil)
			runner, err := NewRunner([]job.Job{job1})
			require.NoError(t, err)

			err = runner.Run(t.Context(), jobName, args)
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未登録のジョブ名を指定して実行すると、エラーが返される", func(t *testing.T) {
			t.Parallel()

			const jobName = "job"
			const unknownJobName = "unknown-job"

			job1 := mock_job.NewMockJob(ctrl)
			job1.EXPECT().Name().Return(jobName).AnyTimes()
			runner, err := NewRunner([]job.Job{job1})
			require.NoError(t, err)

			err = runner.Run(t.Context(), unknownJobName, []string{})
			require.Error(t, err)
		})

		t.Run("ジョブの実行に失敗すると、エラーが返される", func(t *testing.T) {
			t.Parallel()

			const jobName = "job"
			expectedErr := xerrors.New("assertion error")

			job1 := mock_job.NewMockJob(ctrl)
			job1.EXPECT().Name().Return(jobName).AnyTimes()
			job1.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(expectedErr)
			runner, err := NewRunner([]job.Job{job1})
			require.NoError(t, err)

			actualErr := runner.Run(t.Context(), jobName, []string{})
			require.ErrorIs(t, expectedErr, actualErr)
		})
	})
}

func Test_runner_Names(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("登録されたジョブ名の一覧が正しく返される", func(t *testing.T) {
			t.Parallel()

			const jobName = "job"

			job1 := mock_job.NewMockJob(ctrl)
			job1.EXPECT().Name().Return(jobName).AnyTimes()

			runner, err := NewRunner([]job.Job{job1})
			require.NoError(t, err)

			names := runner.Names()
			require.ElementsMatch(t, []string{jobName}, names)
		})
	})
}
