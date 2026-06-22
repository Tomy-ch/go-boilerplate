// Package worker は、worker 実行のオーケストレーション（常駐・グレースフルストップ）のコアロジックを提供します。
package worker

import (
	"context"
	"time"
)

const stopTimeout = 30 * time.Second

// StartFunc は、worker の開始関数の型です（DI から取得した開始関数を注入します）。
type StartFunc func(ctx context.Context, name string, args []string) <-chan error

// StopFunc は、worker の停止関数の型です（DI から取得した停止関数を注入します）。
type StopFunc func(ctx context.Context) error

// RunWorkerWith は、worker ランナーの取得元（provide）を差し替え可能にした上で runWorker へ委譲します。
func RunWorkerWith(
	ctx context.Context,
	name string,
	args []string,
	provide func() (StartFunc, StopFunc),
) error {
	start, stop := provide()
	return runWorker(ctx, name, args, start, stop)
}

// runWorker は、worker を起動して常駐させ、SIGTERM(ctx 完了) もしくは engine 自走停止で
// グレースフルに停止します。job と異なり完了 channel を待ち続ける常駐型です。
func runWorker(ctx context.Context, name string, args []string, start StartFunc, stop StopFunc) error {
	done := start(ctx, name, args)

	select {
	case err := <-done:
		// engine が自走停止（Fatal / unknown worker）。
		gracefulStop(stop)
		return err
	case <-ctx.Done():
		// SIGTERM。engine を drain して停止する。
		gracefulStop(stop)
		return nil
	}
}

// gracefulStop は、停止開始時点から stopTimeout の猶予を与えて後始末（app.Stop=drain）を実行します。
// ctx は SIGTERM で既にキャンセル済みのため、停止用 context は ctx を継承せず作り直します。
func gracefulStop(stop StopFunc) {
	stopCtx, cancel := context.WithTimeout(context.Background(), stopTimeout)
	defer cancel()
	_ = stop(stopCtx)
}
