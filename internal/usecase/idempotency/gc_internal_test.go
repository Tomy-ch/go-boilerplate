package idempotency

import (
	"testing"

	mock_idempotencyuc "go-boilerplate/internal/usecase/idempotency/mock"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_gcUsecase_metrics(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("metricsImpl が nil の場合、no-op 実装を返す", func(t *testing.T) {
			t.Parallel()
			g := &gcUsecase{}

			assert.Equal(t, nopGCMetrics{}, g.metrics())
		})

		t.Run("metricsImpl が設定済みの場合、その実装をそのまま返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			impl := mock_idempotencyuc.NewMockGCMetrics(ctrl)
			g := &gcUsecase{metricsImpl: impl}

			assert.Same(t, impl, g.metrics())
		})
	})
}
