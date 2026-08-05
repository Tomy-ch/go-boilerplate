package withdrawalarchive_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/controller/worker/withdrawalarchive"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	mock_worker "go-boilerplate/internal/usecase/boundary/worker/mock"
	mock_user "go-boilerplate/internal/usecase/user/mock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("注入した broker adapter と業務処理を束ねた Worker を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			consumer := mock_worker.NewMockConsumer(ctrl)
			failure := mock_worker.NewMockFailureHandler(ctrl)

			got := withdrawalarchive.New(
				consumer,
				failure,
				mock_user.NewMockArchiveUsecase(ctrl),
				observability.NewNoopTracerFactory(t),
				logging.NewTestLogger(t),
			)

			assert.Equal(t, withdrawalarchive.Name, got.Name())
			assert.Same(t, consumer, got.Consumer())
			assert.Same(t, failure, got.FailureHandler())
			assert.NotNil(t, got.Handler())
		})
	})
}
