package module

import (
	"testing"

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
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
