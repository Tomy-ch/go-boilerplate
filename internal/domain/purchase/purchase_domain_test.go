package purchase

import (
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/kernel/money"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustPrice は、テスト用に十進文字列（ドル）から非負の money.Price を構築します。
func mustPrice(t *testing.T, s string) money.Price {
	t.Helper()
	p, err := money.NewPrice(decimaltestkit.MustParse(t, s))
	require.NoError(t, err)
	return p
}

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
		NewLockedProduct(productA, mustPrice(t, "800"), 20),
		NewLockedProduct(productB, mustPrice(t, "15"), 10),
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
			assert.Equal(t, "800", details[0].UnitPrice().String())
			assert.Equal(t, "15", details[1].UnitPrice().String())
		})

		t.Run("税額は切り捨てで丸められる", func(t *testing.T) {
			t.Parallel()

			id := uuid.NewTestFromSalt(t, "p_round")
			userID := uuid.NewTestFromSalt(t, "u_round")
			productID := uuid.NewTestFromSalt(t, "prod_round")
			// unit_price=$1.05 → subtotal=105セント → tax=105*10/100=10.5 → 切り捨て 10
			inputs := []DetailInput{{ID: uuid.NewTestFromSalt(t, "d_round"), ProductID: productID, Quantity: 1}}
			locked := []LockedProduct{NewLockedProduct(productID, mustPrice(t, "1.05"), 5)}

			actual, err := New(id, "code-round", userID, inputs, locked)
			require.NoError(t, err)
			assert.Equal(t, 10, actual.TaxAmount())
		})

		t.Run("サブセント単価は小計算出時に決済スケール(整数セント)へ切り捨てられる", func(t *testing.T) {
			t.Parallel()

			id := uuid.NewTestFromSalt(t, "p_subcent")
			userID := uuid.NewTestFromSalt(t, "u_subcent")
			productID := uuid.NewTestFromSalt(t, "prod_subcent")
			// unit_price=$1.005（サブセント）× 3 = $3.015 → 切り捨てで 301 セント
			inputs := []DetailInput{{ID: uuid.NewTestFromSalt(t, "d_subcent"), ProductID: productID, Quantity: 3}}
			locked := []LockedProduct{NewLockedProduct(productID, mustPrice(t, "1.005"), 5)}

			actual, err := New(id, "code-subcent", userID, inputs, locked)
			require.NoError(t, err)
			assert.Equal(t, 301, actual.SubtotalAmount())
			assert.Equal(t, "1.005", actual.Details()[0].UnitPrice().String())
		})

		t.Run("数量が最小値(1)の場合でも購入を生成する", func(t *testing.T) {
			t.Parallel()

			id := uuid.NewTestFromSalt(t, "p_min")
			userID := uuid.NewTestFromSalt(t, "u_min")
			productID := uuid.NewTestFromSalt(t, "prod_min")
			inputs := []DetailInput{{ID: uuid.NewTestFromSalt(t, "d_min"), ProductID: productID, Quantity: minQuantity}}
			locked := []LockedProduct{NewLockedProduct(productID, mustPrice(t, "10"), 1)}

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
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("codeが空の場合、ErrInvalidCodeを返す", func(t *testing.T) {
			t.Parallel()
			id, _, userID, inputs, locked := validNewArgs(t)
			actual, err := New(id, "", userID, inputs, locked)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidCode)
		})

		t.Run("userIDがゼロ値の場合、ErrInvalidUserIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, _, inputs, locked := validNewArgs(t)
			actual, err := New(id, code, uuid.UUID{}, inputs, locked)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidUserID)
		})

		t.Run("明細が空の場合、ErrEmptyDetailsを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, _, locked := validNewArgs(t)
			actual, err := New(id, code, userID, nil, locked)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrEmptyDetails)
		})

		t.Run("数量が0以下の場合、ErrInvalidQuantityを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, inputs, locked := validNewArgs(t)
			inputs[0].Quantity = 0
			actual, err := New(id, code, userID, inputs, locked)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidQuantity)
		})

		t.Run("明細IDがゼロ値の場合、ErrInvalidIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, inputs, locked := validNewArgs(t)
			inputs[0].ID = uuid.UUID{}
			actual, err := New(id, code, userID, inputs, locked)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("同一productIDが重複する場合、ErrDuplicateProductIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, inputs, locked := validNewArgs(t)
			inputs[1].ProductID = inputs[0].ProductID
			actual, err := New(id, code, userID, inputs, locked)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrDuplicateProductID)
		})

		t.Run("明細に対応するロック済み商品が無い場合、ErrProductNotFoundを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, inputs, _ := validNewArgs(t)
			// productB のロックを欠く
			locked := []LockedProduct{NewLockedProduct(inputs[0].ProductID, mustPrice(t, "800"), 20)}
			actual, err := New(id, code, userID, inputs, locked)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrProductNotFound)
		})

		t.Run("要求数量が在庫を超える場合、ErrInsufficientStockを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, inputs, _ := validNewArgs(t)
			inputs[0].Quantity = 999
			locked := []LockedProduct{
				NewLockedProduct(inputs[0].ProductID, mustPrice(t, "800"), 20),
				NewLockedProduct(inputs[1].ProductID, mustPrice(t, "15"), 10),
			}
			actual, err := New(id, code, userID, inputs, locked)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInsufficientStock)
			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("単価が巨大で小計が決済スケール(int64)を超える場合、ErrInvalidAmountを返す", func(t *testing.T) {
			t.Parallel()
			id := uuid.NewTestFromSalt(t, "p_overflow")
			userID := uuid.NewTestFromSalt(t, "u_overflow")
			productID := uuid.NewTestFromSalt(t, "prod_overflow")
			// $92233720368547758.08 × 100 = 9223372036854775808 セント（MaxInt64 超）
			inputs := []DetailInput{{ID: uuid.NewTestFromSalt(t, "d_overflow"), ProductID: productID, Quantity: 1}}
			locked := []LockedProduct{NewLockedProduct(productID, mustPrice(t, "92233720368547758.08"), 5)}

			actual, err := New(id, "code-overflow", userID, inputs, locked)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidAmount)
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
			NewPurchaseDetail(uuid.NewTestFromSalt(t, "rc_d1"), uuid.NewTestFromSalt(t, "rc_p1"), 2, mustPrice(t, "800")),
		}
		orderedAt := time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC)
		return id, "rc-code", userID, statusID, details, orderedAt
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な入力の場合、永続化済みの購入を再構築する", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(
				id,
				code,
				userID,
				statusID,
				StatusCodeUnprocessed,
				160000,
				16000,
				500,
				176500,
				details,
				orderedAt,
				nil,
				nil,
				nil,
				nil,
			)
			require.NoError(t, err)
			assert.Equal(t, id, actual.ID())
			assert.Equal(t, code, actual.Code())
			assert.Equal(t, userID, actual.UserID())
			assert.Equal(t, statusID, actual.StatusID())
			assert.Equal(t, 160000, actual.SubtotalAmount())
			assert.Equal(t, 16000, actual.TaxAmount())
			assert.Equal(t, 500, actual.ShippingFee())
			assert.Equal(t, 176500, actual.TotalAmount())
			assert.Equal(t, StatusCodeUnprocessed, actual.StatusCode())
			assert.Equal(t, orderedAt, actual.OrderedAt())
			assert.Nil(t, actual.CanceledAt())
			require.Len(t, actual.Details(), 1)
			assert.Equal(t, details[0].ProductID(), actual.Details()[0].ProductID())
			assert.True(t, details[0].UnitPrice().Decimal().Equal(actual.Details()[0].UnitPrice().Decimal()))
		})

		t.Run("支払い後にキャンセルされた購入（paidAtとcanceledAtの両方セット）を再構築できる", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			paidAt := orderedAt
			canceledAt := orderedAt
			actual, err := Reconstruct(
				id, code, userID, statusID, StatusCodeCanceled, 160000, 16000, 500, 176500, details, orderedAt,
				&paidAt, &canceledAt, nil, nil,
			)
			require.NoError(t, err)
			require.NotNil(t, actual.PaidAt())
			assert.Equal(t, paidAt, *actual.PaidAt())
			require.NotNil(t, actual.CanceledAt())
		})

		t.Run("発送後に配達された購入（paidAtとshippedAtとdeliveredAtがセット）を再構築できる", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			paidAt := orderedAt
			shippedAt := orderedAt.Add(24 * time.Hour)
			deliveredAt := orderedAt.Add(48 * time.Hour)
			actual, err := Reconstruct(
				id, code, userID, statusID, StatusCodeShipped, 160000, 16000, 500, 176500, details, orderedAt,
				&paidAt, nil, &shippedAt, &deliveredAt,
			)
			require.NoError(t, err)
			require.NotNil(t, actual.PaidAt())
			require.NotNil(t, actual.ShippedAt())
			assert.Equal(t, shippedAt, *actual.ShippedAt())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("statusIDがゼロ値の場合、ErrInvalidStatusIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, _, details, orderedAt := valid(t)
			actual, err := Reconstruct(
				id,
				code,
				userID,
				uuid.UUID{},
				StatusCodeUnprocessed,
				160000,
				16000,
				500,
				176500,
				details,
				orderedAt,
				nil,
				nil,
				nil,
				nil,
			)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStatusID)
		})

		t.Run("金額が負の場合、ErrInvalidAmountを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(
				id,
				code,
				userID,
				statusID,
				StatusCodeUnprocessed,
				-1,
				16000,
				500,
				176500,
				details,
				orderedAt,
				nil,
				nil,
				nil,
				nil,
			)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidAmount)
		})

		t.Run("明細が空の場合、ErrEmptyDetailsを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, _, orderedAt := valid(t)
			actual, err := Reconstruct(
				id,
				code,
				userID,
				statusID,
				StatusCodeUnprocessed,
				160000,
				16000,
				500,
				176500,
				nil,
				orderedAt,
				nil,
				nil,
				nil,
				nil,
			)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrEmptyDetails)
		})

		t.Run("IDがゼロ値の場合、ErrInvalidIDを返す", func(t *testing.T) {
			t.Parallel()
			_, code, userID, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(
				uuid.UUID{},
				code,
				userID,
				statusID,
				StatusCodeUnprocessed,
				160000,
				16000,
				500,
				176500,
				details,
				orderedAt,
				nil,
				nil,
				nil,
				nil,
			)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("codeが空の場合、ErrInvalidCodeを返す", func(t *testing.T) {
			t.Parallel()
			id, _, userID, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(
				id,
				"",
				userID,
				statusID,
				StatusCodeUnprocessed,
				160000,
				16000,
				500,
				176500,
				details,
				orderedAt,
				nil,
				nil,
				nil,
				nil,
			)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidCode)
		})

		t.Run("userIDがゼロ値の場合、ErrInvalidUserIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, _, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(
				id,
				code,
				uuid.UUID{},
				statusID,
				StatusCodeUnprocessed,
				160000,
				16000,
				500,
				176500,
				details,
				orderedAt,
				nil,
				nil,
				nil,
				nil,
			)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidUserID)
		})

		t.Run("税額が負の場合、ErrInvalidAmountを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(
				id,
				code,
				userID,
				statusID,
				StatusCodeUnprocessed,
				160000,
				-1,
				500,
				176500,
				details,
				orderedAt,
				nil,
				nil,
				nil,
				nil,
			)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidAmount)
		})

		t.Run("送料が負の場合、ErrInvalidAmountを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(
				id,
				code,
				userID,
				statusID,
				StatusCodeUnprocessed,
				160000,
				16000,
				-1,
				176500,
				details,
				orderedAt,
				nil,
				nil,
				nil,
				nil,
			)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidAmount)
		})

		t.Run("合計額が負の場合、ErrInvalidAmountを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(
				id,
				code,
				userID,
				statusID,
				StatusCodeUnprocessed,
				160000,
				16000,
				500,
				-1,
				details,
				orderedAt,
				nil,
				nil,
				nil,
				nil,
			)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidAmount)
		})

		t.Run("statusCodeが不正（未処理未満）の場合、ErrInvalidStatusIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(id, code, userID, statusID, 0, 160000, 16000, 500, 176500, details, orderedAt, nil, nil, nil, nil)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStatusID)
		})

		t.Run("キャンセルstatusなのにcanceledAtがnilの場合、ErrInvalidStatusIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(
				id,
				code,
				userID,
				statusID,
				StatusCodeCanceled,
				160000,
				16000,
				500,
				176500,
				details,
				orderedAt,
				nil,
				nil,
				nil,
				nil,
			)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStatusID)
		})

		t.Run("支払い済みstatusなのにpaidAtがnilの場合、ErrInvalidStatusIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			actual, err := Reconstruct(id, code, userID, statusID, StatusCodePaid, 160000, 16000, 500, 176500, details, orderedAt, nil, nil, nil, nil)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStatusID)
		})

		t.Run("発送済みstatusなのにshippedAtがnilの場合、ErrInvalidStatusIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			paidAt := orderedAt
			actual, err := Reconstruct(
				id, code, userID, statusID, StatusCodeShipped, 160000, 16000, 500, 176500, details, orderedAt,
				&paidAt, nil, nil, nil,
			)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStatusID)
		})

		t.Run("キャンセルstatusなのにshippedAtがセット済みの場合、ErrInvalidStatusIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			canceledAt := orderedAt
			shippedAt := orderedAt
			actual, err := Reconstruct(
				id, code, userID, statusID, StatusCodeCanceled, 160000, 16000, 500, 176500, details, orderedAt,
				nil, &canceledAt, &shippedAt, nil,
			)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStatusID)
		})

		t.Run("キャンセルstatusなのにdeliveredAtがセット済みの場合、ErrInvalidStatusIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			canceledAt := orderedAt
			shippedAt := orderedAt
			deliveredAt := orderedAt
			actual, err := Reconstruct(
				id, code, userID, statusID, StatusCodeCanceled, 160000, 16000, 500, 176500, details, orderedAt,
				nil, &canceledAt, &shippedAt, &deliveredAt,
			)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStatusID)
		})

		t.Run("発送済みstatusなのにpaidAtがnilの場合、ErrInvalidStatusIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			shippedAt := orderedAt
			actual, err := Reconstruct(
				id, code, userID, statusID, StatusCodeShipped, 160000, 16000, 500, 176500, details, orderedAt,
				nil, nil, &shippedAt, nil,
			)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStatusID)
		})

		t.Run("deliveredAtがセット済みなのにshippedAtがnilの場合、ErrInvalidStatusIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			paidAt := orderedAt
			deliveredAt := orderedAt
			actual, err := Reconstruct(
				id, code, userID, statusID, StatusCodePaid, 160000, 16000, 500, 176500, details, orderedAt,
				&paidAt, nil, nil, &deliveredAt,
			)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStatusID)
		})

		t.Run("canceledAtがセット済みなのにキャンセルstatusでない場合、ErrInvalidStatusIDを返す", func(t *testing.T) {
			t.Parallel()
			id, code, userID, statusID, details, orderedAt := valid(t)
			canceledAt := orderedAt
			actual, err := Reconstruct(
				id,
				code,
				userID,
				statusID,
				StatusCodeUnprocessed,
				160000,
				16000,
				500,
				176500,
				details,
				orderedAt,
				nil,
				&canceledAt,
				nil,
				nil,
			)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStatusID)
		})
	})
}

