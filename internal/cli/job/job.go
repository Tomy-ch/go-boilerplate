// Package job は、ジョブ実行のオーケストレーション（タイムアウト分岐・停止処理）のコアロジックを提供します。
package job

import (
	"context"
	"time"
)

const stopTimeout = 30 * time.Second

// StartFunc は、ジョブの開始関数の型です（DI から取得した開始関数を注入します）。
type StartFunc func(ctx context.Context, name string, args []string) <-chan error

// StopFunc は、ジョブの停止関数の型です（DI から取得した停止関数を注入します）。
type StopFunc func(ctx context.Context) error

// RunJobWith は、ジョブランナーの取得元（provide）を差し替え可能にした上で runJob へ委譲します。
func RunJobWith(
	ctx context.Context,
	name string,
	args []string,
	timeout time.Duration,
	provide func() (StartFunc, StopFunc),
) error {
	start, stop := provide()
	return runJob(ctx, name, args, timeout, start, stop)
}

// runJob は、ジョブ実行のオーケストレーション（タイムアウト分岐と停止処理）を行います。
func runJob(
	ctx context.Context,
	name string,
	args []string,
	timeout time.Duration,
	start StartFunc,
	stop StopFunc,
) error {
	done := start(ctx, name, args)

	if timeout <= 0 {
		// タイムアウト未指定時は、ジョブ完了を待ってから停止処理だけ確実に流します。
		err := <-done
		gracefulStop(ctx, stop)
		return err
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case err := <-done:
		// 正常終了。
		gracefulStop(ctx, stop)
		return err
	case <-waitCtx.Done():
		// waitCtx は ctx の子。タイムアウト(DeadlineExceeded)・親キャンセル(Canceled)の両方で発火する。
		// 停止は期限切れの waitCtx ではなく専用 context を作り直して猶予を与える。
		gracefulStop(ctx, stop)
		return waitCtx.Err()
	}
}

// gracefulStop は、停止開始時点から stopTimeout の猶予を与えて後始末（app.Stop）を実行します。
func gracefulStop(ctx context.Context, stop StopFunc) {
	stopCtx, cancel := context.WithTimeout(ctx, stopTimeout)
	defer cancel()
	_ = stop(stopCtx)
}
