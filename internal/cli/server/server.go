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

	// メトリクス用の補助サーバーは開発向けのため、本番環境では起動しません。
	// 停止関数はシャットダウン段で再利用するため、関数スコープで保持します。
	var stopMetrics func(context.Context)
	if !appCfg.IsProductionMode() {
		startMetrics, endMetrics := NewMetricsServer(mtcCfg)
		startMetrics()
		stopMetrics = endMetrics
	}

	// DI コンテナ経由でアプリ本体を組み立て、HTTP サーバーを起動します。
	app := server.NewApplicationCore()
	startApp, stopApp := server.NewApplicationServer(app)
	if err = startApp(ctx); err != nil {
		return err
	}

	// 終了シグナルを受け取るまで待機し、その後グレースフルシャットダウンを行います。
	<-ctx.Done()

	// 停止処理は無期限に待たず、シャットダウン開始時点から設定タイムアウト内で完了させます。
	// タイムアウトを起動直後ではなくこの時点で計測することで、稼働時間に消費されないようにします。
	stopCtx, cancel := context.WithTimeout(
		context.Background(),
		appCfg.ShutdownTimeout(),
	)
	defer cancel()

	// メトリクスサーバーとアプリ本体を、同じシャットダウン用 context で順に停止します。
	if stopMetrics != nil {
		stopMetrics(stopCtx)
	}

	return stopApp(stopCtx)
}
