// Package worker は、worker 実行のオーケストレーション（常駐・グレースフルストップ）のコアロジックを提供します。
package worker

import (
	"context"
	"time"
)

const stopTimeout = 30 * time.Second

// StartFunc は、worker を起動して完了チャネルを返す関数の型です。
type StartFunc func(ctx context.Context, name string, args []string) <-chan error

// StopFunc は、worker をグレースフルに停止する関数の型です。
type StopFunc func(ctx context.Context) error

// RunWorkerWith は、provide が返す開始・停止関数で worker を実行します。
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

	var (
		runErr      error
		selfStopped bool
	)
	select {
	case runErr = <-done:
		// engine が自走停止（Fatal / unknown worker）。done は消費済み。
		selfStopped = true
	case <-ctx.Done():
		// SIGTERM。engine を drain して停止し、停止後に結果を確認する。
	}

	gracefulStop(ctx, stop)

	// SIGTERM 経路では engine の実際の終了結果を必ず待ち切る。
	// stopTimeout が engine の drain 完了より先に満了して OnStop が早期 return しても、
	// engine goroutine は drain 後に必ず done へ結果を書く（DrainTimeout で有界）。
	// 非ブロッキングで default を取ると、その間に発生した Fatal を取りこぼし exit 0 になってしまう。
	if !selfStopped {
		runErr = <-done
	}
	return runErr
}

// gracefulStop は、停止開始時点から stopTimeout の猶予を与えて後始末（app.Stop=drain）を実行します。
// ctx は SIGTERM で既にキャンセル済みのため、停止用 context はキャンセルだけ切り離して
// （trace/baggage は引き継ぎつつ）作り直します。
func gracefulStop(ctx context.Context, stop StopFunc) {
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), stopTimeout)
	defer cancel()
	_ = stop(stopCtx)
}
