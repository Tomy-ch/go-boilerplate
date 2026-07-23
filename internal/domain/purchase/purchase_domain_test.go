package purchase

import (
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validNewArgs は、New の有効な入力（id / code / userID / 明細入力 / ロック済み在庫）を返すテストヘルパーです。
func validNewArgs(t *testing.T) (uuid.UUID, string, uuid.UUID, []DetailInput, []LockedProduct) {
	t.Helper()
	id := uuid.NewTestFromSalt(t, "purchase_id")
	userID := uuid.NewTestFromSalt(t, "user_id")
	productA := uuid.NewTestFromSalt(t, "product_a")
	productB := uuid.NewTestFromSalt(t, "product_b")
	inputs := []DetailInput{
		{ID: uuid.NewTestFromSalt(t, "detail_a"), ProductID: productA, Quantity: 2},
		{ID: uuid.NewTestFromSalt(t, "detail_b"), ProductID: productB, Quantity: 1},
	}
	locked := []LockedProduct{
		NewLockedProduct(productA, 80000, 20),
		NewLockedProduct(productB, 1500, 10),
	}
	return id, "purchase-code-001", userID, inputs, locked
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な入力の場合、金額を計算し未処理ステータスで購入を生成する", func(t *testing.T) {
			t.Parallel()

			id, code, userID, inputs, locked := validNewArgs(t)
			actual, err := New(id, code, userID, inputs, locked)
			require.NoError(t, err)

			assert.Equal(t, id, actual.ID())
			assert.Equal(t, code, actual.Code())
			assert.Equal(t, userID, actual.UserID())
			assert.Equal(t, StatusCodeUnprocessed, actual.StatusCode())
			// subtotal = 80000*2 + 1500*1 = 161500
			assert.Equal(t, 161500, actual.SubtotalAmount())
			// tax = 161500 * 10 / 100 = 16150（切り捨て）
			assert.Equal(t, 16150, actual.TaxAmount())
			assert.Equal(t, shippingFeeCents, actual.ShippingFee())
			// total = 161500 + 16150 + 500 = 178150
			assert.Equal(t, 178150, actual.TotalAmount())
			require.Len(t, actual.Details(), 2)
		})

		t.Run("明細の単価は対応するロック済み商品の価格スナップショットになる", func(t *testing.T) {
			t.Parallel()

			id, code, userID, inputs, locked := validNewArgs(t)
			actual, err := New(id, code, userID, inputs, locked)
			require.NoError(t, err)

			details := actual.Details()
			assert.Equal(t, inputs[0].ProductID, details[0].ProductID())
			assert.Equal(t, 2, details[0].Quantity())
			assert.Equal(t, 80000, details[0].UnitPrice())
			assert.Equal(t, 1500, details[1].UnitPrice())
		})

		t.Run("税額は切り捨てで丸められる", func(t *testing.T) {
			t.Parallel()

			id := uuid.NewTestFromSalt(t, "p_round")
			userID := uuid.NewTestFromSalt(t, "u_round")
			productID := uuid.NewTestFromSalt(t, "prod_round")
			// subtotal=105 → tax=105*10/100=10.5 → 切り捨て 10
			inputs := []DetailInput{{ID: uuid.NewTestFromSalt(t, "d_round"), ProductID: productID, Quantity: 1}}
			locked := []LockedProduct{NewLockedProduct(productID, 105, 5)}

			actual, err := New(id, "code-round", userID, inputs, locked)
			require.NoError(t, err)
			assert.Equal(t, 10, actual.TaxAmount())
		})

		t.Run("数量が最小値(1)の場合でも購入を生成する", func(t *testing.T) {
			t.Parallel()

			id := uuid.NewTestFromSalt(t, "p_min")
			userID := uuid.NewTestFromSalt(t, "u_min")
			productID := uuid.NewTestFromSalt(t, "prod_min")
			inputs := []DetailInput{{ID: uuid.NewTestFromSalt(t, "d_min"), ProductID: productID, Quantity: minQuantity}}
			locked := []LockedProduct{NewLockedProduct(productID, 1000, 1)}

			actual, err := New(id, "code-min", userID, inputs, locked)
			require.NoError(t, err)
			require.Len(t, actual.Details(), 1)
			assert.Equal(t, minQuantity, actual.Details()[0].Quantity())
		})

		t.Run("Detailsの返却値を変更しても内部状態は不変", func(t *testing.T) {
			t.Parallel()

			id, code, userID, inputs, locked := validNewArgs(t)
			actual, err := New(id, code, userID, inputs, locked)
			require.NoError(t, err)

			returned := actual.Details()
			returned[0] = PurchaseDetail{}
			// 返却スライスへの破壊的変更が内部へ波及しないこと。
			assert.Equal(t, inputs[0].ProductID, actual.Details()[0].ProductID())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDがゼロ値の場合、ErrInvalidIDを返す", func(t *testing.T) {
			t.Parallel()
			_, code, userID, inputs, locked := validNewArgs(t)
			actual, err := New(uuid.UUID{}, code, userID, inputs, locked)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("codeが空の場合、ErrInvalidCodeを返す", func(t *testing.T) {
			t.Parallel()
			id, _, userID, inputs, locked := validNewArgs(t)
			actual, err := New(id, "", userID, inputs, locked)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidCode)
		})

		t.Run("userIDがゼロ値の場合、ErrInvalidUserIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, _, inputs, locked := validNewArgs(t)
			actual, err := New(id, code, uuid.UUID{}, inputs, locked)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidUserID)
		})

		t.Run("明細が空の場合、ErrEmptyDetailsを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, _, locked := validNewArgs(t)
			actual, err := New(id, code, userID, nil, locked)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrEmptyDetails)
		})

		t.Run("数量が0以下の場合、ErrInvalidQuantityを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, inputs, locked := validNewArgs(t)
			inputs[0].Quantity = 0
			actual, err := New(id, code, userID, inputs, locked)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidQuantity)
		})

		t.Run("明細IDがゼロ値の場合、ErrInvalidIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, inputs, locked := validNewArgs(t)
			inputs[0].ID = uuid.UUID{}
			actual, err := New(id, code, userID, inputs, locked)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("同一productIDが重複する場合、ErrDuplicateProductIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, inputs, locked := validNewArgs(t)
			inputs[1].ProductID = inputs[0].ProductID
			actual, err := New(id, code, userID, inputs, locked)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrDuplicateProductID)
		})

		t.Run("明細に対応するロック済み商品が無い場合、ErrProductNotFoundを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, inputs, _ := validNewArgs(t)
			// productB のロックを欠く
			locked := []LockedProduct{NewLockedProduct(inputs[0].ProductID, 80000, 20)}
			actual, err := New(id, code, userID, inputs, locked)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrProductNotFound)
		})

		t.Run("要求数量が在庫を超える場合、ErrInsufficientStockを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, inputs, _ := validNewArgs(t)
			inputs[0].Quantity = 999
			locked := []LockedProduct{
				NewLockedProduct(inputs[0].ProductID, 80000, 20),
				NewLockedProduct(inputs[1].ProductID, 1500, 10),
			}
			actual, err := New(id, code, userID, inputs, locked)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInsufficientStock)
			require.ErrorIs(t, err, apperror.ErrConflict)
		})
	})
}

