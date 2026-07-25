package event_test

import (
	"encoding/json"
	"testing"
	"time"

	domainpurchase "go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/usecase/purchase/event"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCanceled(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済み購入の自己完結スナップショットJSONを生成する", func(t *testing.T) {
			t.Parallel()

			productA := uuid.NewTestFromSalt(t, "bc_product")
			details := []domainpurchase.PurchaseDetail{
				domainpurchase.NewPurchaseDetail(uuid.NewTestFromSalt(t, "bc_d"), productA, 2, mustPrice(t, "800")),
			}
			entity, err := domainpurchase.Reconstruct(
				uuid.NewTestFromSalt(t, "bc_id"), "bc-code",
				uuid.NewTestFromSalt(t, "bc_user"), uuid.NewTestFromSalt(t, "bc_status"),
				domainpurchase.StatusCodeUnprocessed, 160000, 16000, 500, 176500, details,
				time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC), nil, nil, nil, nil,
			)
			require.NoError(t, err)
			now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
			require.NoError(t, entity.Cancel(now))

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
			assert.Equal(t, domainpurchase.StatusCodeCanceled, decoded.StatusCode)
			assert.Equal(t, now.Format(time.RFC3339Nano), decoded.CanceledAt)
		})

		t.Run("canceledAtがnilの購入はcanceledAtが空文字列になる", func(t *testing.T) {
			t.Parallel()

			details := []domainpurchase.PurchaseDetail{
				domainpurchase.NewPurchaseDetail(
					uuid.NewTestFromSalt(t, "bc_nil_d"),
					uuid.NewTestFromSalt(t, "bc_nil_product"),
					2,
					mustPrice(t, "800"),
				),
			}
			entity, err := domainpurchase.Reconstruct(
				uuid.NewTestFromSalt(t, "bc_nil_id"), "bc-nil-code",
				uuid.NewTestFromSalt(t, "bc_nil_user"), uuid.NewTestFromSalt(t, "bc_nil_status"),
				domainpurchase.StatusCodeUnprocessed, 160000, 16000, 500, 176500, details,
				time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC), nil, nil, nil, nil,
			)
			require.NoError(t, err)

			payload, perr := event.BuildCanceled(entity)
			require.NoError(t, perr)

			var decoded struct {
				CanceledAt string `json:"canceledAt"`
			}
			require.NoError(t, json.Unmarshal(payload, &decoded))
			assert.Empty(t, decoded.CanceledAt)
		})
	})
}
