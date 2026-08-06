package purchase

import (
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/pkg/decimal"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newDetailRow は、明細の永続化行を構築するテストヘルパーです。
func newDetailRow(t *testing.T, salt string, purchaseID uuid.UUID, unitPrice decimal.Decimal) gen.PurchaseDetails {
	t.Helper()
	return gen.PurchaseDetails{
		ID:         uuidtestkit.NewTestFromSalt(t, salt),
		PurchaseID: purchaseID,
		ProductID:  uuidtestkit.NewTestFromSalt(t, salt+"_product"),
		Quantity:   2,
		UnitPrice:  unitPrice,
	}
}

func Test_toPurchaseDetail(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("永続化行の各項目を明細の値オブジェクトへ写す", func(t *testing.T) {
			t.Parallel()

			purchaseID := uuidtestkit.NewTestFromSalt(t, "map_purchase")
			row := newDetailRow(t, "map_detail", purchaseID, decimaltestkit.MustParse(t, "19.99"))

			actual, err := toPurchaseDetail(row)
			require.NoError(t, err)

			assert.Equal(t, row.ID, actual.ID())
			assert.Equal(t, row.ProductID, actual.ProductID())
			assert.Equal(t, 2, actual.Quantity())
			assert.Equal(t, "19.99", actual.UnitPrice().String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("単価が価格として不正な場合、再構築エラーを返す", func(t *testing.T) {
			t.Parallel()

			purchaseID := uuidtestkit.NewTestFromSalt(t, "map_invalid_purchase")
			row := newDetailRow(t, "map_invalid", purchaseID, decimaltestkit.MustParse(t, "-1"))

			_, err := toPurchaseDetail(row)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func Test_groupPurchaseDetails(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入IDごとに明細を振り分け、取得順を保つ", func(t *testing.T) {
			t.Parallel()

			first := uuidtestkit.NewTestFromSalt(t, "grp_purchase_1")
			second := uuidtestkit.NewTestFromSalt(t, "grp_purchase_2")
			price := decimaltestkit.MustParse(t, "10")
			rows := []*gen.ListPurchaseDetailsByPurchaseIDsRow{
				{PurchaseDetails: newDetailRow(t, "grp_d1", first, price)},
				{PurchaseDetails: newDetailRow(t, "grp_d2", first, price)},
				{PurchaseDetails: newDetailRow(t, "grp_d3", second, price)},
			}

			actual, err := groupPurchaseDetails(rows)
			require.NoError(t, err)

			require.Len(t, actual, 2)
			require.Len(t, actual[first], 2)
			assert.Equal(t, rows[0].PurchaseDetails.ID, actual[first][0].ID())
			assert.Equal(t, rows[1].PurchaseDetails.ID, actual[first][1].ID())
			require.Len(t, actual[second], 1)
			assert.Equal(t, rows[2].PurchaseDetails.ID, actual[second][0].ID())
		})

		t.Run("明細行が空の場合、空のマップを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := groupPurchaseDetails(nil)
			require.NoError(t, err)

			assert.Empty(t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("単価が価格として不正な明細が含まれる場合、再構築エラーを返す", func(t *testing.T) {
			t.Parallel()

			purchaseID := uuidtestkit.NewTestFromSalt(t, "grp_invalid_purchase")
			rows := []*gen.ListPurchaseDetailsByPurchaseIDsRow{
				{PurchaseDetails: newDetailRow(t, "grp_invalid", purchaseID, decimaltestkit.MustParse(t, "-1"))},
			}

			_, err := groupPurchaseDetails(rows)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}
