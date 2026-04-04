package observability

import (
	"context"
	"testing"

	mock_lifecycle "go-boilerplate/internal/di/lifecycle/mock"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/mock/gomock"
)

func Test_TracerProvider(t *testing.T) {
	t.Run("ShutdownRegistrarにシャットダウン処理を登録し、グローバルプロバイダーのTracerProviderを返す", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mockReg := mock_lifecycle.NewMockRegistrar(ctrl)

		var shutdownFunc func(context.Context) error
		// キャプチャされた関数を保存
		dummy := func(context.Context) error { return nil }
		mockReg.EXPECT().RegisterStop(gomock.AssignableToTypeOf(dummy)).Do(func(args ...any) {
			shutdownFunc = args[0].(func(context.Context) error)
		}).Times(1)

		tp := TracerProvider(mockReg)

		require.NotNil(t, tp)
		_, ok := tp.(*sdktrace.TracerProvider)
		require.True(t, ok)

		gp := otel.GetTracerProvider()
		require.Equal(t, tp, gp)

		require.NotNil(t, shutdownFunc)
		// 正常系: コンテキストが有効ならエラーは返らない
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

		// キャンセル済みコンテキストを渡すと Shutdown はエラーを返すはず
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := shutdownFunc(ctx)
		require.NoError(t, err)
	})
}
