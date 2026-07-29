package worker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	bw "go-boilerplate/internal/usecase/boundary/worker"
	"go-boilerplate/internal/usecase/boundary/worker/testkit"
)

func TestEngine_markProgress(t *testing.T) {
	t.Parallel()

	noop := handlerFunc(func(context.Context, bw.Message) error { return nil })

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("記録済みの進捗時刻を現在時刻へ進める", func(t *testing.T) {
			t.Parallel()

			w := testWorker{name: "w", cons: testkit.NewFake(), handler: noop}
			eng := newTestEngine(t, baseSettings(), w)
			eng.progress.Store(time.Now().Add(-time.Hour).UnixNano())
			before := time.Now().UnixNano()

			eng.markProgress()

			got := eng.progress.Load()
			assert.GreaterOrEqual(t, got, before)
			assert.LessOrEqual(t, got, time.Now().UnixNano())
		})

		t.Run("進捗が stale になった engine の readiness を回復させる", func(t *testing.T) {
			t.Parallel()

			w := testWorker{name: "w", cons: testkit.NewFake(), handler: noop}
			set := baseSettings()
			set.ProgressStaleAfter = time.Minute
			eng := newTestEngine(t, set, w)
			eng.active.Store(true)
			eng.progress.Store(time.Now().Add(-time.Hour).UnixNano())
			assert.False(t, eng.Healthy())

			eng.markProgress()

			assert.True(t, eng.Healthy())
		})
	})
}