func TestPurchase_Cancel(t *testing.T) {
	t.Parallel()

	build := func(t *testing.T, statusCode int, canceledAt, shippedAt, deliveredAt *time.Time) *Purchase {
		t.Helper()
		details := []PurchaseDetail{
			NewPurchaseDetail(uuid.NewTestFromSalt(t, "cancel_d1"), uuid.NewTestFromSalt(t, "cancel_p1"), 2, mustPrice(t, "800")),
		}
		p, err := Reconstruct(
			uuid.NewTestFromSalt(t, "cancel_id"), "cancel-code",
			uuid.NewTestFromSalt(t, "cancel_user"), uuid.NewTestFromSalt(t, "cancel_status"),
			statusCode, 160000, 16000, 500, 176500, details,
			time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
			nil, canceledAt, shippedAt, deliveredAt,
		)
		require.NoError(t, err)
		return p
	}
	now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル可能状態からキャンセルすると、statusCodeがキャンセルになりcanceledAtがセットされる", func(t *testing.T) {
			t.Parallel()
			p := build(t, StatusCodeUnprocessed, nil, nil, nil)
			err := p.Cancel(now)
			require.NoError(t, err)
			assert.Equal(t, StatusCodeCanceled, p.StatusCode())
			require.NotNil(t, p.CanceledAt())
			assert.Equal(t, now, *p.CanceledAt())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既にキャンセル済み（statusCode）の場合、ErrAlreadyCanceledを返す", func(t *testing.T) {
			t.Parallel()
			p := build(t, StatusCodeCanceled, &now, nil, nil)
			require.ErrorIs(t, p.Cancel(now), ErrAlreadyCanceled)
		})

		t.Run("完了の場合、ErrCancelNotAllowedを返す", func(t *testing.T) {
			t.Parallel()
			p := build(t, StatusCodeCompleted, nil, nil, nil)
			require.ErrorIs(t, p.Cancel(now), ErrCancelNotAllowed)
		})

		t.Run("発送済み（shippedAtセット済）の場合、ErrCancelNotAllowedを返す", func(t *testing.T) {
			t.Parallel()
			p := build(t, StatusCodeUnprocessed, nil, &now, nil)
			require.ErrorIs(t, p.Cancel(now), ErrCancelNotAllowed)
		})

		t.Run("配達済み（shippedAtとdeliveredAtセット済）の場合、ErrCancelNotAllowedを返す", func(t *testing.T) {
			t.Parallel()
			p := build(t, StatusCodeUnprocessed, nil, &now, &now)
			require.ErrorIs(t, p.Cancel(now), ErrCancelNotAllowed)
		})
	})
}

