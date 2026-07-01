package di

import (
	"context"
	"time"

	"go.uber.org/fx"

	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/di/module"
	"go-boilerplate/internal/di/shutdowner"
	"go-boilerplate/internal/logging"
	workerboundary "go-boilerplate/internal/usecase/boundary/worker"
)

// NewWorkerCore は、worker 実行用の fx.App を構成する fx.Option を返します。
func NewWorkerCore() fx.Option {
	return fx.Options(
		shutdowner.Module(),
		lifecycle.Module(),
		module.ConfigModule(),
		module.LoggingModule(),
		module.ObservabilityModule(),
		module.DatabaseModule(),
		module.SystemModule(),
		module.InfrastructureModule(),
		module.UsecaseModule(),
		module.WorkerModule(),
	)
}

// RunWorker は、worker 実行用の開始関数・停止関数を生成して返します。
// grace（APP_SHUTDOWN_TIMEOUT）を fx.StopTimeout に設定し、停止時に fx 既定（15s）が
// DrainTimeout より先に drain を打ち切らないようにします。
func RunWorker(grace time.Duration) (StartFunc, StopFunc) {
	var (
		state  workerboundary.State
		logger logging.Logger
	)

	app := fx.New(
		NewWorkerCore(),
		fx.StopTimeout(grace),
		fx.Populate(&state, &logger),
		fx.WithLogger(NewFxEventLogger),
	)

	start := func(ctx context.Context, name string, args []string) <-chan error {
		l := logger.Named("worker.RunWorker").CallerSkip(callSkip)
		done := make(chan error, 1)
		state.Set(name, args, done)

		if err := app.Start(ctx); err != nil {
			l.Error(
				"failed to start worker application",
				logging.String(logging.WorkerNameKey, name),
				logging.Error(logging.ErrorKey, err),
			)
			// done は hook goroutine が送信/close を所有するため、起動失敗は専用チャネルで返す。
			failCh := make(chan error, 1)
			failCh <- err
			close(failCh)
			return failCh
		}

		l.Info("worker started", logging.String(logging.WorkerNameKey, name))
		return done
	}

	stop := func(ctx context.Context) error {
		l := logger.Named("worker.RunWorker").CallerSkip(callSkip)
		if err := app.Stop(ctx); err != nil {
			l.Error("failed to stop worker application", logging.Error(logging.ErrorKey, err))
			return err
		}
		l.Info("worker application finished")
		return nil
	}

	return start, stop
}
