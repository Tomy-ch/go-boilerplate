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

func TestBuildDelivered(t *testing.T) {
	t.Parallel()

	// shipped は、発送済みステータスの再構築済み購入を salt から生成するローカルヘルパーです。
	shipped := func(t *testing.T, salt string) *domainpurchase.Purchase {
		t.Helper()
		details := []domainpurchase.PurchaseDetail{
			domainpurchase.NewPurchaseDetail(
				uuid.NewTestFromSalt(t, salt+"_d"), uuid.NewTestFromSalt(t, salt+"_product"), 2, mustPrice(t, "800"),
			),
		}
		paidAt := time.Date(2026, time.July, 25, 0, 0, 0, 0, time.UTC)
		shippedAt := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)
		entity, err := domainpurchase.Reconstruct(
			uuid.NewTestFromSalt(t, salt+"_id"), salt+"-code",
			uuid.NewTestFromSalt(t, salt+"_user"), uuid.NewTestFromSalt(t, salt+"_status"),
			domainpurchase.StatusCodeShipped, 160000, 16000, 500, 176500, details,
			time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC), &paidAt, nil, &shippedAt, nil,
		)
		require.NoError(t, err)
		return entity
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("配達済み購入の自己完結スナップショットJSONを生成する", func(t *testing.T) {
			t.Parallel()

			entity := shipped(t, "bd")
			now := time.Date(2026, time.July, 28, 9, 0, 0, 0, time.UTC)
			require.NoError(t, entity.Deliver(now))

			payload, perr := event.BuildDelivered(entity)
			require.NoError(t, perr)

			var decoded struct {
				PurchaseID  string `json:"purchaseId"`
				Code        string `json:"code"`
				UserID      string `json:"userId"`
				StatusCode  int    `json:"statusCode"`
				DeliveredAt string `json:"deliveredAt"`
			}
			require.NoError(t, json.Unmarshal(payload, &decoded))
			assert.Equal(t, entity.ID().String(), decoded.PurchaseID)
			assert.Equal(t, "bd-code", decoded.Code)
			assert.Equal(t, entity.UserID().String(), decoded.UserID)
			assert.Equal(t, domainpurchase.StatusCodeDelivered, decoded.StatusCode)
			assert.Equal(t, now.Format(time.RFC3339Nano), decoded.DeliveredAt)
		})

		t.Run("deliveredAtがnilの購入はdeliveredAtが空文字列になる", func(t *testing.T) {
			t.Parallel()

			// Deliver を呼ばず deliveredAt 未セットのまま payload 化し、防御的な空文字列分岐を検証する。
			payload, perr := event.BuildDelivered(shipped(t, "bd_nil"))
			require.NoError(t, perr)

			var decoded struct {
				DeliveredAt string `json:"deliveredAt"`
			}
			require.NoError(t, json.Unmarshal(payload, &decoded))
			assert.Empty(t, decoded.DeliveredAt)
		})
	})
}
