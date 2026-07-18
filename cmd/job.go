package main

import (
	"time"

	"go-boilerplate/internal/cli/job"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di"

	"github.com/spf13/cobra"
)

// newJobCommand は job コマンドを生成します。
func newJobCommand() *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "job",
		Short: "job <job-name> [args...] コマンドは、指定されたジョブを実行します。",
		Long: "job <job-name> [args...] コマンドは、指定されたジョブを実行します。ジョブ名と引数を指定して実行してください。\n" +
			"例: job <job-name> --timeout 30s",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.SetUpConfig()
			if err != nil {
				return err
			}
			grace := config.NewApplicationConfig(cfg).ShutdownTimeout()

			return job.RunJobWith(cmd.Context(), args[0], args[1:], timeout, grace, func() (job.StartFunc, job.StopFunc) {
				start, stop := di.RunJob(grace)
				return job.StartFunc(start), job.StopFunc(stop)
			})
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", 0, "job execution timeout duration (e.g., 30s, 1m)")

	return cmd
}