func TestPurchase_Pay(t *testing.T) {
	t.Parallel()

	build := func(t *testing.T, statusCode int, paidAt, canceledAt, shippedAt, deliveredAt *time.Time) *Purchase {
		t.Helper()
		details := []PurchaseDetail{
			NewPurchaseDetail(uuid.NewTestFromSalt(t, "pay_d1"), uuid.NewTestFromSalt(t, "pay_p1"), 2, mustPrice(t, "800")),
		}
		p, err := Reconstruct(
			uuid.NewTestFromSalt(t, "pay_id"), "pay-code",
			uuid.NewTestFromSalt(t, "pay_user"), uuid.NewTestFromSalt(t, "pay_status"),
			statusCode, 160000, 16000, 500, 176500, details,
			time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
			paidAt, canceledAt, shippedAt, deliveredAt,
		)
		require.NoError(t, err)
		return p
	}
	now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未払い相当から支払うと、statusCodeが支払い済みになりpaidAtがセットされる", func(t *testing.T) {
			t.Parallel()
			p := build(t, StatusCodeUnprocessed, nil, nil, nil, nil)
			err := p.Pay(now)
			require.NoError(t, err)
			assert.Equal(t, StatusCodePaid, p.StatusCode())
			require.NotNil(t, p.PaidAt())
			assert.Equal(t, now, *p.PaidAt())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既に支払い済み（statusCode）の場合、ErrAlreadyPaidを返す", func(t *testing.T) {
			t.Parallel()
			p := build(t, StatusCodePaid, &now, nil, nil, nil)
			require.ErrorIs(t, p.Pay(now), ErrAlreadyPaid)
		})

		t.Run("キャンセル済みの場合、ErrPayNotAllowedを返す", func(t *testing.T) {
			t.Parallel()
			p := build(t, StatusCodeCanceled, nil, &now, nil, nil)
			require.ErrorIs(t, p.Pay(now), ErrPayNotAllowed)
		})

		t.Run("完了の場合、ErrPayNotAllowedを返す", func(t *testing.T) {
			t.Parallel()
			p := build(t, StatusCodeCompleted, nil, nil, nil, nil)
			require.ErrorIs(t, p.Pay(now), ErrPayNotAllowed)
		})

		t.Run("発送済み（shippedAtセット済）の場合、ErrPayNotAllowedを返す", func(t *testing.T) {
			t.Parallel()
			p := build(t, StatusCodeUnprocessed, nil, nil, &now, nil)
			require.ErrorIs(t, p.Pay(now), ErrPayNotAllowed)
		})

		t.Run("配達済み（shippedAtとdeliveredAtセット済）の場合、ErrPayNotAllowedを返す", func(t *testing.T) {
			t.Parallel()
			p := build(t, StatusCodeUnprocessed, nil, nil, &now, &now)
			require.ErrorIs(t, p.Pay(now), ErrPayNotAllowed)
		})
	})
}

