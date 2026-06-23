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
	engineCtx, cancel := context.WithCancel(context.Background())
	engineDone := make(chan struct{})
	startHealth, stopHealth := cliworker.NewHealthServer(wc.HealthListenAddr(), engine.Healthy, logger)

	reg.RegisterStart(func(_ context.Context) error {
		startHealth()

		name, _, done := state.Snapshot()
		if done == nil {
			logger.Named("worker.Hooks").Info("No worker to run", logging.String(logging.WorkerNameKey, name))
			close(engineDone)
			return nil
		}

		// engineCtx は OnStop でのみキャンセルする（OnStart 完了後の startCtx キャンセルに巻き込まれない）。
		go func() {
			defer close(engineDone)
			defer close(done)
			done <- engine.Run(engineCtx, name)
		}()
		return nil
	})

	reg.RegisterStop(func(stopCtx context.Context) error {
		cancel()
		select {
		case <-engineDone: // drain 完了
		case <-stopCtx.Done(): // 猶予切れ（未完は Ack されず再配送へ）
		}
		stopHealth(stopCtx)
		return nil
	})
}
