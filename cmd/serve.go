package main

import (
	"context"
	"os/signal"
	"syscall"

	"go-boilerplate/internal/cli/server"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di"

	"github.com/spf13/cobra"
)

// newServeCommand は、サーバーを起動するためのコマンドを生成します。
func newServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "サーバーを起動します。",
		Long:  "このコマンドは、アプリケーションのサーバーを起動します。",
		RunE:  serveRun,
	}
}

// serveRun は、設定読込・シグナル・DI アプリ生成を結線し、server.RunServer へ委譲する薄い殻です。
func serveRun(_ *cobra.Command, _ []string) error {
	cfg, err := config.SetUpConfig()
	if err != nil {
		return err
	}
	appCfg := config.NewApplicationConfig(cfg)
	mtcCfg := config.NewMetricsConfig(cfg)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	// 本番モードでない場合のみメトリクス補助サーバーを起動する（判定はコア側）。
	stopMetrics := server.ResolveMetricsStop(appCfg, func() (func(), func(context.Context)) {
		return server.NewMetricsServer(mtcCfg)
	})

	app := di.NewApplicationCore()
	startApp, stopApp := di.NewApplicationServer(app)

	return server.RunServer(ctx, appCfg.ShutdownTimeout(), startApp, stopApp, stopMetrics)
}