func TestLockedProduct_Price(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ロック時のサブセント価格スナップショットを返す", func(t *testing.T) {
			t.Parallel()

			l := NewLockedProduct(uuid.NewTestFromSalt(t, "lp_price"), mustPrice(t, "19.995"), 5)
			assert.Equal(t, "19.995", l.Price().String())
			assert.True(t, l.Price().Decimal().Equal(decimaltestkit.MustParse(t, "19.995")))
		})
	})
}

func TestPurchaseDetail_UnitPrice(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("明細のサブセント単価スナップショットを返す", func(t *testing.T) {
			t.Parallel()

			d := NewPurchaseDetail(
				uuid.NewTestFromSalt(t, "pd_id"),
				uuid.NewTestFromSalt(t, "pd_product"),
				2, mustPrice(t, "1.005"),
			)
			assert.Equal(t, "1.005", d.UnitPrice().String())
			assert.True(t, d.UnitPrice().Decimal().Equal(decimaltestkit.MustParse(t, "1.005")))
		})
	})
}

func TestPurchase_Ship(t *testing.T) {
	t.Parallel()

	build := func(t *testing.T, statusCode int, paidAt, canceledAt, shippedAt, deliveredAt *time.Time) *Purchase {
		t.Helper()
		details := []PurchaseDetail{
			NewPurchaseDetail(uuid.NewTestFromSalt(t, "ship_d1"), uuid.NewTestFromSalt(t, "ship_p1"), 2, mustPrice(t, "800")),
		}
		p, err := Reconstruct(
			uuid.NewTestFromSalt(t, "ship_id"), "ship-code",
			uuid.NewTestFromSalt(t, "ship_user"), uuid.NewTestFromSalt(t, "ship_status"),
			statusCode, 160000, 16000, 500, 176500, details,
			time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
			paidAt, canceledAt, shippedAt, deliveredAt,
		)
		require.NoError(t, err)
		return p
	}
	paidAt := time.Date(2026, time.July, 25, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("支払い済みから発送すると、statusCodeが発送済みになりshippedAtがセットされる", func(t *testing.T) {
			t.Parallel()
			p := build(t, StatusCodePaid, &paidAt, nil, nil, nil)
			err := p.Ship(now)
			require.NoError(t, err)
			assert.Equal(t, StatusCodeShipped, p.StatusCode())
			require.NotNil(t, p.ShippedAt())
			assert.Equal(t, now, *p.ShippedAt())
		})

		t.Run("発送してもpaidAtは保持される", func(t *testing.T) {
			t.Parallel()
			p := build(t, StatusCodePaid, &paidAt, nil, nil, nil)
			require.NoError(t, p.Ship(now))
			require.NotNil(t, p.PaidAt())
			assert.Equal(t, paidAt, *p.PaidAt())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既に発送済み（statusCode）の場合、ErrAlreadyShippedを返す", func(t *testing.T) {
			t.Parallel()
			p := build(t, StatusCodeShipped, &paidAt, nil, &now, nil)
			require.ErrorIs(t, p.Ship(now), ErrAlreadyShipped)
		})

		t.Run("未払い（未処理）の場合、ErrShipNotAllowedを返す", func(t *testing.T) {
			t.Parallel()
			p := build(t, StatusCodeUnprocessed, nil, nil, nil, nil)
			require.ErrorIs(t, p.Ship(now), ErrShipNotAllowed)
		})

		t.Run("キャンセル済みの場合、ErrShipNotAllowedを返す", func(t *testing.T) {
			t.Parallel()
			p := build(t, StatusCodeCanceled, nil, &now, nil, nil)
			require.ErrorIs(t, p.Ship(now), ErrShipNotAllowed)
		})

		t.Run("完了の場合、ErrShipNotAllowedを返す", func(t *testing.T) {
			t.Parallel()
			p := build(t, StatusCodeCompleted, nil, nil, nil, nil)
			require.ErrorIs(t, p.Ship(now), ErrShipNotAllowed)
		})

		t.Run("配達済みの場合、二重発送ではなくErrShipNotAllowedを返す", func(t *testing.T) {
			t.Parallel()
			deliveredAt := time.Date(2026, time.July, 27, 0, 0, 0, 0, time.UTC)
			p := build(t, StatusCodeDelivered, &paidAt, nil, &now, &deliveredAt)
			require.ErrorIs(t, p.Ship(now), ErrShipNotAllowed)
		})

		t.Run("遷移に失敗した場合、statusCodeとshippedAtを変更しない", func(t *testing.T) {
			t.Parallel()
			p := build(t, StatusCodeUnprocessed, nil, nil, nil, nil)
			require.Error(t, p.Ship(now))
			assert.Equal(t, StatusCodeUnprocessed, p.StatusCode())
			assert.Nil(t, p.ShippedAt())
		})
	})
}

