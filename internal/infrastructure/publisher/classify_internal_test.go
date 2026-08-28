package publisher

import (
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_classifyOutcome(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("成功は分類せず nil を返す", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, classifyOutcome(&httpclient.Response{StatusCode: 200}, nil))
		})

		t.Run("4xx は再試行しても変わらない失敗として分類する", func(t *testing.T) {
			t.Parallel()

			err := classifyOutcome(&httpclient.Response{StatusCode: 400}, apperror.ErrInvalidArgument)
			assert.ErrorIs(t, err, apperror.ErrPermanent)
			// 元のエラーは調査のために保持する。
			assert.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("429 は一時失敗として分類する", func(t *testing.T) {
			t.Parallel()

			err := classifyOutcome(&httpclient.Response{StatusCode: 429}, apperror.ErrTooManyRequests)
			assert.ErrorIs(t, err, apperror.ErrRetryable)
		})

		t.Run("5xx は一時失敗として分類する", func(t *testing.T) {
			t.Parallel()

			err := classifyOutcome(&httpclient.Response{StatusCode: 503}, apperror.ErrUnavailable)
			assert.ErrorIs(t, err, apperror.ErrRetryable)
		})

		t.Run("応答を得られない transport 失敗は一時失敗として分類する", func(t *testing.T) {
			t.Parallel()

			err := classifyOutcome(nil, xerrors.Wrap(apperror.ErrUnavailable, "dial failed"))
			assert.ErrorIs(t, err, apperror.ErrRetryable)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ctx キャンセルは停止であって配送の失敗ではないため分類しない", func(t *testing.T) {
			t.Parallel()

			canceled := xerrors.Wrap(apperror.ErrCanceled, "canceled")
			err := classifyOutcome(nil, canceled)

			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.NotErrorIs(t, err, apperror.ErrPermanent)
			assert.NotErrorIs(t, err, apperror.ErrRetryable)
		})
	})
}
