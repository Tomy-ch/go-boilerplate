package worker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/config"
	workerengine "go-boilerplate/internal/controller/worker"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	workerboundary "go-boilerplate/internal/usecase/boundary/worker"
	mock_worker "go-boilerplate/internal/usecase/boundary/worker/mock"
)

// newEngineIn は、ProvideEngine 用の EngineIn を no-op の観測系・テスト用設定で組み立てます。
func newEngineIn(t *testing.T, workers []workerboundary.Worker) EngineIn {
	t.Helper()

	cfg := config.MockConfigForTest(t)
	return EngineIn{
		Workers: workers,
		Config:  config.NewWorkerConfig(cfg),
		TF:      observability.NewNoopTracerFactory(t),
		Metrics: observability.NewNoopWorkerMetrics(t),
		Logger:  logging.NewTestLogger(t),
	}
}

// newNamedWorker は、指定名を返す MockWorker を生成します。
func newNamedWorker(ctrl *gomock.Controller, name string) *mock_worker.MockWorker {
	w := mock_worker.NewMockWorker(ctrl)
	w.EXPECT().Name().Return(name).AnyTimes()
	return w
}

func Test_validateShutdownGrace(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("drainがgrace未満なら成功する", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, validateShutdownGrace(30*time.Second, 45*time.Second))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("drainがgraceより大きいとErrInvalidShutdownGraceを返す", func(t *testing.T) {
			t.Parallel()

			err := validateShutdownGrace(60*time.Second, 45*time.Second)
			require.ErrorIs(t, err, ErrInvalidShutdownGrace)
			require.ErrorContains(t, err, "WORKER_DRAIN_TIMEOUT")
		})

		t.Run("drainとgraceが等しいとErrInvalidShutdownGraceを返す", func(t *testing.T) {
			t.Parallel()

			err := validateShutdownGrace(45*time.Second, 45*time.Second)
			require.ErrorIs(t, err, ErrInvalidShutdownGrace)
		})
	})
}

//nolint:paralleltest // 異常系で config.New + EnsureRepoRootAndEnv(t.Setenv/t.Chdir) を使用するため並列化不可
func TestValidateShutdownGrace(t *testing.T) {
	//nolint:paralleltest // 兄弟の異常系が t.Setenv/t.Chdir を使用するため並列化不可
	t.Run("正常系", func(t *testing.T) {
		//nolint:paralleltest // 兄弟の異常系が t.Setenv/t.Chdir を使用するため並列化不可
		t.Run("drainがgrace未満の設定なら成功する", func(t *testing.T) {
			cfg := config.MockConfigForTest(t)

			err := ValidateShutdownGrace(config.NewApplicationConfig(cfg), config.NewWorkerConfig(cfg))
			require.NoError(t, err)
		})
	})

	//nolint:paralleltest // EnsureRepoRootAndEnv(t.Setenv/t.Chdir) を使用するため並列化不可
	t.Run("異常系", func(t *testing.T) {
		//nolint:paralleltest // EnsureRepoRootAndEnv(t.Setenv/t.Chdir) を使用するため並列化不可
		t.Run("drainがgrace以上の設定だとErrInvalidShutdownGraceを返す", func(t *testing.T) {
			config.EnsureRepoRootAndEnv(t, config.TestingEnvValue)
			t.Setenv("WORKER_DRAIN_TIMEOUT", "60s")
			t.Setenv("APP_SHUTDOWN_TIMEOUT", "45s")

			cfg, err := config.New()
			require.NoError(t, err)

			err = ValidateShutdownGrace(config.NewApplicationConfig(cfg), config.NewWorkerConfig(cfg))
			require.ErrorIs(t, err, ErrInvalidShutdownGrace)
		})
	})
}

func TestProvideEngine(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("worker無しでもEngineをエラーなく構築する", func(t *testing.T) {
			t.Parallel()

			eng, err := ProvideEngine(newEngineIn(t, nil))
			require.NoError(t, err)
			require.NotNil(t, eng)
			assert.Empty(t, eng.Names())
		})

		t.Run("複数のworkerを渡すとNamesに全名称が登録される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			workers := []workerboundary.Worker{
				newNamedWorker(ctrl, "alpha"),
				newNamedWorker(ctrl, "beta"),
			}

			eng, err := ProvideEngine(newEngineIn(t, workers))
			require.NoError(t, err)
			require.NotNil(t, eng)
			assert.ElementsMatch(t, []string{"alpha", "beta"}, eng.Names())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同名のworkerが重複する場合ErrDuplicateWorkerを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			workers := []workerboundary.Worker{
				newNamedWorker(ctrl, "dup"),
				newNamedWorker(ctrl, "dup"),
			}

			eng, err := ProvideEngine(newEngineIn(t, workers))
			assert.Nil(t, eng)
			require.ErrorIs(t, err, workerengine.ErrDuplicateWorker)
		})
	})
}
