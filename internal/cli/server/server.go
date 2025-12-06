// Package server は、サーバーを起動するためのコマンドを提供するためのパッケージです。
package server

import (
	"context"
	"os/signal"
	"syscall"

	"boilerplate-go/internal/config"
	server "boilerplate-go/internal/di"

	"github.com/spf13/cobra"
)

// NewServeCommand は、サーバーを起動するためのコマンドを生成します。
func NewServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "サーバーを起動します。",
		Long:  "このコマンドは、アプリケーションのサーバーを起動します。",
		RunE:  serveRun,
	}
}

// serveRun は、サーバーを起動するための実行関数です。
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

	stopCtx, cancel := context.WithTimeout(
		context.Background(),
		appCfg.ShutdownTimeout(),
	)
	defer cancel()

	// メトリクスサーバーの起動（本番環境では起動しない）
	if !appCfg.IsProductionMode() {
		startMetrics, stopMetrics := NewMetricsServer(mtcCfg)
		startMetrics()
		defer stopMetrics(stopCtx)
	}

	app := server.NewApplicationCore()
	startApp, stopApp := server.NewApplicationServer(app)
	if err = startApp(ctx); err != nil {
		return err
	}

	<-ctx.Done()

	return stopApp(stopCtx)
}
