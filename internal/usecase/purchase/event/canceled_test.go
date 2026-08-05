package event_test

import (
	"encoding/json"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/usecase/purchase/event"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCanceled(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済み購入の自己完結スナップショットJSONを生成する", func(t *testing.T) {
			t.Parallel()

			productA := uuidtestkit.NewTestFromSalt(t, "bc_product")
			details := []domainpurchase.PurchaseDetail{
				domainpurchase.NewPurchaseDetail(uuidtestkit.NewTestFromSalt(t, "bc_d"), domainpurchase.PurchaseDetailAttributes{
					ProductID: productA,
					Quantity:  2,
					UnitPrice: mustPrice(t, "800"),
				}),
			}
			entity, err := domainpurchase.Reconstruct(uuidtestkit.NewTestFromSalt(t, "bc_id"), domainpurchase.Attributes{
				Code:           "bc-code",
				UserID:         uuidtestkit.NewTestFromSalt(t, "bc_user"),
				StatusID:       uuidtestkit.NewTestFromSalt(t, "bc_status"),
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
			now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
			_, err = entity.Cancel(now)
			require.NoError(t, err)

			payload, perr := event.BuildCanceled(entity)
			require.NoError(t, perr)

			var decoded struct {
				PurchaseID string `json:"purchaseId"`
				Code       string `json:"code"`
				UserID     string `json:"userId"`
				StatusCode int    `json:"statusCode"`
				CanceledAt string `json:"canceledAt"`
			}
			require.NoError(t, json.Unmarshal(payload, &decoded))
			assert.Equal(t, entity.ID().String(), decoded.PurchaseID)
			assert.Equal(t, "bc-code", decoded.Code)
			assert.Equal(t, entity.UserID().String(), decoded.UserID)
			assert.Equal(t, domainpurchase.StatusCanceled.Code(), decoded.StatusCode)
			assert.Equal(t, now.Format(time.RFC3339Nano), decoded.CanceledAt)
		})

	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセルされていない購入からはpayloadを生成せずErrInternalを返す", func(t *testing.T) {
			t.Parallel()

			details := []domainpurchase.PurchaseDetail{
				domainpurchase.NewPurchaseDetail(uuidtestkit.NewTestFromSalt(t, "bc_nil_d"), domainpurchase.PurchaseDetailAttributes{
					ProductID: uuidtestkit.NewTestFromSalt(t, "bc_nil_product"),
					Quantity:  2,
					UnitPrice: mustPrice(t, "800"),
				}),
			}
			entity, err := domainpurchase.Reconstruct(uuidtestkit.NewTestFromSalt(t, "bc_nil_id"), domainpurchase.Attributes{
				Code:           "bc-nil-code",
				UserID:         uuidtestkit.NewTestFromSalt(t, "bc_nil_user"),
				StatusID:       uuidtestkit.NewTestFromSalt(t, "bc_nil_status"),
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

			payload, perr := event.BuildCanceled(entity)
			require.ErrorIs(t, perr, apperror.ErrInternal)
			assert.Nil(t, payload)
		})
	})
}