func TestReconstruct(t *testing.T) {
	t.Parallel()

	valid := func(t *testing.T) (uuid.UUID, string, uuid.UUID, uuid.UUID, []PurchaseDetail, time.Time) {
		t.Helper()
		id := uuid.NewTestFromSalt(t, "rc_id")
		userID := uuid.NewTestFromSalt(t, "rc_user")
		statusID := uuid.NewTestFromSalt(t, "rc_status")
		details := []PurchaseDetail{
			NewPurchaseDetail(uuid.NewTestFromSalt(t, "rc_d1"), uuid.NewTestFromSalt(t, "rc_p1"), 2, 80000),
		}
		orderedAt := time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC)
		return id, "rc-code", userID, statusID, details, orderedAt
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な入力の場合、永続化済みの購入を再構築する", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(id, code, userID, statusID, 160000, 16000, 500, 176500, details, orderedAt)
			require.NoError(t, err)
			assert.Equal(t, id, actual.ID())
			assert.Equal(t, code, actual.Code())
			assert.Equal(t, userID, actual.UserID())
			assert.Equal(t, statusID, actual.StatusID())
			assert.Equal(t, 160000, actual.SubtotalAmount())
			assert.Equal(t, 16000, actual.TaxAmount())
			assert.Equal(t, 500, actual.ShippingFee())
			assert.Equal(t, 176500, actual.TotalAmount())
			assert.Equal(t, orderedAt, actual.OrderedAt())
			require.Len(t, actual.Details(), 1)
			assert.Equal(t, details[0].ProductID(), actual.Details()[0].ProductID())
			assert.Equal(t, details[0].UnitPrice(), actual.Details()[0].UnitPrice())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("statusIDがゼロ値の場合、ErrInvalidStatusIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, _, details, orderedAt := valid(t)
			actual, err := Reconstruct(id, code, userID, uuid.UUID{}, 160000, 16000, 500, 176500, details, orderedAt)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStatusID)
		})

		t.Run("金額が負の場合、ErrInvalidAmountを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(id, code, userID, statusID, -1, 16000, 500, 176500, details, orderedAt)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidAmount)
		})

		t.Run("明細が空の場合、ErrEmptyDetailsを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, _, orderedAt := valid(t)
			actual, err := Reconstruct(id, code, userID, statusID, 160000, 16000, 500, 176500, nil, orderedAt)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrEmptyDetails)
		})

		t.Run("IDがゼロ値の場合、ErrInvalidIDを返す", func(t *testing.T) {
			t.Parallel()
			_, code, userID, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(uuid.UUID{}, code, userID, statusID, 160000, 16000, 500, 176500, details, orderedAt)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("codeが空の場合、ErrInvalidCodeを返す", func(t *testing.T) {
			t.Parallel()
			id, _, userID, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(id, "", userID, statusID, 160000, 16000, 500, 176500, details, orderedAt)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidCode)
		})

		t.Run("userIDがゼロ値の場合、ErrInvalidUserIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, _, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(id, code, uuid.UUID{}, statusID, 160000, 16000, 500, 176500, details, orderedAt)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidUserID)
		})

		t.Run("税額が負の場合、ErrInvalidAmountを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(id, code, userID, statusID, 160000, -1, 500, 176500, details, orderedAt)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidAmount)
		})

		t.Run("送料が負の場合、ErrInvalidAmountを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(id, code, userID, statusID, 160000, 16000, -1, 176500, details, orderedAt)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidAmount)
		})

		t.Run("合計額が負の場合、ErrInvalidAmountを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(id, code, userID, statusID, 160000, 16000, 500, -1, details, orderedAt)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidAmount)
		})
	})
}
