package module

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	workercontroller "go-boilerplate/internal/controller/worker"
	queuemetrics "go-boilerplate/internal/observability/metrics/queue"
	workerboundary "go-boilerplate/internal/usecase/boundary/worker"
)

// fakeWorker は、名前だけを返す最小の Worker 実装です（group へ集約された個々の寄与を識別するために使う）。
type fakeWorker struct {
	workerboundary.Worker

	name string
}

func (w fakeWorker) Name() string { return w.name }

func TestWorkerModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("worker未登録でも engine とフックが欠落なく結線される", func(t *testing.T) {
			t.Parallel()

			// 個々の worker の振る舞いは controller 層のテストに任せ、ここでは engine と
			// そのフック（ValidateShutdownGrace / RegisterWorkerHooks / RegisterStatsCollector）が
			// 依存と欠落なく結線されることを確認する。
			opts := append(commonDeps(), InfrastructureModule(), UsecaseModule(), WorkerModule())
			validateGraph(t, opts...)
		})

		t.Run("workerとqueue_stats_targetを登録してもグラフが結線される", func(t *testing.T) {
			t.Parallel()

			// provideWorkers / provideQueueStatsTargets で実際に group へ登録した場合も
			// グラフが成立することを確認する（登録経路の結線検証）。
			opts := append(commonDeps(), InfrastructureModule(), UsecaseModule(),
				WorkerModule(),
				provideWorkers(func() workerboundary.Worker { return nil }),
				provideQueueStatsTargets(func() queuemetrics.Target { return queuemetrics.Target{} }),
			)
			validateGraph(t, opts...)
		})
	})
}

func Test_provideQueueStatsCollector(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("収集対象が無くても nil でない収集器を返す", func(t *testing.T) {
			t.Parallel()

			c := provideQueueStatsCollector(queueStatsTargetsIn{})
			assert.NotNil(t, c)
		})
	})
}

func Test_provideWorkers(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡した全コンストラクタの Worker が workers グループへ集まる", func(t *testing.T) {
			t.Parallel()

			got := collectGroup[workerboundary.Worker](t, `group:"workers"`, provideWorkers(
				func() workerboundary.Worker { return fakeWorker{name: "a"} },
				func() workerboundary.Worker { return fakeWorker{name: "b"} },
			))

			names := make([]string, 0, len(got))
			for _, w := range got {
				names = append(names, w.Name())
			}
			assert.ElementsMatch(t, []string{"a", "b"}, names)
		})

		t.Run("コンストラクタが 0 個の場合は何も登録しない", func(t *testing.T) {
			t.Parallel()

			// WorkerModule は既定で worker を 1 つも登録しないため、空呼び出しが成立することが前提条件になる。
			assert.Empty(t, collectGroup[workerboundary.Worker](t, `group:"workers"`, provideWorkers()))
		})
	})
}

func Test_provideQueueStatsTargets(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡した全コンストラクタの Target が queue_stats_targets グループへ集まる", func(t *testing.T) {
			t.Parallel()

			got := collectGroup[queuemetrics.Target](t, `group:"worker.queue_stats_targets"`, provideQueueStatsTargets(
				func() queuemetrics.Target { return queuemetrics.Target{WorkerName: "a"} },
				func() queuemetrics.Target { return queuemetrics.Target{WorkerName: "b"} },
			))

			names := make([]string, 0, len(got))
			for _, target := range got {
				names = append(names, target.WorkerName)
			}
			assert.ElementsMatch(t, []string{"a", "b"}, names)
		})

		t.Run("コンストラクタが 0 個の場合は何も登録しない", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, collectGroup[queuemetrics.Target](t, `group:"worker.queue_stats_targets"`, provideQueueStatsTargets()))
		})
	})
}

func TestWorkerModule(t *testing.T) {
	t.Parallel()

	workerDeps := func() []fx.Option {
		return append(commonDeps(), InfrastructureModule(), UsecaseModule())
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("worker engine / State / queue 統計収集器を提供する", func(t *testing.T) {
			t.Parallel()

			var (
				engine    *workercontroller.Engine
				state     workerboundary.State
				collector *queuemetrics.StatsCollector
			)

			validateGraph(t, append(workerDeps(), WorkerModule(),
				fx.Populate(&engine, &state, &collector))...)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未配線では worker engine が解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			var engine *workercontroller.Engine

			opts := append(workerDeps(), fx.Populate(&engine), fx.NopLogger)
			require.Error(t, fx.ValidateApp(opts...))
		})
	})
}
