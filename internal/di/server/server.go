// Package server は、Echoサーバーの初期化と設定を提供します。
package server

import (
	"boilerplate-go/internal/controller/server"
	"boilerplate-go/internal/di/server/extension"
	"boilerplate-go/internal/di/server/extension/decoration"
	"boilerplate-go/internal/di/server/extension/inbound"
	"boilerplate-go/internal/di/server/extension/instrumentation"
	"boilerplate-go/internal/di/server/extension/nonprod"
	"boilerplate-go/internal/di/server/extension/outbound"
	"boilerplate-go/internal/di/server/extension/security"
	"boilerplate-go/internal/di/server/hook"

	"go.uber.org/fx"
)

// Module は、サーバー関連の依存関係を提供するfx.Moduleです。
func Module() fx.Option {
	return fx.Module("server",
		fx.Provide(
			server.NewAppServer,
		),
	)
}

// HookModule は、サーバーのライフサイクルフックを提供するfx.Moduleです。
func HookModule() fx.Option {
	return fx.Module("server.hook",
		fx.Invoke(
			hook.RegisterHTTPServerHooks,
			hook.RegisterRateLimitHooks,
		),
	)
}

// MiddlewareModule は、HTTP スタック関連の依存関係を提供するfx.Moduleです。
func MiddlewareModule() fx.Option {
	return fx.Module("httpstack",
		// Middleware Modules
		decoration.BannerModule(),
		decoration.DefaultPortModule(),
		decoration.DefaultPortModule(),
		inbound.IPExtractorModule(),
		inbound.URIModule(),
		inbound.OpenAPIModule(),
		outbound.ErrorHandlerModule(),
		outbound.ForceJSONModule(),
		outbound.RecoveryModule(),
		security.Module(),
		security.CORSModule(),
		security.CookieModule(),
		security.RateLimitModule(),
		instrumentation.RequestIDModule(),
		instrumentation.LoggingModule(),
		instrumentation.ObservabilityModule(),
		nonprod.DebugModeModule(),
		fx.Provide(
			extension.ApplyExtends,
		),
	)
}
