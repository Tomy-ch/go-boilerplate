package membership

import (
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/domain/user"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// baseTime は、テスト用ユーザーの作成・更新日時の基準です。
var baseTime = time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

// newActiveUser は、在籍しているユーザーを生成します。
func newActiveUser(t *testing.T) *user.User {
	t.Helper()

	u, err := user.New(uuidtestkit.NewTestFromSalt(t, "membership_user"), user.Attributes{
		Profile: user.Profile{
			FirstName:    "John",
			LastName:     "Doe",
			Email:        "john.doe@example.com",
			Phone:        "1234567890",
			PrefectureID: uuidtestkit.NewTestFromSalt(t, "membership_prefecture"),
			City:         "Shibuya",
			Street:       "1-2-3",
			PostalCode:   "150-0001",
		},
		CreatedAt: baseTime,
		UpdatedAt: baseTime,
	})
	require.NoError(t, err)
	return u
}

// newWithdrawnUser は、退会済みのユーザーを生成します。
func newWithdrawnUser(t *testing.T) *user.User {
	t.Helper()

	u := newActiveUser(t)
	require.NoError(t, u.MarkAsDeleted(baseTime.Add(time.Hour)))
	return u
}

func TestEnsurePurchasable(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("在籍している購入者は購入できる", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, EnsurePurchasable(newActiveUser(t)))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("退会済みの購入者は拒否する", func(t *testing.T) {
			t.Parallel()

			err := EnsurePurchasable(newWithdrawnUser(t))

			require.ErrorIs(t, err, ErrPurchaserWithdrawn)
			assert.ErrorIs(t, err, apperror.ErrConflict)
		})
	})
}

func TestEnsureWithdrawable(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入を持たないユーザーは退会できる", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, EnsureWithdrawable(newActiveUser(t), nil))
		})

		t.Run("終端に達した購入しか持たないユーザーは退会できる", func(t *testing.T) {
			t.Parallel()

			statuses := []purchase.Status{
				purchase.StatusCompleted, purchase.StatusCanceled, purchase.StatusDelivered,
			}

			require.NoError(t, EnsureWithdrawable(newActiveUser(t), statuses))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未処理の購入が残っているユーザーは退会できない", func(t *testing.T) {
			t.Parallel()

			err := EnsureWithdrawable(newActiveUser(t), []purchase.Status{purchase.StatusUnprocessed})

			require.ErrorIs(t, err, ErrInProgressPurchaseExists)
			assert.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("終端に達していない購入が1件でも混じっていれば退会できない", func(t *testing.T) {
			t.Parallel()

			statuses := []purchase.Status{
				purchase.StatusCompleted, purchase.StatusShipped, purchase.StatusDelivered,
			}

			err := EnsureWithdrawable(newActiveUser(t), statuses)

			require.ErrorIs(t, err, ErrInProgressPurchaseExists)
		})

		t.Run("既に退会しているユーザーは退会の対象にならない", func(t *testing.T) {
			t.Parallel()

			err := EnsureWithdrawable(newWithdrawnUser(t), nil)

			require.ErrorIs(t, err, user.ErrAlreadyDeleted)
		})
	})
}
