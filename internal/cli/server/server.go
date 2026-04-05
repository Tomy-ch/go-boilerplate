// Package server は、サーバーを起動するためのコマンドを提供するためのパッケージです。
package server

import (
	"context"
	"os/signal"
	"syscall"

	"go-boilerplate/internal/config"
	server "go-boilerplate/internal/di"

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

	// 停止処理は無期限に待たず、設定されたタイムアウト内で完了させます。
	stopCtx, cancel := context.WithTimeout(
		context.Background(),
		appCfg.ShutdownTimeout(),
	)
	defer cancel()

	// メトリクス用の補助サーバーは開発向けのため、本番環境では起動しません。
	if !appCfg.IsProductionMode() {
		startMetrics, stopMetrics := NewMetricsServer(mtcCfg)
		startMetrics()
		defer stopMetrics(stopCtx)
	}

	// DI コンテナ経由でアプリ本体を組み立て、HTTP サーバーを起動します。
	app := server.NewApplicationCore()
	startApp, stopApp := server.NewApplicationServer(app)
	if err = startApp(ctx); err != nil {
		return err
	}

	// 終了シグナルを受け取るまで待機し、その後グレースフルシャットダウンを行います。
	<-ctx.Done()

	return stopApp(stopCtx)
}
