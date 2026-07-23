package idempotency

import (
	"context"
	"testing"

	mock_idempotencyuc "go-boilerplate/internal/usecase/idempotency/mock"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_nopMetrics(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Metrics 未配線時の no-op 実装は全カウンタを安全に呼べる", func(t *testing.T) {
			t.Parallel()

			// Metrics 未配線 → dispatcher が返す no-op 実装で、全カウンタが panic せず呼べることを固定する
			// （メトリクス任意＝nil-safety の end-to-end 契約）。
			m := Deps{}.metrics()
			ctx := context.Background()

			assert.NotPanics(t, func() {
				m.IncHit(ctx, "op")
				m.IncMiss(ctx, "op")
				m.IncConflict(ctx, "op")
				m.IncFingerprintMismatch(ctx, "op")
				m.IncClaimFailure(ctx, "op")
				m.IncCompleteFailure(ctx, "op")
			})
		})
	})
}

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
