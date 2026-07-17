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
//   - OnStart: health listener を起動し、選択された worker を detached goroutine で実行する。
//   - OnStop:  engine の context をキャンセルして drain 完了を（stopCtx の範囲で）待ち、health listener を停止する。
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
			done <- engine.Run(ctx, name) // 猶予超過時の未完は Ack されず再配送へ
		},
		OnStopAux: stopHealth,
	}.Register(reg)
}
