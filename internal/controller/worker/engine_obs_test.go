package worker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/logging"
	bw "go-boilerplate/internal/usecase/boundary/worker"
	"go-boilerplate/internal/usecase/boundary/worker/testkit"
	"go-boilerplate/pkg/xerrors"
)

func Test_Engine_D3_StructuredLog(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("失敗ログに message id と receive count が構造化フィールドで残る", func(t *testing.T) {
			t.Parallel()

			logger, observed := logging.NewObservedTestLogger(t)
			f := testkit.NewFake()
			f.Enqueue(bw.Message{ID: "a"})
			w := testWorker{name: "w", cons: f, handler: handlerFunc(func(context.Context, bw.Message) error {
				return xerrors.Wrap(apperror.ErrRetryable, "downstream")
			})}

			cancel, done := startEngine(t, baseSettings(), logger, w)
			defer func() { cancel(); <-done }()

			require.Eventually(t, func() bool {
				return observed.FilterMessage("retryable failure, nacked").Len() >= 1
			}, eventually, tick)

			entry := observed.FilterMessage("retryable failure, nacked").All()[0]
			ctxMap := entry.ContextMap()
			assert.Equal(t, "a", ctxMap[logging.MessageIDKey])
			assert.Contains(t, ctxMap, logging.ReceiveCountKey)
		})
	})
}
