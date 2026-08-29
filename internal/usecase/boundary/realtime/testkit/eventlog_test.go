package testkit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	rt "go-boilerplate/internal/usecase/boundary/realtime"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStream は、テストで使う stream の識別子です。
const testStream = rt.StreamID("stream-1")

// occurred は、event の発生時刻として使う基準時刻です。
var occurred = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

// event は、seq の位置に置く最小限の妥当な封筒を組み立てます。
func event(seq rt.Sequence, id string) rt.DeliveryEvent {
	return rt.DeliveryEvent{
		EventID:       id,
		StreamID:      testStream,
		Sequence:      seq,
		Type:          "sample.thing.created.v1",
		OccurredAt:    occurred,
		SchemaVersion: 1,
		Payload:       json.RawMessage(`{}`),
	}
}

func TestNewEventLog(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空の stream を読んでも event は返らない", func(t *testing.T) {
			t.Parallel()

			res, err := NewEventLog().ReadAfter(context.Background(), rt.ReadAfterQuery{StreamID: testStream})

			require.NoError(t, err)
			assert.Empty(t, res.Events)
		})
	})
}

func TestEventLog_Seed(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("飛び番を置ける", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			l.Seed(event(1, "e1"), event(3, "e3"))

			res, err := l.ReadAfter(context.Background(), rt.ReadAfterQuery{StreamID: testStream})

			require.NoError(t, err)
			require.Len(t, res.Events, 2)
			assert.Equal(t, rt.Sequence(3), res.Events[1].Sequence)
		})

		t.Run("順不同で渡しても sequence 昇順に並ぶ", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			l.Seed(event(2, "e2"), event(1, "e1"))

			res, err := l.ReadAfter(context.Background(), rt.ReadAfterQuery{StreamID: testStream})

			require.NoError(t, err)
			require.Len(t, res.Events, 2)
			assert.Equal(t, rt.Sequence(1), res.Events[0].Sequence)
		})

		t.Run("Validate を通らない封筒も置ける", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			l.Seed(rt.DeliveryEvent{StreamID: testStream, Sequence: 1})

			got, ok, err := l.Find(context.Background(), testStream, 1)

			require.NoError(t, err)
			require.True(t, ok)
			assert.Empty(t, got.EventID)
		})
	})
}

func TestEventLog_Hold(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("解くまで読み取りが返らない", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			l.Seed(event(1, "e1"))
			release := l.Hold()

			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			t.Cleanup(cancel)
			_, err := l.ReadAfter(ctx, rt.ReadAfterQuery{StreamID: testStream})

			require.ErrorIs(t, err, context.DeadlineExceeded)
			release()
		})

		t.Run("解いた後は読み取れる", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			l.Seed(event(1, "e1"))
			l.Hold()()

			res, err := l.ReadAfter(context.Background(), rt.ReadAfterQuery{StreamID: testStream})

			require.NoError(t, err)
			assert.Len(t, res.Events, 1)
		})
	})
}

