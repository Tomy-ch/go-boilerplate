package event_test

import (
	"encoding/json"
	"testing"

	"go-boilerplate/internal/domain/kernel/money"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/usecase/purchase/event"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustPrice は、テスト用に十進文字列（ドル）から非負の money.Price を構築します。
//
//nolint:unparam // テスト補助ヘルパー。現行の呼び出しは同一値だが用途は可変
func mustPrice(t *testing.T, s string) money.Price {
	t.Helper()
	p, err := money.NewPrice(decimaltestkit.MustParse(t, s))
	require.NoError(t, err)
	return p
}

func TestBuildCreated(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入の自己完結スナップショットJSONを生成する", func(t *testing.T) {
			t.Parallel()

			productA := uuid.NewTestFromSalt(t, "bp_product")
			entity, err := domainpurchase.New(
				uuid.NewTestFromSalt(t, "bp_id"),
				"bp-code",
				uuid.NewTestFromSalt(t, "bp_user"),
				[]domainpurchase.DetailInput{{ID: uuid.NewTestFromSalt(t, "bp_d"), ProductID: productA, Quantity: 2}},
				[]domainpurchase.LockedProduct{domainpurchase.NewLockedProduct(productA, mustPrice(t, "800"), 20)},
			)
			require.NoError(t, err)

			payload, perr := event.BuildCreated(entity)
			require.NoError(t, perr)

			var decoded struct {
				PurchaseID     string `json:"purchaseId"`
				Code           string `json:"code"`
				UserID         string `json:"userId"`
				StatusCode     int    `json:"statusCode"`
				SubtotalAmount int    `json:"subtotalAmount"`
				TaxAmount      int    `json:"taxAmount"`
				ShippingFee    int    `json:"shippingFee"`
				TotalAmount    int    `json:"totalAmount"`
				Details        []struct {
					ProductID string `json:"productId"`
					Quantity  int    `json:"quantity"`
					UnitPrice string `json:"unitPrice"`
				} `json:"details"`
			}
			require.NoError(t, json.Unmarshal(payload, &decoded))
			assert.Equal(t, entity.ID().String(), decoded.PurchaseID)
			assert.Equal(t, "bp-code", decoded.Code)
			assert.Equal(t, entity.UserID().String(), decoded.UserID)
			assert.Equal(t, domainpurchase.StatusCodeUnprocessed, decoded.StatusCode)
			// subtotal=160000 / tax=16000（切り捨て10%）/ shipping=500 / total=176500
			assert.Equal(t, 160000, decoded.SubtotalAmount)
			assert.Equal(t, 16000, decoded.TaxAmount)
			assert.Equal(t, 500, decoded.ShippingFee)
			assert.Equal(t, 176500, decoded.TotalAmount)
			require.Len(t, decoded.Details, 1)
			assert.Equal(t, productA.String(), decoded.Details[0].ProductID)
			assert.Equal(t, 2, decoded.Details[0].Quantity)
			assert.Equal(t, "800", decoded.Details[0].UnitPrice)
		})
	})
}
