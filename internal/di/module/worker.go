package module

import (
	"go.uber.org/fx"

	workercontroller "go-boilerplate/internal/controller/worker"
	diworker "go-boilerplate/internal/di/worker"
	"go-boilerplate/internal/di/worker/hook"
	"go-boilerplate/internal/observability"
)

// WorkerModule は、worker engine 関連の依存関係を提供する fx.Module です。
// 既定では worker を 1 つも登録しません。
func WorkerModule() fx.Option {
	return fx.Module("worker",
		provideWorkers(
		// ここに worker のコンストラクタを追加します（例: <pkg>.New）。
		// 各 worker は usecase/boundary/worker.Worker を実装し、Consumer(broker adapter) を内包します。
		),
		fx.Provide(
			observability.NewWorkerMetrics,
			diworker.ProvideEngine,
			workercontroller.NewState,
		),
		// 起動時に DrainTimeout < grace を検証する。違反時は app.Start が失敗する。
		fx.Invoke(diworker.ValidateShutdownGrace),
		fx.Invoke(hook.RegisterWorkerHooks),
	)
}

// provideWorkers は、worker のコンストラクタ群を worker グループへ登録します。
func provideWorkers(constructors ...any) fx.Option {
	opts := make([]fx.Option, len(constructors))
	for i, c := range constructors {
		opts[i] = fx.Provide(
			fx.Annotate(
				c,
				fx.ResultTags(`group:"workers"`),
			),
		)
	}
	return fx.Options(opts...)
}
