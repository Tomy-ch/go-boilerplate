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
	// サーバー起動に必要な設定をまとめて読み込みます。
	cfg, err := config.SetUpConfig()
	if err != nil {
		return err
	}
	appCfg := config.NewApplicationConfig(cfg)
	mtcCfg := config.NewMetricsConfig(cfg)

	// SIGINT / SIGTERM を受け取ったら、アプリ全体の停止処理へ移行します。
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	// メトリクス用の補助サーバーは開発向けのため、本番環境では起動しません（判定はコア側）。
	stopMetrics := server.ResolveMetricsStop(appCfg, func() (func(), func(context.Context)) {
		return server.NewMetricsServer(mtcCfg)
	})

	// DI コンテナ経由でアプリ本体を組み立て、HTTP サーバーを起動します。
	app := di.NewApplicationCore()
	startApp, stopApp := di.NewApplicationServer(app)

	return server.RunServer(ctx, appCfg.ShutdownTimeout(), startApp, stopApp, stopMetrics)
}
