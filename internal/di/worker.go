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
// DI グラフの構築に失敗した場合、返す start／stop はいずれもその構築エラーを返します。
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
		// fx.Populate は fx.New 時点の invoke なので、グラフ構築が失敗すると対象は nil のまま残る。
		// logger を触る前に構築エラーを返し、nil 参照ではなく DI エラーをオペレータへ届ける。
		if err := app.Err(); err != nil {
			return failClosedChan(err)
		}

		l := logger.Named("worker.RunWorker").CallerSkip(callSkip)
		done := make(chan error, 1)
		state.Set(name, args, done)

		if err := app.Start(ctx); err != nil {
			l.Error(
				ctx,
				"failed to start worker application",
				logging.String(logging.WorkerNameKey, name),
				logging.Error(logging.ErrorKey, err),
			)
			return failClosedChan(err)
		}

		l.Info(ctx, "worker started", logging.String(logging.WorkerNameKey, name))
		return done
	}

	stop := func(ctx context.Context) error {
		// start と同じく、構築失敗時は nil の logger を参照せずに構築エラーを返す。
		if err := app.Err(); err != nil {
			return err
		}

		l := logger.Named("worker.RunWorker").CallerSkip(callSkip)
		if err := app.Stop(ctx); err != nil {
			l.Error(ctx, "failed to stop worker application", logging.Error(logging.ErrorKey, err))
			return err
		}
		l.Info(ctx, "worker application finished")
		return nil
	}

	return start, stop
}
