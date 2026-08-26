// Package hook は、worker のライフサイクルフックを提供します。
package hook

import (
	"context"

	cliworker "go-boilerplate/internal/cli/worker"
	"go-boilerplate/internal/config"
	workerengine "go-boilerplate/internal/controller/worker"
	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/logging"
	workerboundary "go-boilerplate/internal/usecase/boundary/worker"
)

// RegisterWorkerHooks は、worker engine と health listener のライフサイクルフックを登録します。
// health listener は OnStart で起動し OnStop で停止します。engine の起動・停止契約は
// [lifecycle.SupervisedRunner] に従います。
func RegisterWorkerHooks(
	reg lifecycle.Registrar,
	engine *workerengine.Engine,
	state workerboundary.State,
	wc *config.WorkerConfig,
	logger logging.Logger,
) {
	startHealth, stopHealth := cliworker.NewHealthServer(wc.HealthListenAddr(), engine.Healthy, logger)

	lifecycle.SupervisedRunner{
		OnStartAux: startHealth,
		Body: func(ctx context.Context) {
			name, _, done := state.Snapshot()
			if done == nil {
				logger.Named("worker.Hooks").Info(ctx, "No worker to run", logging.String(logging.WorkerNameKey, name))
				return
			}
			defer close(done)
			done <- engine.Run(ctx, name)
		},
		OnStopAux: stopHealth,
	}.Register(reg)
}