func Test_EventLog_awaitRelease(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("関門が無ければ待たずに返る", func(t *testing.T) {
			t.Parallel()

			assert.NoError(t, NewEventLog().awaitRelease(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("待っている間に ctx が終わればその error を返す", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			release := l.Hold()
			t.Cleanup(release)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			require.ErrorIs(t, l.awaitRelease(ctx), context.Canceled)
		})
	})
}

func TestEventLog_SetUnavailable(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("false に戻すと読めるようになる", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			l.Seed(event(1, "e1"))
			l.SetUnavailable(true)
			l.SetUnavailable(false)

			_, ok, err := l.Find(context.Background(), testStream, 1)

			require.NoError(t, err)
			assert.True(t, ok)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("true の間は ErrUnavailable を返す", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			l.SetUnavailable(true)

			_, err := l.ReadAfter(context.Background(), rt.ReadAfterQuery{StreamID: testStream})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}

func TestEventLog_Append(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("書いた event を読み出せる", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()

			require.NoError(t, l.Append(context.Background(), event(1, "e1")))

			got, ok, err := l.Find(context.Background(), testStream, 1)
			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, "e1", got.EventID)
		})

		t.Run("同じ位置に同じ EventID を再度書いても成功する", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			require.NoError(t, l.Append(context.Background(), event(1, "e1")))

			assert.NoError(t, l.Append(context.Background(), event(1, "e1")))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同じ位置に異なる EventID を書くと ErrSequenceConflict を返す", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			require.NoError(t, l.Append(context.Background(), event(1, "e1")))

			err := l.Append(context.Background(), event(1, "other"))

			require.ErrorIs(t, err, rt.ErrSequenceConflict)
		})

		t.Run("Validate を通らない封筒は書けない", func(t *testing.T) {
			t.Parallel()

			err := NewEventLog().Append(context.Background(), rt.DeliveryEvent{StreamID: testStream, Sequence: 1})

			require.ErrorIs(t, err, rt.ErrInvalidEvent)
		})

		t.Run("unavailable の間は ErrUnavailable を返す", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			l.SetUnavailable(true)

			err := l.Append(context.Background(), event(1, "e1"))

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}

func TestEventLog_ReadAfter(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("After より後ろだけを昇順で返す", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			l.Seed(event(1, "e1"), event(2, "e2"), event(3, "e3"))

			res, err := l.ReadAfter(context.Background(), rt.ReadAfterQuery{StreamID: testStream, After: 1})

			require.NoError(t, err)
			require.Len(t, res.Events, 2)
			assert.Equal(t, rt.Sequence(2), res.Events[0].Sequence)
			assert.False(t, res.HasMore)
		})

		t.Run("Limit を超えると HasMore が true になる", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			l.Seed(event(1, "e1"), event(2, "e2"))

			res, err := l.ReadAfter(context.Background(), rt.ReadAfterQuery{StreamID: testStream, Limit: 1})

			require.NoError(t, err)
			require.Len(t, res.Events, 1)
			assert.True(t, res.HasMore)
		})

		t.Run("別の stream の event は混ざらない", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			other := event(1, "other")
			other.StreamID = "stream-2"
			l.Seed(event(1, "e1"), other)

			res, err := l.ReadAfter(context.Background(), rt.ReadAfterQuery{StreamID: testStream})

			require.NoError(t, err)
			require.Len(t, res.Events, 1)
			assert.Equal(t, "e1", res.Events[0].EventID)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("unavailable の間は ErrUnavailable を返す", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			l.SetUnavailable(true)

			_, err := l.ReadAfter(context.Background(), rt.ReadAfterQuery{StreamID: testStream})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}

func TestEventLog_Latest(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最後の event を返す", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			l.Seed(event(1, "e1"), event(5, "e5"))

			got, ok, err := l.Latest(context.Background(), testStream)

			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, rt.Sequence(5), got.Sequence)
		})

		t.Run("1 件も無ければ ok が false になる", func(t *testing.T) {
			t.Parallel()

			_, ok, err := NewEventLog().Latest(context.Background(), testStream)

			require.NoError(t, err)
			assert.False(t, ok)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("unavailable の間は ErrUnavailable を返す", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			l.SetUnavailable(true)

			_, _, err := l.Latest(context.Background(), testStream)

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}

func TestEventLog_Find(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定 sequence の event を返す", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			l.Seed(event(1, "e1"), event(2, "e2"))

			got, ok, err := l.Find(context.Background(), testStream, 2)

			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, "e2", got.EventID)
		})

		t.Run("無い位置は ok が false になる", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			l.Seed(event(1, "e1"))

			_, ok, err := l.Find(context.Background(), testStream, 2)

			require.NoError(t, err)
			assert.False(t, ok)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("unavailable の間は ErrUnavailable を返す", func(t *testing.T) {
			t.Parallel()

			l := NewEventLog()
			l.SetUnavailable(true)

			_, _, err := l.Find(context.Background(), testStream, 1)

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}

func Test_insert(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既存より小さい sequence は前に入る", func(t *testing.T) {
			t.Parallel()

			got := insert([]rt.DeliveryEvent{event(2, "e2")}, event(1, "e1"))

			require.Len(t, got, 2)
			assert.Equal(t, rt.Sequence(1), got[0].Sequence)
		})

		t.Run("同じ sequence は置き換えられ件数が増えない", func(t *testing.T) {
			t.Parallel()

			got := insert([]rt.DeliveryEvent{event(1, "e1")}, event(1, "replaced"))

			require.Len(t, got, 1)
			assert.Equal(t, "replaced", got[0].EventID)
		})
	})
}
