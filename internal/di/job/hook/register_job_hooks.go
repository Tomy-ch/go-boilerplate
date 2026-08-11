// Package hook は、ジョブのライフサイクルフックを提供します。
package hook

import (
	"context"
	"time"

	"go-boilerplate/internal/config"

	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/di/shutdowner"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/usecase/boundary/job"
)

const loggerName = "job.Hooks"

// RegisterJobHooks は、ジョブのライフサイクルフックを登録します。
//
// OnStop で実行 context がキャンセルされるため、`--timeout` 超過で cli が
// app.Stop を呼ぶと実行中のジョブ（DB クエリ等）が中断されます。
func RegisterJobHooks(
	reg lifecycle.Registrar,
	sd shutdowner.Shutdowner,
	runner job.Runner,
	logger logging.Logger,
	osCfg *config.OperatingSystemConfig,
	state job.State,
) {
	lifecycle.SupervisedRunner{
		Body: func(ctx context.Context) { runJobAndShutdown(ctx, sd, runner, logger, osCfg, state) },
	}.Register(reg)
}

// runJobAndShutdown は、スナップショットしたジョブを実行し done に結果を送って停止を要求する。
// ジョブ未設定時（done==nil）はログ出力のみ行って停止する。
// jobCtx は [lifecycle.SupervisedRunner] が供給する実行 context。
func runJobAndShutdown(
	jobCtx context.Context,
	sd shutdowner.Shutdowner,
	runner job.Runner,
	logger logging.Logger,
	osCfg *config.OperatingSystemConfig,
	state job.State,
) {
	name, args, done := state.Snapshot()
	if done == nil {
		logger.Named(loggerName).Info(
			jobCtx,
			"No job to run",
			logging.String(logging.EventTypeKey, logging.EventTypeStart),
			logging.Time(logging.EventAtKey, time.Now()),
			logging.String(logging.EventTzKey, osCfg.TimeZone()),
			logging.String(logging.JobNameKey, name),
			logging.Strings(logging.JobArgsKey, args),
		)
		shutdown(jobCtx, sd, logger)
		return
	}

	defer close(done)

	done <- runner.Run(jobCtx, name, args)

	shutdown(jobCtx, sd, logger)
}

// shutdown は、ジョブ完了後のアプリ停止を要求し、失敗時はジョブ系のログ様式に合わせて記録します。
func shutdown(ctx context.Context, sd shutdowner.Shutdowner, logger logging.Logger) {
	if err := sd.Shutdown(); err != nil {
		logger.Named(loggerName).Error(
			ctx,
			"failed to shutdown",
			logging.Error(logging.JobErrorKey, err),
		)
	}
}
