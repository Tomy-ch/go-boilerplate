// Package server は、Echoサーバーの初期化と設定を提供します。
package server

import (
	"go-boilerplate/internal/controller/server"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/di/server/extension/inbound"
	"go-boilerplate/internal/di/server/extension/instrumentation"
	"go-boilerplate/internal/di/server/extension/nonprod"
	"go-boilerplate/internal/di/server/extension/outbound"
	"go-boilerplate/internal/di/server/extension/security"
	"go-boilerplate/internal/di/server/hook"

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
		),
	)
}

// MiddlewareModule は、HTTP スタック関連の依存関係を提供するfx.Moduleです。
func MiddlewareModule() fx.Option {
	return fx.Module("server.httpstack",
		// Middleware Modules
		inbound.IPExtractorModule(),
		inbound.URIModule(),
		inbound.TimeoutModule(),
		inbound.OpenAPIModule(),
		outbound.ErrorHandlerModule(),
		outbound.ForceJSONModule(),
		outbound.RecoveryModule(),
		security.Module(),
		security.CORSModule(),
		security.CookieModule(),
		instrumentation.RequestIDModule(),
		instrumentation.LoggingModule(),
		instrumentation.ObservabilityModule(),
		nonprod.DebugModeModule(),
		fx.Provide(
			extension.ApplyExtends,
		),
	)
}
