package hook

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/config"
	workerengine "go-boilerplate/internal/controller/worker"
	mock_lifecycle "go-boilerplate/internal/di/lifecycle/mock"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	workerboundary "go-boilerplate/internal/usecase/boundary/worker"
)

// newTestEngine は、worker 未登録の Engine を生成する。engine.Run は未知 worker で
// 即時に ErrUnknownWorker を返すため、hook の Body 分岐検証に用いる。
func newTestEngine(t *testing.T, log logging.Logger) *workerengine.Engine {
	t.Helper()
	eng, err := workerengine.New(
		nil,
		workerengine.Settings{},
		observability.NewNoopTracerFactory(t),
		observability.NewNoopWorkerMetrics(t),
		log,
	)
	require.NoError(t, err)
	return eng
}

// captureWorkerHooks は、RegisterWorkerHooks が登録する start / stop 関数を mock 経由で捕捉する。
func captureWorkerHooks(
	t *testing.T,
	engine *workerengine.Engine,
	state workerboundary.State,
	wc *config.WorkerConfig,
	log logging.Logger,
) (func(context.Context) error, func(context.Context) error) {
	t.Helper()

	ctrl := gomock.NewController(t)
	reg := mock_lifecycle.NewMockRegistrar(ctrl)
	var start, stop func(context.Context) error
	reg.EXPECT().RegisterStart(gomock.AssignableToTypeOf(start)).
		Do(func(fn func(context.Context) error) { start = fn }).Times(1)
	reg.EXPECT().RegisterStop(gomock.AssignableToTypeOf(stop)).
		Do(func(fn func(context.Context) error) { stop = fn }).Times(1)

	RegisterWorkerHooks(reg, engine, state, wc, log)
	require.NotNil(t, start)
	require.NotNil(t, stop)
	return start, stop
}

func TestRegisterWorkerHooks(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("起動対象がある場合_engineを実行しdoneへ結果を送って閉じる", func(t *testing.T) {
			t.Parallel()

			log := logging.NewTestLogger(t)
			engine := newTestEngine(t, log)
			state := workerengine.NewState()
			done := make(chan error, 1)
			state.Set("missing-worker", nil, done)

			wc := config.NewWorkerConfig(config.MockConfigForTest(t))
			// 並列サブテスト間で health listener の既定ポート(:8081)が衝突するため空きポートを使う。
			wc.SetHealthListenAddr(t, "127.0.0.1:0")
			start, stop := captureWorkerHooks(t, engine, state, wc, log)

			require.NoError(t, start(context.Background()))

			err := <-done
			require.ErrorIs(t, err, workerengine.ErrUnknownWorker)
			_, ok := <-done
			assert.False(t, ok, "done は close される")

			require.NoError(t, stop(context.Background()))
		})

		t.Run("起動対象がない場合_engineを起動せずNoWorkerToRunをログ出力する", func(t *testing.T) {
			t.Parallel()

			log, observed := logging.NewObservedTestLogger(t)
			engine := newTestEngine(t, log)
			state := workerengine.NewState() // Set しない = done は nil

			wc := config.NewWorkerConfig(config.MockConfigForTest(t))
			// 並列サブテスト間で health listener の既定ポート(:8081)が衝突するため空きポートを使う。
			wc.SetHealthListenAddr(t, "127.0.0.1:0")
			start, stop := captureWorkerHooks(t, engine, state, wc, log)

			require.NoError(t, start(context.Background()))
			require.NoError(t, stop(context.Background()))

			assert.Positive(t, observed.FilterMessage("No worker to run").Len())
		})
	})
}
