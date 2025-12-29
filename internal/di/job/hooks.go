package job

import (
	"context"
	"time"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/di/lifecycle"
	"boilerplate-go/internal/logging"
	"boilerplate-go/internal/usecase/support/job"

	"go.uber.org/fx"
)

// RegisterJobHooks は、ジョブのライフサイクルフックを登録します。
func RegisterJobHooks(
	reg lifecycle.Registrar,
	sd fx.Shutdowner,
	runner job.Runner,
	logger logging.Logger,
	osCfg *config.OperationSystemConfig,
	state *State,
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
