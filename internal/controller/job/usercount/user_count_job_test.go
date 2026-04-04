package usercount

import (
	"testing"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_user "go-boilerplate/internal/usecase/user/mock"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)

	logging := logging.NewTestLogger(t)

	mockApp := mock_user.NewMockUsecase(ctrl)

	job := New(logging, tf, mockApp)
	require.NotNil(t, job)
}

func Test_jobImpl_Name(t *testing.T) {
	t.Parallel()
	job := &jobImpl{}
	actual := job.Name()

	require.Equal(t, jobName, actual)
}

func Test_jobImpl_Execute(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	tf := observability.NewNoopTracerFactory(t)
	logging := logging.NewTestLogger(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("--active-onlyオプションが指定された場合、CountUsersがtrueで呼び出される", func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.
				EXPECT().
				CountUsers(gomock.Any(), gomock.Any()).
				Return(int64(42), nil)

			job := &jobImpl{
				logging: logging,
				tracer:  tf.Controller(),
				usecase: mockApp,
			}

			err := job.Execute(ctx, []string{"--active-only"})
			require.NoError(t, err)
		})

		t.Run("--inactive-onlyオプションが指定された場合、CountUsersがfalseで呼び出される", func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.
				EXPECT().
				CountUsers(gomock.Any(), gomock.Any()).
				Return(int64(24), nil)

			job := &jobImpl{
				logging: logging,
				tracer:  tf.Controller(),
				usecase: mockApp,
			}

			err := job.Execute(ctx, []string{"--inactive-only"})
			require.NoError(t, err)
		})

		t.Run("オプションが指定されなかった場合、CountUsersがnilで呼び出される", func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.
				EXPECT().
				CountUsers(gomock.Any(), gomock.Any()).
				Return(int64(100), nil)

			job := &jobImpl{
				logging: logging,
				tracer:  tf.Controller(),
				usecase: mockApp,
			}

			err := job.Execute(ctx, []string{})
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("CountUsersがエラーを返した場合、Executeもエラーを返す", func(t *testing.T) {
			t.Parallel()
			assertError := xerrors.New("assert error")
			ctx := t.Context()
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.
				EXPECT().
				CountUsers(gomock.Any(), gomock.Any()).
				Return(int64(0), assertError)

			job := &jobImpl{
				logging: logging,
				tracer:  tf.Controller(),
				usecase: mockApp,
			}

			err := job.Execute(ctx, []string{})
			require.Equal(t, assertError, err)
		})
	})
}
