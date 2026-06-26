// Package job は、ジョブ実行のオーケストレーション（タイムアウト分岐・停止処理）のコアロジックを提供します。
package job

import (
	"context"
	"time"

	"go-boilerplate/pkg/xerrors"
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
		err := <-done
		return xerrors.Join(err, gracefulStop(ctx, stop))
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case err := <-done:
		return xerrors.Join(err, gracefulStop(ctx, stop))
	case <-waitCtx.Done():
		// waitCtx は ctx の子。タイムアウト(DeadlineExceeded)・親キャンセル(Canceled)の両方で発火する。
		// 停止は期限切れの waitCtx ではなく専用 context を作り直して猶予を与える。
		return xerrors.Join(waitCtx.Err(), gracefulStop(ctx, stop))
	}
}

// gracefulStop は、停止開始時点から stopTimeout の猶予を与えて後始末（app.Stop）を実行し、
// その結果を返します（H3）。停止失敗（OTel flush 失敗・DB pool close 失敗等）は呼出側が
// 本体結果と xerrors.Join し、Execute 側の os.Exit 判定で拾えるようにします。
// xerrors.Join は nil を畳むため、停止成功時は本体結果のみが返ります。
func gracefulStop(ctx context.Context, stop StopFunc) error {
	stopCtx, cancel := context.WithTimeout(ctx, stopTimeout)
	defer cancel()
	return stop(stopCtx)
}
