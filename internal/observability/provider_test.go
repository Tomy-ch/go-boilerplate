package observability

import (
	"context"
	"testing"

	mock_lifecycle "go-boilerplate/internal/di/lifecycle/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/mock/gomock"
)

func Test_TracerProvider(t *testing.T) {
	// otel.SetTracerProvider / otel.GetTracerProvider をグローバルに触るため Parallel 不可。

	t.Run("正常系", func(t *testing.T) {
		t.Run("ShutdownRegistrarにシャットダウン処理を登録し、グローバルプロバイダーのTracerProviderを返す", func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockReg := mock_lifecycle.NewMockRegistrar(ctrl)

			var shutdownFunc func(context.Context) error
			dummy := func(context.Context) error { return nil }
			mockReg.EXPECT().RegisterStop(gomock.AssignableToTypeOf(dummy)).Do(func(args ...any) {
				shutdownFunc = args[0].(func(context.Context) error)
			}).Times(1)

			tp := TracerProvider(mockReg)

			require.NotNil(t, tp)
			_, ok := tp.(*sdktrace.TracerProvider)
			assert.True(t, ok)

			gp := otel.GetTracerProvider()
			assert.Equal(t, tp, gp)

			require.NotNil(t, shutdownFunc)
			require.NoError(t, shutdownFunc(context.Background()))
		})

		t.Run("シャットダウン時にコンテキストがキャンセルされてもエラーを返さない", func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockReg := mock_lifecycle.NewMockRegistrar(ctrl)
			var shutdownFunc func(context.Context) error
			dummy := func(context.Context) error { return nil }
			mockReg.EXPECT().RegisterStop(gomock.AssignableToTypeOf(dummy)).Do(func(args ...any) {
				shutdownFunc = args[0].(func(context.Context) error)
			}).Times(1)

			_ = TracerProvider(mockReg)
			require.NotNil(t, shutdownFunc)

			// キャンセル済みコンテキストを渡しても Shutdown は内部で握り潰しエラーを返さない。
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err := shutdownFunc(ctx)
			require.NoError(t, err)
		})
	})
}
