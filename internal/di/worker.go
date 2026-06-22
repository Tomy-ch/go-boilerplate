package di

import (
	"context"

	"go.uber.org/fx"

	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/di/module"
	"go-boilerplate/internal/di/shutdowner"
	"go-boilerplate/internal/logging"
	workerboundary "go-boilerplate/internal/usecase/boundary/worker"
)

// NewWorkerCore は、worker の fx.App を構成する fx.Option を返します。
// WorkerModule は本コアにのみ含め、serve 用の NewApplicationCore には含めません（依存隔離）。
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
// context の制御は呼び出し側（cli/worker）が担います。
func RunWorker() (StartFunc, StopFunc) {
	var (
		state  workerboundary.State
		logger logging.Logger
	)

	app := fx.New(
		NewWorkerCore(),
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
			// 共有 done に触れると hook goroutine と二重送信になり得るため、専用チャネルで返す。
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
