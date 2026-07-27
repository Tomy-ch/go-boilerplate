// Package server は、サーバー起動・グレースフルシャットダウンのコアロジックを提供します。
package server

import (
	"context"
	"time"

	"go-boilerplate/internal/config"
)

// StartMetricsAndResolveStop は、本番モードでない場合のみ補助メトリクスサーバーを起動し、その停止関数を返します。
// 本番モードでは起動せず nil を返します（呼び出し側は nil を「停止不要」として扱います）。
func StartMetricsAndResolveStop(appCfg *config.ApplicationConfig, newMetrics func() (func(), func(context.Context))) func(context.Context) {
	if appCfg.IsProductionMode() {
		return nil
	}
	startMetrics, stopMetrics := newMetrics()
	startMetrics()
	return stopMetrics
}

// RunServer は, アプリ本体を起動し、ctx のキャンセル（終了シグナル）を受けてから
// グレースフルシャットダウンを行います。停止用 context のタイムアウトは「停止開始時点」から
// 計測することで稼働時間に消費されないようにしています。
func RunServer(
	ctx context.Context,
	shutdownTimeout time.Duration,
	startApp func(context.Context) error,
	stopApp func(context.Context) error,
	stopMetrics func(context.Context),
) error {
	if err := startApp(ctx); err != nil {
		// 起動に失敗しても、既に起動済みの補助メトリクスサーバーは停止する（確保と解放の対称性）。
		if stopMetrics != nil {
			stopCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
			defer cancel()
			stopMetrics(stopCtx) //nolint:contextcheck // 停止用 context は意図的に ctx を継承しない
		}
		return err
	}

	<-ctx.Done()

	// 停止処理は無期限に待たず、シャットダウン開始時点から設定タイムアウト内で完了させます。
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
