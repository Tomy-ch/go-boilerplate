// Package job は、ジョブを管理・実行するためのコマンドを提供するためのパッケージです。
package job

import (
	"context"
	"time"

	"boilerplate-go/internal/di"

	"github.com/spf13/cobra"
)

const stopTimeout = 30 * time.Second

// timeOut は、ジョブ実行のタイムアウト時間を表します。
var timeOut time.Duration

// NewCommand は job コマンドを生成します。
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job",
		Short: "job <job-name> [args...] コマンドは、指定されたジョブを実行します。",
		Long: "job <job-name> [args...] コマンドは、指定されたジョブを実行します。ジョブ名と引数を指定して実行してください。\n" +
			"例: job usercount --timeout 30s",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobName := args[0]
			jobArgs := args[1:]
			return runJobExec(cmd.Context(), jobName, jobArgs)
		},
	}

	cmd.Flags().DurationVar(&timeOut, "timeout", 0, "job execution timeout duration (e.g., 30s, 1m)")

	return cmd
}

// runJobExec は、指定されたジョブを実行します。
func runJobExec(ctx context.Context, name string, args []string) error {
	start, stop := di.RunJob()

	done := start(ctx, name, args)

	if timeOut <= 0 {
		err := <-done
		stopCtx, cancel := context.WithTimeout(ctx, stopTimeout)
		defer cancel()

		_ = stop(stopCtx)
		return err
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeOut)
	defer cancel()

	select {
	case err := <-done:
		stopCtx, timeoutCancel := context.WithTimeout(ctx, stopTimeout)
		defer timeoutCancel()
		_ = stop(stopCtx)
		return err
	case <-waitCtx.Done():
		_ = stop(waitCtx)
		return waitCtx.Err()
	case <-ctx.Done():
		stopCtx, timeoutCancel := context.WithTimeout(ctx, stopTimeout)
		defer timeoutCancel()
		_ = stop(stopCtx)
		return ctx.Err()
	}
}
