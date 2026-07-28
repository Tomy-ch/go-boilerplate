package module

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"go-boilerplate/internal/di/module/core"
	"go-boilerplate/internal/di/server"
)

func TestControllerModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// コントローラ層は OpenAPI ハンドラ追加で増える領域。個々のハンドラの入出力は controller 層の
	// テストに任せ、ここでは BindHandler 群が依存（ユースケース・echo・ミドルウェア等）と
	// 正しく結線されることを確認する。
	opts := append(commonDeps(),
		core.ValidatorModule(), core.SecurityCookieModule(),
		core.AuthnModule(), core.BasicAuthModule(), core.SkipperModule(),
		InfrastructureModule(), UsecaseModule(),
		server.MiddlewareModule(), server.Module(), server.HookModule(),
		ControllerModule(),
	)
	validateGraph(t, opts...)
}

func TestControllerModule(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("HTTPサーバ未配線ではハンドラ登録が解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			// BindHandler 群が echo を要求することの裏返し。登録が空になっても
			// 気付けない状態（invoke を持たないモジュール）への退行をここで検出する。
			opts := append(commonDeps(), InfrastructureModule(), UsecaseModule(),
				ControllerModule(), fx.NopLogger)

			require.Error(t, fx.ValidateApp(opts...))
		})

		t.Run("ユースケース未配線ではハンドラ登録が解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			opts := append(commonDeps(),
				core.ValidatorModule(), core.SecurityCookieModule(),
				core.AuthnModule(), core.BasicAuthModule(), core.SkipperModule(),
				InfrastructureModule(),
				server.MiddlewareModule(), server.Module(), server.HookModule(),
				ControllerModule(), fx.NopLogger,
			)

			require.Error(t, fx.ValidateApp(opts...))
		})
	})
}