func TestPurchase_ShippedAt(t *testing.T) {
	t.Parallel()

	build := func(t *testing.T, shippedAt *time.Time) *Purchase {
		t.Helper()
		details := []PurchaseDetail{
			NewPurchaseDetail(uuid.NewTestFromSalt(t, "sa_d1"), uuid.NewTestFromSalt(t, "sa_p1"), 2, mustPrice(t, "800")),
		}
		statusCode := StatusCodePaid
		paidAt := time.Date(2026, time.July, 25, 0, 0, 0, 0, time.UTC)
		if shippedAt != nil {
			statusCode = StatusCodeShipped
		}
		p, err := Reconstruct(
			uuid.NewTestFromSalt(t, "sa_id"), "sa-code",
			uuid.NewTestFromSalt(t, "sa_user"), uuid.NewTestFromSalt(t, "sa_status"),
			statusCode, 160000, 16000, 500, 176500, details,
			time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
			&paidAt, nil, shippedAt, nil,
		)
		require.NoError(t, err)
		return p
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未発送の場合、nilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, build(t, nil).ShippedAt())
		})

		t.Run("発送済みの場合、内部状態を保護するためコピーを返す", func(t *testing.T) {
			shippedAt := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)
			p := build(t, &shippedAt)

			t.Run("再構築時に渡したポインタを変更しても、購入のshippedAtは変わらない", func(t *testing.T) {
				t.Parallel()

				original := shippedAt
				shippedAt = time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC)

				require.NotNil(t, p.ShippedAt())
				assert.Equal(t, original, *p.ShippedAt())
			})

			t.Run("ShippedAtの返り値を変更しても、購入のshippedAtは変わらない", func(t *testing.T) {
				t.Parallel()

				original := *p.ShippedAt()

				got := p.ShippedAt()
				*got = time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC)

				require.NotNil(t, p.ShippedAt())
				assert.NotEqual(t, *got, *p.ShippedAt())
				assert.Equal(t, original, *p.ShippedAt())
			})
		})
	})
}

func TestLockedProduct_ID(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestLockedProduct_Quantity(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestNewLockedProduct(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestNewPurchaseDetail(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestPurchaseDetail_ID(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestPurchaseDetail_ProductID(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestPurchaseDetail_Quantity(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestPurchase_CanceledAt(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestPurchase_Code(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestPurchase_Details(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestPurchase_ID(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestPurchase_OrderedAt(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestPurchase_PaidAt(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestPurchase_ShippingFee(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestPurchase_StatusCode(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestPurchase_StatusID(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestPurchase_SubtotalAmount(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestPurchase_TaxAmount(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestPurchase_TotalAmount(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestPurchase_UserID(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
