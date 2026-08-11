// Package job は、ジョブ実行のオーケストレーション（タイムアウト分岐・停止処理）のコアロジックを提供します。
package job

import (
	"context"
	"time"

	"go-boilerplate/pkg/xerrors"
)

// StartFunc は、ジョブを起動してエラーチャネルを返す関数の型です。
type StartFunc func(ctx context.Context, name string, args []string) <-chan error

// StopFunc は、ジョブをグレースフルに停止する関数の型です。
type StopFunc func(ctx context.Context) error

// RunJobWith は、provide が返す開始・停止関数でジョブを実行します。
// grace は停止 context の deadline に用います。
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

// gracefulStop は、親キャンセルから切り離した grace の猶予を停止処理に与え、その結果を返します。
// 親 ctx が --timeout で期限切れになった後でも grace の全猶予が停止処理に保証されます。停止失敗を
// 呼び出し元のエラーチェーン（＝exit code）に反映できるよう、エラーは破棄せず返します。
func gracefulStop(ctx context.Context, grace time.Duration, stop StopFunc) error {
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), grace)
	defer cancel()
	return stop(stopCtx)
}
