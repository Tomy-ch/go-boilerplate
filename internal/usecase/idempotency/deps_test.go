package idempotency_test

import (
	"testing"

	"go-boilerplate/internal/usecase/boundary/clock/testkit"
	mock_idempotency "go-boilerplate/internal/usecase/boundary/idempotency/mock"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	"go-boilerplate/internal/usecase/idempotency"
	mock_idempotencyuc "go-boilerplate/internal/usecase/idempotency/mock"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewDeps(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡した依存をそのまま束ねて返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			txm := mock_tx.NewMockManager(ctrl)
			store := mock_idempotency.NewMockStore(ctrl)
			clk := testkit.NewMockClock(t, fixedNow)
			metrics := mock_idempotencyuc.NewMockMetrics(ctrl)

			deps := idempotency.NewDeps(txm, store, clk, metrics)

			assert.Equal(t, txm, deps.Txm)
			assert.Equal(t, store, deps.Store)
			assert.Equal(t, clk, deps.Clock)
			assert.Equal(t, metrics, deps.Metrics)
		})

		t.Run("metrics が nil でも束ねられる", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			txm := mock_tx.NewMockManager(ctrl)
			store := mock_idempotency.NewMockStore(ctrl)
			clk := testkit.NewMockClock(t, fixedNow)

			deps := idempotency.NewDeps(txm, store, clk, nil)

			assert.Nil(t, deps.Metrics)
		})
	})
}
