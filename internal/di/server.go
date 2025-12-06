// Package di は、依存性注入（DI）コンテナの中心的な役割を果たす機能を提供します。
package di

import (
	"context"

	"boilerplate-go/internal/di/lifecycle"
	"boilerplate-go/internal/di/module"
	"boilerplate-go/internal/di/module/core"
	"boilerplate-go/internal/di/server"

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
		// Core Module
		lifecycle.Module(),
		module.ConfigModule(),
		module.LoggingModule(),
		module.ObservabilityModule(),
		module.DatabaseModule(),
		module.SystemModule(),
		core.ValidatorModule(),
		server.HTTPStackModule(),
		// DDD Modules
		module.InfrastructureModule(),
		module.UsecaseModule(),
		module.ControllerModule(),
		// Server Module
		server.ServeModule(),
	)
}
