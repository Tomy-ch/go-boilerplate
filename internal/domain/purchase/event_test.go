package purchase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
)

func allEventTypes() []EventType {
	return []EventType{EventCreated, EventPaid, EventCanceled, EventShipped, EventDelivered}
}

func TestEventType_Name(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("事象の名前を過去形で返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "created", EventCreated.Name())
			assert.Equal(t, "paid", EventPaid.Name())
			assert.Equal(t, "canceled", EventCanceled.Name())
			assert.Equal(t, "shipped", EventShipped.Name())
			assert.Equal(t, "delivered", EventDelivered.Name())
		})

		t.Run("名前は互いに重複しない", func(t *testing.T) {
			t.Parallel()

			seen := map[string]struct{}{}
			for _, e := range allEventTypes() {
				_, dup := seen[e.Name()]
				require.False(t, dup, "duplicated name: %s", e.Name())
				seen[e.Name()] = struct{}{}
			}
		})
	})
}

func Test_newEvent(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡した事象・購入ID・時刻をそのまま保持する", func(t *testing.T) {
			t.Parallel()

			id := uuidtestkit.NewTestFromSalt(t, "ev_purchase")
			at := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)

			actual := newEvent(EventPaid, id, at)
			assert.Equal(t, EventPaid, actual.Type())
			assert.Equal(t, id, actual.PurchaseID())
			assert.Equal(t, at, actual.OccurredAt())
		})
	})
}

func TestEvent_Type(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("遷移が返した事象の種別を返す", func(t *testing.T) {
			t.Parallel()

			now := time.Date(2026, time.July, 25, 0, 0, 0, 0, time.UTC)
			id, code, userID, inputs, locked := validNewArgs(t)
			p, err := New(id, code, userID, inputs, locked)
			require.NoError(t, err)

			ev, cerr := p.Cancel(now)
			require.NoError(t, cerr)
			assert.Equal(t, EventCanceled, ev.Type())
		})
	})
}

func TestEvent_PurchaseID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("事象が起きた購入のIDを返す", func(t *testing.T) {
			t.Parallel()

			now := time.Date(2026, time.July, 25, 0, 0, 0, 0, time.UTC)
			id, code, userID, inputs, locked := validNewArgs(t)
			p, err := New(id, code, userID, inputs, locked)
			require.NoError(t, err)

			ev, perr := p.Pay(now)
			require.NoError(t, perr)
			assert.Equal(t, p.ID(), ev.PurchaseID())
		})
	})
}

func TestEvent_OccurredAt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("遷移に用いた時刻と同一の時刻を返す", func(t *testing.T) {
			t.Parallel()

			now := time.Date(2026, time.July, 25, 9, 30, 0, 0, time.UTC)
			id, code, userID, inputs, locked := validNewArgs(t)
			p, err := New(id, code, userID, inputs, locked)
			require.NoError(t, err)

			ev, perr := p.Pay(now)
			require.NoError(t, perr)
			assert.Equal(t, now, ev.OccurredAt())
			// 遷移が記録した時刻とイベントの時刻が同一であることを固定する。
			require.NotNil(t, p.PaidAt())
			assert.Equal(t, *p.PaidAt(), ev.OccurredAt())
		})
	})
}
