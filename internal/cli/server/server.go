// Package server は、サーバーを起動するためのコマンドを提供するためのパッケージです。
package server

import (
	"context"
	"os/signal"
	"syscall"
	"time"

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
	stopMetrics := resolveMetricsStop(appCfg, func() (func(), func(context.Context)) {
		return NewMetricsServer(mtcCfg)
	})

	// DI コンテナ経由でアプリ本体を組み立て、HTTP サーバーを起動します。
	app := server.NewApplicationCore()
	startApp, stopApp := server.NewApplicationServer(app)

	return runServer(ctx, appCfg.ShutdownTimeout(), startApp, stopApp, stopMetrics)
}

// resolveMetricsStop は、本番モードでない場合のみ補助メトリクスサーバーを起動し、その停止関数を返します。
// 本番モードでは起動せず nil を返します（呼び出し側は nil を「停止不要」として扱います）。
func resolveMetricsStop(appCfg *config.ApplicationConfig, newMetrics func() (func(), func(context.Context))) func(context.Context) {
	if appCfg.IsProductionMode() {
		return nil
	}
	startMetrics, stopMetrics := newMetrics()
	startMetrics()
	return stopMetrics
}

// runServer は、アプリ本体を起動し、ctx のキャンセル（終了シグナル）を受けてから
// グレースフルシャットダウンを行います。停止用 context のタイムアウトは「停止開始時点」から
// 計測することで稼働時間に消費されないようにしています。
func runServer(
	ctx context.Context,
	shutdownTimeout time.Duration,
	startApp func(context.Context) error,
	stopApp func(context.Context) error,
	stopMetrics func(context.Context),
) error {
	if err := startApp(ctx); err != nil {
		return err
	}

	// 終了シグナルを受け取るまで待機し、その後グレースフルシャットダウンを行います。
	<-ctx.Done()

	// 停止処理は無期限に待たず、シャットダウン開始時点から設定タイムアウト内で完了させます。
	// タイムアウトを起動直後ではなくこの時点で計測することで、稼働時間に消費されないようにします。
	stopCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// メトリクスサーバーとアプリ本体を、同じシャットダウン用 context で順に停止します。
	// 停止用 context は ctx を継承しない（ctx は既にキャンセル済みのため、継承すると即時に
	// 期限切れになり停止猶予が無くなる）。これは意図した設計なので contextcheck を抑制する。
	if stopMetrics != nil {
		stopMetrics(stopCtx) //nolint:contextcheck // 停止用 context は意図的に ctx を継承しない
	}

	return stopApp(stopCtx) //nolint:contextcheck // 停止用 context は意図的に ctx を継承しない
}
