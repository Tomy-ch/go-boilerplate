package module

import (
	"go.uber.org/fx"

	workercontroller "go-boilerplate/internal/controller/worker"
	diworker "go-boilerplate/internal/di/worker"
	"go-boilerplate/internal/di/worker/hook"
	"go-boilerplate/internal/observability"
	queuemetrics "go-boilerplate/internal/observability/metrics/queue"
)

// queueStatsTargetsIn は、DI group から QueueStatsTarget を集約します。
type queueStatsTargetsIn struct {
	fx.In

	Targets []queuemetrics.Target `group:"worker.queue_stats_targets"`
}

// WorkerModule は、worker engine 関連の依存関係を提供する fx.Module です。
// 既定では worker を 1 つも登録しません。
func WorkerModule() fx.Option {
	return fx.Module("worker",
		provideWorkers(
		// ここに worker のコンストラクタを追加します（例: <pkg>.New）。
		// 各 worker は usecase/boundary/worker.Worker を実装し、Consumer(broker adapter) を内包します。
		),
		provideQueueStatsTargets(
		// ここに QueueStatsTarget のコンストラクタを追加します。
		// SQS など QueueStatsProvider を実装する adapter を使う worker のときだけ任意登録します。
		// コンストラクタは queuemetrics.Target を返し、sqs.NewQueueStatsProvider で Provider を組み立てます。
		),
		fx.Provide(
			observability.NewWorkerMetrics,
			diworker.ProvideEngine,
			workercontroller.NewState,
			provideQueueStatsCollector,
		),
		// 起動時に DrainTimeout < grace を検証する。違反時は app.Start が失敗する。
		fx.Invoke(diworker.ValidateShutdownGrace),
		fx.Invoke(hook.RegisterWorkerHooks),
		// queue depth / DLQ 収集器を Prometheus レジストリへ登録する（対象が無ければ何も出さない）。
		fx.Invoke(queuemetrics.RegisterStatsCollector),
	)
}

// provideQueueStatsCollector は、収集対象から queue depth / DLQ 収集器を生成します。
// 対象が 1 つも無い場合は、何も出力しない収集器になります。
func provideQueueStatsCollector(in queueStatsTargetsIn) *queuemetrics.StatsCollector {
	return queuemetrics.NewStatsCollector(in.Targets)
}

// provideQueueStatsTargets は、QueueStatsTarget のコンストラクタ群を group へ登録します。
func provideQueueStatsTargets(constructors ...any) fx.Option {
	opts := make([]fx.Option, len(constructors))
	for i, c := range constructors {
		opts[i] = fx.Provide(
			fx.Annotate(
				c,
				fx.ResultTags(`group:"worker.queue_stats_targets"`),
			),
		)
	}
	return fx.Options(opts...)
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
