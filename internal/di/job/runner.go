// Package job は、ジョブ関連の依存性注入を提供します。
package job

import (
	jobrunner "boilerplate-go/internal/controller/job"
	"boilerplate-go/internal/usecase/support/job"

	"go.uber.org/fx"
)

// JobRunnerIn は、Runnerの入力パラメータを表します。
type JobRunnerIn struct {
	fx.In
	Jobs []job.Job `group:"jobs"`
}

// ProvideRunner は、ジョブランナーを提供します。
func ProvideRunner(in JobRunnerIn) (job.Runner, error) {
	return jobrunner.NewRunner(in.Jobs)
}
