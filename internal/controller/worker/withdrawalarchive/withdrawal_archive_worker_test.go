package withdrawalarchive

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	workerbd "go-boilerplate/internal/usecase/boundary/worker"
	mock_worker "go-boilerplate/internal/usecase/boundary/worker/mock"
	mock_user "go-boilerplate/internal/usecase/user/mock"
)

// newWorkerUnderTest は、mock を差した worker と、注入した broker adapter を返します。
func newWorkerUnderTest(t *testing.T) (workerbd.Worker, workerbd.Consumer, workerbd.FailureHandler) {
	t.Helper()

	ctrl := gomock.NewController(t)
	consumer := mock_worker.NewMockConsumer(ctrl)
	failure := mock_worker.NewMockFailureHandler(ctrl)
	w := New(
		consumer,
		failure,
		mock_user.NewMockArchiveUsecase(ctrl),
		observability.NewNoopTracerFactory(t),
		logging.NewTestLogger(t),
	)
	return w, consumer, failure
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Worker を返す", func(t *testing.T) {
			t.Parallel()

			got, _, _ := newWorkerUnderTest(t)

			assert.NotNil(t, got)
		})
	})
}

func Test_workerImpl_Name(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("サブコマンド引数で選択される worker 名を返す", func(t *testing.T) {
			t.Parallel()

			got, _, _ := newWorkerUnderTest(t)

			assert.Equal(t, "withdrawal-archive", got.Name())
		})
	})
}

func Test_workerImpl_Consumer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("注入された Consumer をそのまま返す", func(t *testing.T) {
			t.Parallel()

			got, consumer, _ := newWorkerUnderTest(t)

			assert.Same(t, consumer, got.Consumer())
		})
	})
}

func Test_workerImpl_Handler(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("業務処理を返す", func(t *testing.T) {
			t.Parallel()

			got, _, _ := newWorkerUnderTest(t)

			assert.NotNil(t, got.Handler())
		})
	})
}

func Test_workerImpl_FailureHandler(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("注入された FailureHandler をそのまま返す", func(t *testing.T) {
			t.Parallel()

			got, _, failure := newWorkerUnderTest(t)

			assert.Same(t, failure, got.FailureHandler())
		})

		t.Run("退避先を注入しなければ nil を返す", func(t *testing.T) {
			t.Parallel()
			// nil は「engine 既定の扱いに委ねる」の表明で、退避先を持たない構成の正規の形。
			ctrl := gomock.NewController(t)
			w := New(
				mock_worker.NewMockConsumer(ctrl),
				nil,
				mock_user.NewMockArchiveUsecase(ctrl),
				observability.NewNoopTracerFactory(t),
				logging.NewTestLogger(t),
			)

			assert.Nil(t, w.FailureHandler())
		})
	})
}
