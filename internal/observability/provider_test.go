package observability

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

func Test_TracerProvider(t *testing.T) {
	t.Parallel()

	t.Run("fx.Lifecycleにシャットダイン処理を登録し、グローバルプロバイダーのTracerProviderを返す", func(t *testing.T) {
		var tp trace.TracerProvider

		app := fx.New(
			fx.Invoke(func(lc fx.Lifecycle) {
				tp = TracerProvider(lc)
			}),
		)

		ctx := context.Background()
		require.NoError(t, app.Start(ctx))
		defer func() { _ = app.Stop(ctx) }()

		require.NotNil(t, tp)

		_, ok := tp.(*sdktrace.TracerProvider)
		require.True(t, ok)

		gp := otel.GetTracerProvider()
		require.Equal(t, tp, gp)
	})

	t.Run("シャットダウン時にエラーが発生した場合、そのエラーを返す", func(t *testing.T) {
		app := fx.New(
			fx.Invoke(func(lc fx.Lifecycle) {
				_ = TracerProvider(lc)
			}),
		)

		ctx := context.Background()
		require.NoError(t, app.Start(ctx))

		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		require.NoError(t, app.Stop(stopCtx))
	})
}
