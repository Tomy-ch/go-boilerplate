package module

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/observability"
	publisherbndry "go-boilerplate/internal/usecase/boundary/publisher"
	mock_realtime "go-boilerplate/internal/usecase/boundary/realtime/mock"
)

func Test_realtimePublisherModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	validateGraph(t, append(commonDeps(), realtimePublisherModule())...)
}

func Test_realtimePublisherModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("realtime channel の Publisher を提供する", func(t *testing.T) {
			t.Parallel()

			var publisher publisherbndry.Publisher

			validateGraph(t, append(commonDeps(), realtimePublisherModule(), fx.Populate(&publisher))...)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未配線では Publisher が解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			var publisher publisherbndry.Publisher

			require.Error(t, fx.ValidateApp(append(commonDeps(), fx.Populate(&publisher), fx.NopLogger)...))
		})
	})
}

func Test_provideRealtimePublisher(t *testing.T) {
	t.Parallel()

	log := mock_realtime.NewMockEventLogStore(gomock.NewController(t))
	assert.NotNil(t, provideRealtimePublisher(log, testFanout(t),
		observability.NewNoopTracerFactory(t), observability.NewNoopRealtimeMetrics(t)))
}
