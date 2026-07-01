package module

import (
	"testing"

	"github.com/stretchr/testify/assert"

	queuemetrics "go-boilerplate/internal/observability/metrics/queue"
	workerboundary "go-boilerplate/internal/usecase/boundary/worker"
)

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
