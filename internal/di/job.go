package di

import (
	"context"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/di/module"
	"go-boilerplate/internal/di/shutdowner"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/usecase/boundary/job"

	"go.uber.org/fx"
)

const callSkip = 2

type (
	// StartFunc は、ジョブの開始関数の型を表します。
	StartFunc func(ctx context.Context, name string, args []string) <-chan error
	// StopFunc は、ジョブの停止関数の型を表します。
	StopFunc func(ctx context.Context) error
)

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

// RunJob は、ジョブ実行用の開始関数・停止関数を生成して返します。context／タイムアウトの制御は返した関数の呼出側が担います。
func RunJob() (StartFunc, StopFunc) {
	var (
		state  job.State
		logger logging.Logger
		osCfg  *config.OperatingSystemConfig
	)

	app := fx.New(
		NewJobCore(),
		fx.Populate(&state, &logger, &osCfg),
	)

	start := func(ctx context.Context, name string, args []string) <-chan error {
		l := logger.Named("job.RunJob").CallerSkip(callSkip)
		done := make(chan error, 1)
		state.Set(name, args, done)

		if err := app.Start(ctx); err != nil {
			fields := append(jobEventFields(logging.EventTypeStart, osCfg.TimeZone()),
				logging.String(logging.JobNameKey, name),
				logging.Strings(logging.JobArgsKey, args),
				logging.Error(logging.JobErrorKey, err),
			)
			l.Error("failed to start job application", fields...)
			// 共有 done は hook goroutine の単独所有のため触れず、起動失敗は専用チャネルで返す
			// （done を触ると hook 側と二重 close／送信になり得る）。
			failCh := make(chan error, 1)
			failCh <- err
			close(failCh)
			return failCh
		}

		fields := append(jobEventFields(logging.EventTypeStart, osCfg.TimeZone()),
			logging.String(logging.JobNameKey, name),
			logging.Strings(logging.JobArgsKey, args),
		)
		l.Info("job started", fields...)

		return done
	}

	stop := func(ctx context.Context) error {
		l := logger.Named("job.RunJob").CallerSkip(callSkip)
		if err := app.Stop(ctx); err != nil {
			fields := append(jobEventFields(logging.EventTypeEnd, osCfg.TimeZone()),
				logging.Error(logging.JobErrorKey, err),
			)
			l.Error("failed to stop job application", fields...)
			return err
		}

		l.Info("job application finished", jobEventFields(logging.EventTypeEnd, osCfg.TimeZone())...)
		return nil
	}

	return start, stop
}

// jobEventFields は、ジョブ起動／停止ログ共通のイベントフィールド（種別・発生時刻・TZ）を返します。
func jobEventFields(eventType, tz string) []*logging.Field {
	return []*logging.Field{
		logging.String(logging.EventTypeKey, eventType),
		logging.Time(logging.EventAtKey, time.Now()),
		logging.String(logging.EventTzKey, tz),
	}
}
