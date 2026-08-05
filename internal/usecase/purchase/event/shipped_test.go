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

func TestBuildShipped(t *testing.T) {
	t.Parallel()

	// paid は、支払い済みステータスの再構築済み購入を salt から生成するローカルヘルパーです。
	paid := func(t *testing.T, salt string) *domainpurchase.Purchase {
		t.Helper()
		details := []domainpurchase.PurchaseDetail{
			domainpurchase.NewPurchaseDetail(
				uuidtestkit.NewTestFromSalt(t, salt+"_d"), uuidtestkit.NewTestFromSalt(t, salt+"_product"), 2, mustPrice(t, "800"),
			),
		}
		paidAt := time.Date(2026, time.July, 25, 0, 0, 0, 0, time.UTC)
		entity, err := domainpurchase.Reconstruct(
			uuidtestkit.NewTestFromSalt(t, salt+"_id"), salt+"-code",
			uuidtestkit.NewTestFromSalt(t, salt+"_user"), uuidtestkit.NewTestFromSalt(t, salt+"_status"),
			domainpurchase.StatusCodePaid, 160000, 16000, 500, 176500, details,
			time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC), &paidAt, nil, nil, nil,
		)
		require.NoError(t, err)
		return entity
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("発送済み購入の自己完結スナップショットJSONを生成する", func(t *testing.T) {
			t.Parallel()

			entity := paid(t, "bs")
			now := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)
			require.NoError(t, entity.Ship(now))

			payload, perr := event.BuildShipped(entity)
			require.NoError(t, perr)

			var decoded struct {
				PurchaseID string `json:"purchaseId"`
				Code       string `json:"code"`
				UserID     string `json:"userId"`
				StatusCode int    `json:"statusCode"`
				ShippedAt  string `json:"shippedAt"`
			}
			require.NoError(t, json.Unmarshal(payload, &decoded))
			assert.Equal(t, entity.ID().String(), decoded.PurchaseID)
			assert.Equal(t, "bs-code", decoded.Code)
			assert.Equal(t, entity.UserID().String(), decoded.UserID)
			assert.Equal(t, domainpurchase.StatusCodeShipped, decoded.StatusCode)
			assert.Equal(t, now.Format(time.RFC3339Nano), decoded.ShippedAt)
		})

		t.Run("shippedAtがnilの購入はshippedAtが空文字列になる", func(t *testing.T) {
			t.Parallel()

			// Ship を呼ばず shippedAt 未セットのまま payload 化し、防御的な空文字列分岐を検証する。
			payload, perr := event.BuildShipped(paid(t, "bs_nil"))
			require.NoError(t, perr)

			var decoded struct {
				ShippedAt string `json:"shippedAt"`
			}
			require.NoError(t, json.Unmarshal(payload, &decoded))
			assert.Empty(t, decoded.ShippedAt)
		})
	})
}
