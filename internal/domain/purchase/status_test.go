package purchase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStatus(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既知のcodeからステータスを解決する", func(t *testing.T) {
			t.Parallel()

			for _, expected := range allStatuses() {
				actual, err := NewStatus(expected.Code())
				require.NoError(t, err)
				assert.Equal(t, expected, actual)
			}
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未知のcodeはErrInvalidStatusIDを返す", func(t *testing.T) {
			t.Parallel()

			// 0（未設定）と、既知の code の範囲外を両方弾くことを固定する。
			for _, code := range []int{0, 2, 99, -1} {
				_, err := NewStatus(code)
				require.ErrorIs(t, err, ErrInvalidStatusID)
			}
		})
	})
}

func TestStatus_Code(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("永続化に用いる業務キーを返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, 1, StatusUnprocessed.Code())
			assert.Equal(t, 5, StatusCompleted.Code())
			assert.Equal(t, 6, StatusCanceled.Code())
			assert.Equal(t, 7, StatusPaid.Code())
			assert.Equal(t, 8, StatusShipped.Code())
			assert.Equal(t, 9, StatusDelivered.Code())
		})
	})
}

func TestStatus_Name(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("外部へ伝えるための名前を返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "unprocessed", StatusUnprocessed.Name())
			assert.Equal(t, "completed", StatusCompleted.Name())
			assert.Equal(t, "canceled", StatusCanceled.Name())
			assert.Equal(t, "paid", StatusPaid.Name())
			assert.Equal(t, "shipped", StatusShipped.Name())
			assert.Equal(t, "delivered", StatusDelivered.Name())
		})

		t.Run("名前は互いに重複しない", func(t *testing.T) {
			t.Parallel()

			seen := map[string]struct{}{}
			for _, s := range allStatuses() {
				_, dup := seen[s.Name()]
				require.False(t, dup, "duplicated name: %s", s.Name())
				seen[s.Name()] = struct{}{}
			}
		})
	})
}

func TestStatus_IsZero(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ゼロ値はtrue、既知のステータスはfalse", func(t *testing.T) {
			t.Parallel()

			assert.True(t, Status{}.IsZero())
			for _, s := range allStatuses() {
				assert.False(t, s.IsZero(), s.Name())
			}
		})
	})
}

func TestStatus_IsTerminal(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("完了とキャンセルと配達済みだけが終端", func(t *testing.T) {
			t.Parallel()

			assert.True(t, StatusCompleted.IsTerminal())
			assert.True(t, StatusCanceled.IsTerminal())
			assert.True(t, StatusDelivered.IsTerminal())
			assert.False(t, StatusUnprocessed.IsTerminal())
			assert.False(t, StatusPaid.IsTerminal())
			assert.False(t, StatusShipped.IsTerminal())
		})
	})
}

func TestStatus_CanTransitionTo(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("支払いは未処理からのみ到達する", func(t *testing.T) {
			t.Parallel()

			assert.True(t, StatusUnprocessed.CanTransitionTo(StatusPaid))
			assert.False(t, StatusShipped.CanTransitionTo(StatusPaid))
		})

		t.Run("発送は支払い済みからのみ到達する", func(t *testing.T) {
			t.Parallel()

			assert.True(t, StatusPaid.CanTransitionTo(StatusShipped))
			assert.False(t, StatusUnprocessed.CanTransitionTo(StatusShipped))
		})

		t.Run("配達は発送済みからのみ到達する", func(t *testing.T) {
			t.Parallel()

			assert.True(t, StatusShipped.CanTransitionTo(StatusDelivered))
			assert.False(t, StatusPaid.CanTransitionTo(StatusDelivered))
		})

		t.Run("キャンセルは進行中から到達する", func(t *testing.T) {
			t.Parallel()

			assert.True(t, StatusUnprocessed.CanTransitionTo(StatusCanceled))
			assert.True(t, StatusPaid.CanTransitionTo(StatusCanceled))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("終端状態からはどこへも遷移しない", func(t *testing.T) {
			t.Parallel()

			for _, from := range allStatuses() {
				if !from.IsTerminal() {
					continue
				}
				for _, to := range allStatuses() {
					assert.False(t, from.CanTransitionTo(to), "%s -> %s", from.Name(), to.Name())
				}
			}
		})

		t.Run("同じステータスへは遷移しない", func(t *testing.T) {
			t.Parallel()

			for _, s := range allStatuses() {
				assert.False(t, s.CanTransitionTo(s), s.Name())
			}
		})

		t.Run("完了へは遷移できない（遷移メソッドを持たない）", func(t *testing.T) {
			t.Parallel()

			assert.False(t, StatusUnprocessed.CanTransitionTo(StatusCompleted))
			assert.False(t, StatusPaid.CanTransitionTo(StatusCompleted))
		})
	})
}

func Test_allStatuses(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既知のステータスを重複なく列挙し、codeも重複しない", func(t *testing.T) {
			t.Parallel()

			all := allStatuses()
			require.Len(t, all, 6)

			seen := map[int]struct{}{}
			for _, s := range all {
				_, dup := seen[s.Code()]
				require.False(t, dup, "duplicated code: %d", s.Code())
				seen[s.Code()] = struct{}{}
			}
		})
	})
}
