// Package job は、ジョブを管理・実行するためのコマンドを提供するためのパッケージです。
package job

import (
	"context"
	"time"

	"go-boilerplate/internal/di"

	"github.com/spf13/cobra"
)

const stopTimeout = 30 * time.Second

// NewCommand は job コマンドを生成します。
func NewCommand() *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "job",
		Short: "job <job-name> [args...] コマンドは、指定されたジョブを実行します。",
		Long: "job <job-name> [args...] コマンドは、指定されたジョブを実行します。ジョブ名と引数を指定して実行してください。\n" +
			"例: job usercount --timeout 30s",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJobExec(cmd.Context(), args[0], args[1:], timeout)
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 0, "job execution timeout duration (e.g., 30s, 1m)")

	return cmd
}

// runJobExec は、DI 経由でジョブランナーを取得し、オーケストレーションを runJob へ委譲します。
func runJobExec(ctx context.Context, name string, args []string, timeout time.Duration) error {
	start, stop := di.RunJob()
	return runJob(ctx, name, args, timeout, start, stop)
}

// runJob は、ジョブ実行のオーケストレーション（タイムアウト分岐と停止処理）を行います。
func runJob(
	ctx context.Context,
	name string,
	args []string,
	timeout time.Duration,
	start di.StartFunc,
	stop di.StopFunc,
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
		// タイムアウト時も親 ctx は生きているため、停止専用 context を作り直して後始末に猶予を与えます
		// （期限切れの waitCtx を渡すと後始末が即時打ち切られる）。
		gracefulStop(ctx, stop)
		return waitCtx.Err()
	case <-ctx.Done():
		// 親 context のキャンセル時も、停止処理だけは流してジョブを終了させます。
		gracefulStop(ctx, stop)
		return ctx.Err()
	}
}

// gracefulStop は、停止開始時点から stopTimeout の猶予を与えて後始末（app.Stop）を実行します。
func gracefulStop(ctx context.Context, stop di.StopFunc) {
	stopCtx, cancel := context.WithTimeout(ctx, stopTimeout)
	defer cancel()
	_ = stop(stopCtx)
}
