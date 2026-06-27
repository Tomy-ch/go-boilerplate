// Package job は、ジョブ実行のオーケストレーション（タイムアウト分岐・停止処理）のコアロジックを提供します。
package job

import (
	"context"
	"time"

	"go-boilerplate/pkg/xerrors"
)

// StartFunc は、ジョブの開始関数の型です（DI から取得した開始関数を注入します）。
type StartFunc func(ctx context.Context, name string, args []string) <-chan error

// StopFunc は、ジョブの停止関数の型です（DI から取得した停止関数を注入します）。
type StopFunc func(ctx context.Context) error

// RunJobWith は、ジョブランナーの取得元（provide）を差し替え可能にした上で runJob へ委譲します。
// grace（APP_SHUTDOWN_TIMEOUT）は停止猶予の単一軸で、停止 context の deadline に用います。
func RunJobWith(
	ctx context.Context,
	name string,
	args []string,
	timeout time.Duration,
	grace time.Duration,
	provide func() (StartFunc, StopFunc),
) error {
	start, stop := provide()
	return runJob(ctx, name, args, timeout, grace, start, stop)
}

// runJob は、ジョブ実行のオーケストレーション（タイムアウト分岐と停止処理）を行います。
func runJob(
	ctx context.Context,
	name string,
	args []string,
	timeout time.Duration,
	grace time.Duration,
	start StartFunc,
	stop StopFunc,
) error {
	done := start(ctx, name, args)

	if timeout <= 0 {
		err := <-done
		return xerrors.Join(err, gracefulStop(ctx, grace, stop))
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case err := <-done:
		return xerrors.Join(err, gracefulStop(ctx, grace, stop))
	case <-waitCtx.Done():
		// waitCtx は ctx の子。タイムアウト(DeadlineExceeded)・親キャンセル(Canceled)の両方で発火する。
		// 停止は期限切れの waitCtx ではなく専用 context を作り直して猶予を与える。
		return xerrors.Join(waitCtx.Err(), gracefulStop(ctx, grace, stop))
	}
}

// gracefulStop は、親キャンセルに左右されない grace の猶予を停止処理（app.Stop）に与え、
// その結果を返します。SIGINT 伝播や親 ctx タイムアウト後でも OTel flush / DB pool close に
// grace の全猶予が保証されます。停止失敗を呼び出し元のエラーチェーンに含め exit code へ
// 反映できるよう、エラーは破棄せず返します。
func gracefulStop(ctx context.Context, grace time.Duration, stop StopFunc) error {
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), grace)
	defer cancel()
	return stop(stopCtx)
}
