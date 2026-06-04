package outbound

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/logging"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

// requireProvidesOne は、与えたモジュール群が指定 group に要素をちょうど1件 provide することを検証する。
// fx.NopLogger は構成時のログ出力を抑えるだけで、検証結果には影響しない。
func requireProvidesOne[T any](t *testing.T, group string, opts ...fx.Option) {
	t.Helper()

	var got []T
	app := fx.New(append(opts,
		fx.Populate(fx.Annotate(&got, fx.ParamTags(`group:"`+group+`"`))),
		fx.NopLogger,
	)...)

	require.NoError(t, app.Start(context.Background()))
	defer func() { require.NoError(t, app.Stop(context.Background())) }()

	require.Len(t, got, 1)
}

func TestErrorHandlerModule_ProvidesServeConfig(t *testing.T) {
	t.Parallel()
	requireProvidesOne[extension.SrvCfg](t, "server.configurators",
		ErrorHandlerModule(),
		fx.Provide(func() logging.Logger { return logging.NewTestLogger(t) }),
		fx.Provide(func() logging.LogFieldBuilder { return logging.NewTestLogFieldBuilder(t) }),
		fx.Supply(config.NewObservabilityConfig(config.MockConfigForTest(t))),
	)
}

func TestForceJSONModule_ProvidesUseMiddleware(t *testing.T) {
	t.Parallel()
	requireProvidesOne[extension.UseMiddleware](t, "middlewares.use", ForceJSONModule())
}

func TestRecoveryModule_ProvidesUseMiddleware(t *testing.T) {
	t.Parallel()
	requireProvidesOne[extension.UseMiddleware](t, "middlewares.use",
		RecoveryModule(),
		fx.Provide(func() logging.Logger { return logging.NewTestLogger(t) }),
		fx.Provide(func() logging.LogFieldBuilder { return logging.NewTestLogFieldBuilder(t) }),
		fx.Supply(config.NewApplicationConfig(config.MockConfigForTest(t))),
	)
}
