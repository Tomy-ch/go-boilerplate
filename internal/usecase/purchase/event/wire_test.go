package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainpurchase "go-boilerplate/internal/domain/purchase"
)

func TestWireType(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ドメインの事象を版付きの種別へ写像する", func(t *testing.T) {
			t.Parallel()

			cases := map[domainpurchase.EventType]string{
				domainpurchase.EventCreated:   "purchase.created.v1",
				domainpurchase.EventPaid:      "purchase.paid.v1",
				domainpurchase.EventCanceled:  "purchase.canceled.v1",
				domainpurchase.EventShipped:   "purchase.shipped.v1",
				domainpurchase.EventDelivered: "purchase.delivered.v1",
			}
			for domainEvent, expected := range cases {
				actual, err := WireType(domainEvent)
				require.NoError(t, err)
				assert.Equal(t, expected, actual)
			}
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ワイヤ表現が未定義の事象はerrNoWireTypeを返す", func(t *testing.T) {
			t.Parallel()

			// ゼロ値は既知のどの事象でもないため、写像漏れと同じ扱いになる。
			_, err := WireType(domainpurchase.EventType{})
			require.ErrorIs(t, err, errNoWireType)
		})
	})
}
