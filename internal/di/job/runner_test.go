package job

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	jobrunner "go-boilerplate/internal/controller/job"
	usecasejob "go-boilerplate/internal/usecase/boundary/job"
	mock_job "go-boilerplate/internal/usecase/boundary/job/mock"
)

func TestProvideRunner(t *testing.T) {
	t.Parallel()

	t.Run("正常系: ジョブが1件の場合、ランナーが返る", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)

		j1 := mock_job.NewMockJob(ctrl)
		j1.EXPECT().Name().Return("job1").AnyTimes()

		in := RunnerIn{Jobs: []usecasejob.Job{j1}}
		r, err := ProvideRunner(in)
		require.NoError(t, err)
		require.NotNil(t, r)
		assert.Equal(t, []string{"job1"}, r.Names())
	})

	t.Run("正常系: ジョブが0件の場合、空のランナーが返る", func(t *testing.T) {
		t.Parallel()

		r, err := ProvideRunner(RunnerIn{Jobs: []usecasejob.Job{}})
		require.NoError(t, err)
		require.NotNil(t, r)
		assert.Empty(t, r.Names())
	})

	t.Run("異常系: 同一名ジョブがあるとエラー", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)

		j1 := mock_job.NewMockJob(ctrl)
		j1.EXPECT().Name().Return("dup").AnyTimes()
		j2 := mock_job.NewMockJob(ctrl)
		j2.EXPECT().Name().Return("dup").AnyTimes()

		in := RunnerIn{Jobs: []usecasejob.Job{j1, j2}}
		r, err := ProvideRunner(in)
		require.ErrorIs(t, err, jobrunner.ErrDuplicateJob)
		assert.Nil(t, r)
	})
}
