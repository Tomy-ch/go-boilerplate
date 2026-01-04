package job

import (
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	usecasejob "boilerplate-go/internal/usecase/support/job"
	mock_job "boilerplate-go/internal/usecase/support/job/mock"
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
		require.Len(t, r.Names(), 1)
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
		require.Error(t, err)
		require.Nil(t, r)
	})
}
