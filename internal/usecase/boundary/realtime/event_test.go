package realtime

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validEvent() DeliveryEvent {
	return DeliveryEvent{
		EventID:       "01J0000000000000000000000",
		StreamID:      "stream-1",
		Sequence:      7,
		Type:          "inquiry.message.appended.v1",
		OccurredAt:    time.Date(2026, time.August, 29, 1, 2, 3, 0, time.UTC),
		SchemaVersion: 1,
		Payload:       json.RawMessage(`{"body":"hi"}`),
	}
}

// eventWithPayloadOfSerializedSize は、直列化後の大きさがちょうど size byte になる event を返します。
func eventWithPayloadOfSerializedSize(t *testing.T, size int) DeliveryEvent {
	t.Helper()

	e := validEvent()
	e.Payload = json.RawMessage(`""`)
	base, err := e.MarshalJSON()
	require.NoError(t, err)
	require.Less(t, len(base), size)

	e.Payload = json.RawMessage(`"` + strings.Repeat("x", size-len(base)) + `"`)
	b, err := e.MarshalJSON()
	require.NoError(t, err)
	require.Len(t, b, size)

	return e
}

func TestSequence_String(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "0", Sequence(0).String())
	assert.Equal(t, "42", Sequence(42).String())
	assert.Equal(t, "9223372036854775807", Sequence(math.MaxInt64).String(), "ゼロ埋めなしの 10 進")
}

func TestParseDeliveryEvent(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("MarshalJSON の出力から同じ封筒へ戻る", func(t *testing.T) {
			t.Parallel()

			want := validEvent()
			b, err := want.MarshalJSON()
			require.NoError(t, err)

			got, err := ParseDeliveryEvent(b)
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JSON でなければ ErrInvalidEvent を返す", func(t *testing.T) {
			t.Parallel()

			_, err := ParseDeliveryEvent([]byte("not json"))
			require.ErrorIs(t, err, ErrInvalidEvent)
		})

		t.Run("sequence が 10 進整数でなければ ErrInvalidEvent を返す", func(t *testing.T) {
			t.Parallel()

			_, err := ParseDeliveryEvent([]byte(`{"eventId":"e","streamId":"s","sequence":"seven"}`))
			require.ErrorIs(t, err, ErrInvalidEvent)
			assert.Contains(t, err.Error(), "decimal")
		})
	})
}

func TestDeliveryEvent_MarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("sequence は 10 進文字列、時刻は UTC の RFC 3339 で直列化される", func(t *testing.T) {
			t.Parallel()

			b, err := validEvent().MarshalJSON()
			require.NoError(t, err)
			assert.JSONEq(t, `{"eventId":"01J0000000000000000000000","streamId":"stream-1","sequence":"7",`+
				`"type":"inquiry.message.appended.v1","occurredAt":"2026-08-29T01:02:03Z","schemaVersion":1,"payload":{"body":"hi"}}`, string(b))
		})

		t.Run("payload が空なら null になる", func(t *testing.T) {
			t.Parallel()

			e := validEvent()
			e.Payload = nil
			b, err := e.MarshalJSON()
			require.NoError(t, err)
			assert.Contains(t, string(b), `"payload":null`)
		})

		t.Run("時刻は UTC へ正規化される", func(t *testing.T) {
			t.Parallel()

			e := validEvent()
			e.OccurredAt = time.Date(2026, time.August, 29, 10, 2, 3, 0, time.FixedZone("JST", 9*60*60))
			b, err := e.MarshalJSON()
			require.NoError(t, err)
			assert.Contains(t, string(b), `"occurredAt":"2026-08-29T01:02:03Z"`)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("payload が JSON でなければ ErrInvalidEvent を返す", func(t *testing.T) {
			t.Parallel()

			e := validEvent()
			e.Payload = json.RawMessage(`{broken`)
			_, err := e.MarshalJSON()
			require.ErrorIs(t, err, ErrInvalidEvent)
		})
	})
}

func TestDeliveryEvent_Validate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全ての項目が揃っていれば nil を返す", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, validEvent().Validate())
		})

		t.Run("payload が無くても封筒として成立する", func(t *testing.T) {
			t.Parallel()

			e := validEvent()
			e.Payload = nil
			require.NoError(t, e.Validate())
		})

		t.Run("直列化後がちょうど上限なら受け付ける", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, eventWithPayloadOfSerializedSize(t, MaxSerializedBytes).Validate())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("直列化後が上限を 1 byte 超えると ErrPayloadTooLarge を返す", func(t *testing.T) {
			t.Parallel()

			err := eventWithPayloadOfSerializedSize(t, MaxSerializedBytes+1).Validate()
			require.ErrorIs(t, err, ErrPayloadTooLarge)
		})

		t.Run("event id が空なら ErrInvalidEvent を返す", func(t *testing.T) {
			t.Parallel()

			e := validEvent()
			e.EventID = ""
			require.ErrorIs(t, e.Validate(), ErrInvalidEvent)
		})

		t.Run("stream id が空なら ErrInvalidEvent を返す", func(t *testing.T) {
			t.Parallel()

			e := validEvent()
			e.StreamID = ""
			require.ErrorIs(t, e.Validate(), ErrInvalidEvent)
		})

		t.Run("sequence が 0 なら ErrInvalidEvent を返す", func(t *testing.T) {
			t.Parallel()

			e := validEvent()
			e.Sequence = 0
			require.ErrorIs(t, e.Validate(), ErrInvalidEvent)
		})

		t.Run("sequence が負なら ErrInvalidEvent を返す", func(t *testing.T) {
			t.Parallel()

			e := validEvent()
			e.Sequence = -1
			require.ErrorIs(t, e.Validate(), ErrInvalidEvent)
		})

		t.Run("type が空なら ErrInvalidEvent を返す", func(t *testing.T) {
			t.Parallel()

			e := validEvent()
			e.Type = ""
			require.ErrorIs(t, e.Validate(), ErrInvalidEvent)
		})

		t.Run("occurredAt がゼロ値なら ErrInvalidEvent を返す", func(t *testing.T) {
			t.Parallel()

			e := validEvent()
			e.OccurredAt = time.Time{}
			require.ErrorIs(t, e.Validate(), ErrInvalidEvent)
		})

		t.Run("payload が JSON でなければ ErrInvalidEvent を返す", func(t *testing.T) {
			t.Parallel()

			e := validEvent()
			e.Payload = json.RawMessage(`not json`)
			require.ErrorIs(t, e.Validate(), ErrInvalidEvent)
		})
	})
}
