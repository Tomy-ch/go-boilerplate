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

// RegisterJobHooks は、ジョブのライフサイクルフックを登録します。
func RegisterJobHooks(
	reg lifecycle.Registrar,
	sd shutdowner.Shutdowner,
	runner job.Runner,
	logger logging.Logger,
	osCfg *config.OperatingSystemConfig,
	state job.State,
) {
	reg.RegisterStart(func(startCtx context.Context) error {
		go runJobAndShutdown(startCtx, sd, runner, logger, osCfg, state)
		return nil
	})
}

// runJobAndShutdown は、スナップショットしたジョブを detached goroutine の本体として実行し、
// done に結果を送って完了後に停止を要求する。ジョブ未設定時はログのみ出力して停止する。
// detached goroutine の本体を名前付き関数に切り出すことで、テストから同期的に直接呼び出せる。
func runJobAndShutdown(
	startCtx context.Context,
	sd shutdowner.Shutdowner,
	runner job.Runner,
	logger logging.Logger,
	osCfg *config.OperatingSystemConfig,
	state job.State,
) {
	name, args, done := state.Snapshot()
	if done == nil {
		logger.Named("job.Hooks").Info(
			"No job to run",
			logging.String(logging.EventTypeKey, logging.EventTypeStart),
			logging.Time(logging.EventAtKey, time.Now()),
			logging.String(logging.EventTzKey, osCfg.TimeZone()),
			logging.String(logging.JobNameKey, name),
			logging.Strings(logging.JobArgsKey, args),
		)
		shutdown(sd, logger)
		return
	}

	defer close(done)

	// startCtx は fx の OnStart フック用で、OnStart 完了後にキャンセルされる。
	// ジョブ本体は detached goroutine で起動後も走り続けるため、キャンセルだけ
	// 無効化した派生 context を渡し、起動 ctx のキャンセルに巻き込まれないようにする。
	jobCtx := context.WithoutCancel(startCtx)
	done <- runner.Run(jobCtx, name, args)

	shutdown(sd, logger)
}

// shutdown は、ジョブ完了後のアプリ停止を要求し、失敗時はジョブ系のログ様式に合わせて記録します。
// 隣接する di/job.go の app.Stop 失敗ログと方針を揃え、エラーの黙殺を避けます。
func shutdown(sd shutdowner.Shutdowner, logger logging.Logger) {
	if err := sd.Shutdown(); err != nil {
		logger.Named("job.Hooks").Error(
			"failed to shutdown",
			logging.Error(logging.JobErrorKey, err),
		)
	}
}
