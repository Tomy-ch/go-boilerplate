package hook

import (
	"context"
	"errors"
	"testing"

	mock_lifecycle "go-boilerplate/internal/di/lifecycle/mock"
	mock_observability "go-boilerplate/internal/observability/mock"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRegisterObservabilityShutdownHooks(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ProviderShutdownerのShutdownをStopフックとして登録する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			reg := mock_lifecycle.NewMockRegistrar(ctrl)
			shutdowner := mock_observability.NewMockProviderShutdowner(ctrl)

			var stopFn func(context.Context) error
			dummy := func(context.Context) error { return nil }
			reg.EXPECT().RegisterStop(gomock.AssignableToTypeOf(dummy)).Do(func(args ...any) {
				stopFn = args[0].(func(context.Context) error)
			}).Times(1)

			RegisterObservabilityShutdownHooks(reg, shutdowner)

			// 登録された Stop フックが ProviderShutdowner.Shutdown へ委譲されること。
			require.NotNil(t, stopFn)
			shutdowner.EXPECT().Shutdown(gomock.Any()).Return(nil).Times(1)
			require.NoError(t, stopFn(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ShutdownがエラーのときStopフックもそのエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			reg := mock_lifecycle.NewMockRegistrar(ctrl)
			shutdowner := mock_observability.NewMockProviderShutdowner(ctrl)

			var stopFn func(context.Context) error
			dummy := func(context.Context) error { return nil }
			reg.EXPECT().RegisterStop(gomock.AssignableToTypeOf(dummy)).Do(func(args ...any) {
				stopFn = args[0].(func(context.Context) error)
			}).Times(1)

			RegisterObservabilityShutdownHooks(reg, shutdowner)

			require.NotNil(t, stopFn)
			wantErr := errors.New("shutdown failed")
			shutdowner.EXPECT().Shutdown(gomock.Any()).Return(wantErr).Times(1)
			require.ErrorIs(t, stopFn(context.Background()), wantErr)
		})
	})
}
