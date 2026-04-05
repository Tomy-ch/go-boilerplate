// Package di は、依存性注入（DI）コンテナの中心的な役割を果たす機能を提供します。
package di

import (
	"context"

	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/di/module"
	"go-boilerplate/internal/di/module/core"
	"go-boilerplate/internal/di/server"

	"go.uber.org/fx"
)

// NewApplicationServer は、サーバー関連の依存関係を提供するfx.Moduleです。
func NewApplicationServer(app *fx.App) (func(context.Context) error, func(context.Context) error) {
	start := func(ctx context.Context) error {
		return app.Start(ctx)
	}

	stop := func(stopCtx context.Context) error {
		return app.Stop(stopCtx)
	}

	return start, stop
}

// NewApplicationCore は、アプリケーションの fx.App インスタンスを作成します。
func NewApplicationCore() *fx.App {
	return fx.New(
		// Lifecycle Module
		lifecycle.Module(),
		// Config Module
		module.ConfigModule(),
		// Core Module
		core.IPRateLimiterModule(),
		core.ValidatorModule(),
		core.SecurityCookieModule(),
		core.AuthnModule(),
		core.BasicAuthModule(),
		core.SkipperModule(),
		// Common Module
		module.LoggingModule(),
		module.ObservabilityModule(),
		module.DatabaseModule(),
		module.SystemModule(),
		server.MiddlewareModule(),
		// DDD Modules
		module.InfrastructureModule(),
		module.UsecaseModule(),
		module.ControllerModule(),
		// Server Module
		server.Module(),
		server.HookModule(),
	)
}
