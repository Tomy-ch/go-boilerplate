package usercount

import (
	"context"
	"testing"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_user "go-boilerplate/internal/usecase/user/mock"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
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

	// New が job.Job 実装を生成し、Name() が規定値を返すことまで検証する。
	assert.Equal(t, jobName, job.Name())
}

func Test_jobImpl_Name(t *testing.T) {
	t.Parallel()
	job := &jobImpl{}
	actual := job.Name()

	assert.Equal(t, jobName, actual)
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
			var gotActive *bool
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.
				EXPECT().
				CountUsers(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, active *bool) (int64, error) {
					gotActive = active
					return int64(42), nil
				})

			job := &jobImpl{
				logging: logging,
				tracer:  tf.Controller(),
				usecase: mockApp,
			}

			err := job.Execute(ctx, []string{"--active-only"})
			require.NoError(t, err)
			require.NotNil(t, gotActive)
			assert.True(t, *gotActive)
		})

		t.Run("--inactive-onlyオプションが指定された場合、CountUsersがfalseで呼び出される", func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var gotActive *bool
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.
				EXPECT().
				CountUsers(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, active *bool) (int64, error) {
					gotActive = active
					return int64(24), nil
				})

			job := &jobImpl{
				logging: logging,
				tracer:  tf.Controller(),
				usecase: mockApp,
			}

			err := job.Execute(ctx, []string{"--inactive-only"})
			require.NoError(t, err)
			require.NotNil(t, gotActive)
			assert.False(t, *gotActive)
		})

		t.Run("オプションが指定されなかった場合、CountUsersがnilで呼び出される", func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var gotActive *bool
			called := false
			mockApp := mock_user.NewMockUsecase(ctrl)
			mockApp.
				EXPECT().
				CountUsers(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, active *bool) (int64, error) {
					gotActive = active
					called = true
					return int64(100), nil
				})

			job := &jobImpl{
				logging: logging,
				tracer:  tf.Controller(),
				usecase: mockApp,
			}

			err := job.Execute(ctx, []string{})
			require.NoError(t, err)
			require.True(t, called)
			assert.Nil(t, gotActive)
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
			require.ErrorIs(t, err, assertError)
		})

		t.Run("未知のフラグが指定された場合、CountUsersを呼ばずにエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			// CountUsers は呼ばれない（EXPECT 未設定）
			mockApp := mock_user.NewMockUsecase(ctrl)

			job := &jobImpl{
				logging: logging,
				tracer:  tf.Controller(),
				usecase: mockApp,
			}

			err := job.Execute(ctx, []string{"--actve-only"})
			require.Error(t, err)
		})

		t.Run("相反するフィルタフラグが同時指定された場合、CountUsersを呼ばずにエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			// CountUsers は呼ばれない（EXPECT 未設定）
			mockApp := mock_user.NewMockUsecase(ctrl)

			job := &jobImpl{
				logging: logging,
				tracer:  tf.Controller(),
				usecase: mockApp,
			}

			err := job.Execute(ctx, []string{"--active-only", "--inactive-only"})
			require.Error(t, err)
		})

		t.Run("同一フィルタフラグが重複指定された場合、CountUsersを呼ばずにエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			// CountUsers は呼ばれない（EXPECT 未設定）
			mockApp := mock_user.NewMockUsecase(ctrl)

			job := &jobImpl{
				logging: logging,
				tracer:  tf.Controller(),
				usecase: mockApp,
			}

			err := job.Execute(ctx, []string{"--active-only", "--active-only"})
			require.Error(t, err)
		})
	})
}
