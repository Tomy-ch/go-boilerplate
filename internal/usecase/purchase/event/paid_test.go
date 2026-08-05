package event_test

import (
	"encoding/json"
	"testing"
	"time"

	domainpurchase "go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/usecase/purchase/event"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPaid(t *testing.T) {
	t.Parallel()

	// unprocessed は、未処理ステータスの再構築済み購入を salt から生成するローカルヘルパーです。
	unprocessed := func(t *testing.T, salt string) *domainpurchase.Purchase {
		t.Helper()
		details := []domainpurchase.PurchaseDetail{
			domainpurchase.NewPurchaseDetail(uuidtestkit.NewTestFromSalt(t, salt+"_d"), domainpurchase.PurchaseDetailAttributes{
				ProductID: uuidtestkit.NewTestFromSalt(t, salt+"_product"),
				Quantity:  2,
				UnitPrice: mustPrice(t, "800"),
			}),
		}
		entity, err := domainpurchase.Reconstruct(uuidtestkit.NewTestFromSalt(t, salt+"_id"), domainpurchase.Attributes{
			Code:           salt + "-code",
			UserID:         uuidtestkit.NewTestFromSalt(t, salt+"_user"),
			StatusID:       uuidtestkit.NewTestFromSalt(t, salt+"_status"),
			StatusCode:     domainpurchase.StatusUnprocessed.Code(),
			SubtotalAmount: 160000,
			TaxAmount:      16000,
			ShippingFee:    500,
			TotalAmount:    176500,
			Details:        details,
			OrderedAt:      time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
			PaidAt:         nil,
			CanceledAt:     nil,
			ShippedAt:      nil,
			DeliveredAt:    nil,
		})
		require.NoError(t, err)
		return entity
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("支払い済み購入の自己完結スナップショットJSONを生成する", func(t *testing.T) {
			t.Parallel()

			entity := unprocessed(t, "bp")
			now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
			require.NoError(t, entity.Pay(now))

			payload, perr := event.BuildPaid(entity)
			require.NoError(t, perr)

			var decoded struct {
				PurchaseID string `json:"purchaseId"`
				Code       string `json:"code"`
				UserID     string `json:"userId"`
				StatusCode int    `json:"statusCode"`
				PaidAt     string `json:"paidAt"`
			}
			require.NoError(t, json.Unmarshal(payload, &decoded))
			assert.Equal(t, entity.ID().String(), decoded.PurchaseID)
			assert.Equal(t, "bp-code", decoded.Code)
			assert.Equal(t, entity.UserID().String(), decoded.UserID)
			assert.Equal(t, domainpurchase.StatusPaid.Code(), decoded.StatusCode)
			assert.Equal(t, now.Format(time.RFC3339Nano), decoded.PaidAt)
		})

		t.Run("paidAtがnilの購入はpaidAtが空文字列になる", func(t *testing.T) {
			t.Parallel()

			// Pay を呼ばず paidAt 未セットのまま payload 化し、防御的な空文字列分岐を検証する。
			payload, perr := event.BuildPaid(unprocessed(t, "bp_nil"))
			require.NoError(t, perr)

			var decoded struct {
				PaidAt string `json:"paidAt"`
			}
			require.NoError(t, json.Unmarshal(payload, &decoded))
			assert.Empty(t, decoded.PaidAt)
		})
	})
}
