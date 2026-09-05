package discontinuation

import (
	"testing"

	"go-boilerplate/internal/domain/purchase"

	"github.com/stretchr/testify/require"
)

func newTestStatus(t *testing.T, code int) purchase.Status {
	t.Helper()
	s, err := purchase.NewStatus(code)
	require.NoError(t, err)

	return s
}

func TestEnsureDiscontinuable(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入が 1 件も無い場合、エラーを返さない", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, EnsureDiscontinuable(nil))
		})

		t.Run("終端のステータスだけの場合、エラーを返さない", func(t *testing.T) {
			t.Parallel()

			statuses := []purchase.Status{
				newTestStatus(t, purchase.StatusCompleted.Code()),
				newTestStatus(t, purchase.StatusCanceled.Code()),
				newTestStatus(t, purchase.StatusDelivered.Code()),
			}

			require.NoError(t, EnsureDiscontinuable(statuses))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("進行中のステータスが 1 件でもある場合、ErrInProgressPurchaseExistsを返す", func(t *testing.T) {
			t.Parallel()

			statuses := []purchase.Status{
				newTestStatus(t, purchase.StatusCompleted.Code()),
				newTestStatus(t, purchase.StatusPaid.Code()),
			}

			require.ErrorIs(t, EnsureDiscontinuable(statuses), ErrInProgressPurchaseExists)
		})

		t.Run("未処理だけの場合も、ErrInProgressPurchaseExistsを返す", func(t *testing.T) {
			t.Parallel()

			statuses := []purchase.Status{newTestStatus(t, purchase.StatusUnprocessed.Code())}

			require.ErrorIs(t, EnsureDiscontinuable(statuses), ErrInProgressPurchaseExists)
		})
	})
}
