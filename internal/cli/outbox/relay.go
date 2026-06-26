// Package outbox は、outbox relay の起動・グレースフルシャットダウンと replay 実行のコアロジックを提供します。
package outbox

import (
	"context"
	"time"
)

// RunRelay は、relay アプリを起動し、ctx のキャンセル（終了シグナル）を受けてから
// グレースフルシャットダウンを行います。停止用 context のタイムアウトは「停止開始時点」から
// 計測することで稼働時間に消費されないようにします。
func RunRelay(
	ctx context.Context,
	shutdownTimeout time.Duration,
	startApp func(context.Context) error,
	stopApp func(context.Context) error,
) error {
	if err := startApp(ctx); err != nil {
		return err
	}

	<-ctx.Done()

	stopCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	return stopApp(stopCtx) //nolint:contextcheck // 停止用 context は意図的に ctx を継承しない
}
