package worker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/observability"
	bw "go-boilerplate/internal/usecase/boundary/worker"
	"go-boilerplate/internal/usecase/boundary/worker/testkit"
)

type withTraceCtxKey struct{}

func Test_run_withTrace(t *testing.T) {
	t.Parallel()

	newTestRun := func(t *testing.T) *run {
		t.Helper()
		f := testkit.NewFake()
		w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error { return nil })}
		return newRun(newTestEngine(t, baseSettings(), w), w)
	}

	// producer が traceparent を載せた Attributes を再現する。
	newCarrier := func(t *testing.T) map[string]string {
		t.Helper()
		ctx, end := observability.NewStubSpanContext(t)
		t.Cleanup(end)
		attrs := map[string]string{}
		observability.InjectTraceContextToCarrier(ctx, attrs)
		require.Contains(t, attrs, "traceparent")
		return attrs
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("traceparent を含む Attributes でも親 ctx の値を引き継ぐ", func(t *testing.T) {
			t.Parallel()

			parent := context.WithValue(context.Background(), withTraceCtxKey{}, "v")

			got := newTestRun(t).withTrace(parent, bw.Message{ID: "a", Attributes: newCarrier(t)})

			assert.Equal(t, "v", got.Value(withTraceCtxKey{}))
		})

		t.Run("traceparent を含む Attributes でも親 ctx のキャンセルを引き継ぐ", func(t *testing.T) {
			t.Parallel()

			parent, cancel := context.WithCancel(context.Background())

			// 親から切り離した ctx を返すと停止シグナルが Handle まで届かなくなる。
			got := newTestRun(t).withTrace(parent, bw.Message{ID: "a", Attributes: newCarrier(t)})
			require.NoError(t, got.Err())

			cancel()

			require.ErrorIs(t, got.Err(), context.Canceled)
		})

		t.Run("Attributes が空の場合は ctx をそのまま返す", func(t *testing.T) {
			t.Parallel()

			parent := context.Background()

			got := newTestRun(t).withTrace(parent, bw.Message{ID: "a"})

			assert.Equal(t, parent, got)
		})
	})
}
