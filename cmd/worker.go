package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	cliworker "go-boilerplate/internal/cli/worker"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di"
)

// newWorkerCommand は worker コマンドを生成します。
func newWorkerCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "worker",
		Short: "worker <worker-name> [args...] コマンドは、指定された worker を起動します。",
		Long: "worker <worker-name> [args...] コマンドは、指定された pull-ack worker を起動し、SIGTERM まで常駐します。\n" +
			"例: worker myworker",
		Args: cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := config.SetUpConfig()
			if err != nil {
				return err
			}
			grace := config.NewApplicationConfig(cfg).ShutdownTimeout()

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			return cliworker.RunWorkerWith(ctx, args[0], args[1:], grace, func() (cliworker.StartFunc, cliworker.StopFunc) {
				start, stopFn := di.RunWorker(grace)
				return cliworker.StartFunc(start), cliworker.StopFunc(stopFn)
			})
		},
	}
}
