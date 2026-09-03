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

// NewApplicationServer は、組み立て済みの fx.App を受け取り、その起動／停止を行う lifecycle 関数（start, stop）を返します。
func NewApplicationServer(app *fx.App) (func(context.Context) error, func(context.Context) error) {
	start := func(ctx context.Context) error {
		return app.Start(ctx)
	}

	stop := func(ctx context.Context) error {
		return app.Stop(ctx)
	}

	return start, stop
}

// applicationCoreOptions は、アプリケーションコアを構成する fx.Option 群を返します。
func applicationCoreOptions() []fx.Option {
	return append(serveBaseOptions(), serveRealtimeOptions()...)
}

// serveRealtimeOptions は、Realtime Delivery の受信側 runtime を結線します。
// feature の realtime adapter を消すと結線の 1 行も消えて空を返すようになり、serve graph から
// Realtime と AWS / DynamoDB のクライアントが丸ごと外れます
// （"Zero adapters, zero runtime"。docs/design/realtime-delivery.md §3.1）。
// append で書くのは、マーカー行が落ちた後も関数がそのまま成り立つようにするためです。
func serveRealtimeOptions() []fx.Option {
	opts := make([]fx.Option, 0, 1)
	opts = append(opts, module.ServeRealtimeModule()) // sample-api:line

	return opts
}

// serveBaseOptions は、Realtime Delivery を含まないアプリケーションコアです。
func serveBaseOptions() []fx.Option {
	return []fx.Option{
		// Lifecycle Module
		lifecycle.Module(),
		// Config Module
		module.ConfigModule(),
		// Core Module
		core.ValidatorModule(),
		core.RedactionModule(),
		core.SecurityCookieModule(),
		core.AuthnModule(),
		core.BasicAuthModule(),
		core.SkipperModule(),
		// Common Module
		module.LoggingModule(),
		module.ObservabilityModule(),
		module.DatabaseModule(),
		module.SystemModule(),
		// DDD Modules
		module.InfrastructureModule(),
		module.UsecaseModule(),
		module.ControllerModule(),
		// Server Module
		server.MiddlewareModule(),
		server.Module(),
		server.HookModule(),
	}
}

// NewApplicationCore は、アプリケーションの fx.App インスタンスを作成します。
// extra はテスト時に依存を差し替える（fx.Replace / fx.Decorate）ための seam で、本番呼び出しでは渡しません。
func NewApplicationCore(extra ...fx.Option) *fx.App {
	opts := append(applicationCoreOptions(), fx.WithLogger(NewFxEventLogger))
	return fx.New(append(opts, extra...)...)
}
