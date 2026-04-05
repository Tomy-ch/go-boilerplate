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
	osCfg *config.OperationSystemConfig,
	state job.State,
) {
	reg.RegisterStart(func(startCtx context.Context) error {
		go func() {
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
				_ = sd.Shutdown()
				return
			}

			defer close(done)

			err := runner.Run(startCtx, name, args)
			done <- err

			_ = sd.Shutdown()
		}()
		return nil
	})
}
