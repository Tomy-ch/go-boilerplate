package di

import (
	"context"
	"time"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/di/lifecycle"
	"boilerplate-go/internal/di/module"
	"boilerplate-go/internal/di/shutdowner"
	"boilerplate-go/internal/logging"
	"boilerplate-go/internal/usecase/support/job"

	"go.uber.org/fx"
)

const callSkip = 2

type (
	// StartFunc は、ジョブの開始関数の型を表します。
	StartFunc func(ctx context.Context, name string, args []string) <-chan error
	// StopFunc は、ジョブの停止関数の型を表します。
	StopFunc func(ctx context.Context) error
)

type JobDeps struct {
	State  job.State
	Logger logging.Logger
	OSCfg  *config.OperationSystemConfig
}

// NewJobCore は、ジョブの fx.App インスタンスを作成します。
func NewJobCore() fx.Option {
	return fx.Options(
		// Shutdowner Module
		shutdowner.Module(),
		// Lifecycle Module
		lifecycle.Module(),
		// Common Module
		module.ConfigModule(),
		module.LoggingModule(),
		module.ObservabilityModule(),
		module.DatabaseModule(),
		module.SystemModule(),
		// DDD Modules
		module.InfrastructureModule(),
		module.UsecaseModule(),
		// Job Module
		module.JobModule(),
	)
}

// RunJob は、指定されたコンテキストとタイムアウトでジョブランナーを実行します。
func RunJob() (StartFunc, StopFunc) {
	var (
		state  job.State
		logger logging.Logger
		osCfg  *config.OperationSystemConfig
	)

	app := fx.New(
		NewJobCore(),
		fx.Populate(&state, &logger, &osCfg),
	)

	start := func(ctx context.Context, name string, args []string) <-chan error {
		done := make(chan error, 1)
		state.Set(name, args, done)

		if err := app.Start(ctx); err != nil {
			logger.Named("job.RunJob").CallerSkip(callSkip).Error(
				"failed to start job application",
				logging.String(logging.EventTypeKey, logging.EventTypeStart),
				logging.Time(logging.EventAtKey, time.Now()),
				logging.String(logging.EventTzKey, osCfg.TimeZone()),
				logging.String(logging.JobNameKey, name),
				logging.Strings(logging.JobArgsKey, args),
				logging.Error(logging.JobErrorKey, err),
			)
			done <- err
			close(done)
			return done
		}

		logger.Named("job.RunJob").CallerSkip(callSkip).Info(
			"job started",
			logging.String(logging.EventTypeKey, logging.EventTypeStart),
			logging.Time(logging.EventAtKey, time.Now()),
			logging.String(logging.EventTzKey, osCfg.TimeZone()),
			logging.String(logging.JobNameKey, name),
			logging.Strings(logging.JobArgsKey, args),
		)

		return done
	}

	stop := func(ctx context.Context) error {
		if err := app.Stop(ctx); err != nil {
			logger.Named("job.RunJob").CallerSkip(callSkip).Error(
				"failed to stop job application",
				logging.String(logging.EventTypeKey, logging.EventTypeEnd),
				logging.Time(logging.EventAtKey, time.Now()),
				logging.String(logging.EventTzKey, osCfg.TimeZone()),
				logging.Error(logging.JobErrorKey, err),
			)
			return err
		}

		logger.Named("job.RunJob").CallerSkip(callSkip).Info(
			"job application finished",
			logging.String(logging.EventTypeKey, logging.EventTypeEnd),
			logging.Time(logging.EventAtKey, time.Now()),
			logging.String(logging.EventTzKey, osCfg.TimeZone()),
		)
		return nil
	}

	return start, stop
}
