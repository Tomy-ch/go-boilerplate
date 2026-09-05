package testkit_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/internal/usecase/boundary/realtime/testkit"
)

var ticketNow = time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)

func ticket(hash rt.TicketHash, subject string, destination rt.StreamID) rt.StreamTicket {
	return rt.StreamTicket{
		Hash: hash, Subject: subject, Destination: destination,
		IssuedAt: ticketNow, ExpiresAt: ticketNow.Add(5 * time.Minute),
	}
}

func TestNewStreamTicketStore(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空の store を返す", func(t *testing.T) {
			t.Parallel()

			assert.Zero(t, testkit.NewStreamTicketStore().Len())
		})
	})
}

func TestStreamTicketStore_Save(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同じ hash への再保存は上書きになる", func(t *testing.T) {
			t.Parallel()

			s := testkit.NewStreamTicketStore()
			require.NoError(t, s.Save(context.Background(), ticket("h1", "subject-1", "stream-1")))
			require.NoError(t, s.Save(context.Background(), ticket("h1", "subject-2", "stream-1")))

			assert.Equal(t, 1, s.Len())
			got, ok, err := s.Find(context.Background(), "h1", ticketNow)
			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, "subject-2", got.Subject)
		})
	})
}

func TestStreamTicketStore_Find(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期限内の ticket を返す", func(t *testing.T) {
			t.Parallel()

			s := testkit.NewStreamTicketStore()
			require.NoError(t, s.Save(context.Background(), ticket("h1", "subject-1", "stream-1")))

			got, ok, err := s.Find(context.Background(), "h1", ticketNow)
			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, rt.StreamID("stream-1"), got.Destination)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("知らない hash は ok=false を返す", func(t *testing.T) {
			t.Parallel()

			_, ok, err := testkit.NewStreamTicketStore().Find(context.Background(), "missing", ticketNow)
			require.NoError(t, err)
			assert.False(t, ok)
		})

		t.Run("期限に達した ticket は ok=false を返す", func(t *testing.T) {
			t.Parallel()

			s := testkit.NewStreamTicketStore()
			issued := ticket("h1", "subject-1", "stream-1")
			require.NoError(t, s.Save(context.Background(), issued))

			_, ok, err := s.Find(context.Background(), "h1", issued.ExpiresAt)
			require.NoError(t, err)
			assert.False(t, ok, "ExpiresAt ちょうどは期限内ではないこと")
		})
	})
}

func TestStreamTicketStore_Invalidate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("subject と destination が一致する ticket だけを消す", func(t *testing.T) {
			t.Parallel()

			s := testkit.NewStreamTicketStore()
			require.NoError(t, s.Save(context.Background(), ticket("h1", "subject-1", "stream-1")))
			require.NoError(t, s.Save(context.Background(), ticket("h2", "subject-1", "stream-2")))
			require.NoError(t, s.Save(context.Background(), ticket("h3", "subject-2", "stream-1")))

			require.NoError(t, s.Invalidate(context.Background(), "subject-1", "stream-1"))

			assert.Equal(t, 2, s.Len())
			_, ok, err := s.Find(context.Background(), "h1", ticketNow)
			require.NoError(t, err)
			assert.False(t, ok)
			for _, hash := range []rt.TicketHash{"h2", "h3"} {
				_, ok, err := s.Find(context.Background(), hash, ticketNow)
				require.NoError(t, err)
				assert.True(t, ok, "%s は巻き添えにならないこと", hash)
			}
		})

		t.Run("該当が無くてもエラーにならない", func(t *testing.T) {
			t.Parallel()

			assert.NoError(t, testkit.NewStreamTicketStore().Invalidate(context.Background(), "subject-1", "stream-1"))
		})
	})
}

func TestStreamTicketStore_Len(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保持している件数を返す", func(t *testing.T) {
			t.Parallel()

			s := testkit.NewStreamTicketStore()
			require.NoError(t, s.Save(context.Background(), ticket("h1", "subject-1", "stream-1")))
			require.NoError(t, s.Save(context.Background(), ticket("h2", "subject-1", "stream-2")))

			assert.Equal(t, 2, s.Len())
		})
	})
}
