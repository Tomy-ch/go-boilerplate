package idempotency

import (
	"context"
	"testing"

	mock_idempotencyuc "go-boilerplate/internal/usecase/idempotency/mock"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_nopGCMetrics(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GCMetrics 未配線時の no-op 実装は全カウンタを安全に呼べる", func(t *testing.T) {
			t.Parallel()

			// GCMetrics 未配線 → dispatcher が返す no-op 実装で全カウンタが安全に呼べることを固定する。
			m := (&gcUsecase{}).metrics()
			ctx := context.Background()

			assert.NotPanics(t, func() {
				m.IncExpiredCleanup(ctx, 3)
				m.IncExpiredCleanupFailure(ctx)
			})
		})
	})
}

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
