package idempotency

import (
	"testing"

	mock_idempotencyuc "go-boilerplate/internal/usecase/idempotency/mock"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_Deps_metrics(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Metrics が nil の場合、no-op 実装を返す", func(t *testing.T) {
			t.Parallel()
			d := Deps{}

			assert.Equal(t, nopMetrics{}, d.metrics())
		})

		t.Run("Metrics が設定済みの場合、その実装をそのまま返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			impl := mock_idempotencyuc.NewMockMetrics(ctrl)
			d := Deps{Metrics: impl}

			assert.Same(t, impl, d.metrics())
		})
	})
}
