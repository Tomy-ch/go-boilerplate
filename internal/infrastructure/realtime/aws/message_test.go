package aws

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rt "go-boilerplate/internal/usecase/boundary/realtime"
)

func message(kind, body string) sqstypes.Message {
	m := sqstypes.Message{Body: awssdk.String(body), ReceiptHandle: awssdk.String("receipt-1")}
	if kind != "" {
		m.MessageAttributes = map[string]sqstypes.MessageAttributeValue{
			AttrKind: {DataType: awssdk.String(stringAttr), StringValue: awssdk.String(kind)},
		}
	}

	return m
}

func Test_encodeWakeup(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("本文は eventId / streamId / sequence（10 進文字列）のみで、種別属性は wakeup", func(t *testing.T) {
			t.Parallel()

			body, attrs, err := encodeWakeup(rt.Wakeup{EventID: "e1", StreamID: "s1", Sequence: 42})
			require.NoError(t, err)
			assert.JSONEq(t, `{"eventId":"e1","streamId":"s1","sequence":"42"}`, body)
			assert.Equal(t, "wakeup", awssdk.ToString(attrs[AttrKind].StringValue))
			assert.Equal(t, stringAttr, awssdk.ToString(attrs[AttrKind].DataType))
		})
	})
}

func Test_encodeRevocation(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("本文は subject / destination で、種別属性は revocation", func(t *testing.T) {
			t.Parallel()

			body, attrs, err := encodeRevocation(rt.Revocation{Subject: "u1", Destination: "s1"})
			require.NoError(t, err)
			assert.JSONEq(t, `{"subject":"u1","destination":"s1"}`, body)
			assert.Equal(t, "revocation", awssdk.ToString(attrs[AttrKind].StringValue))
		})
	})
}

func Test_kindAttribute(t *testing.T) {
	t.Parallel()

	attrs := kindAttribute(rt.KindWakeup)
	assert.Len(t, attrs, 1)
	assert.Equal(t, "wakeup", awssdk.ToString(attrs[AttrKind].StringValue))
}

func Test_decodeNotification(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("wakeup は EventID / StreamID / Sequence を復元し Receipt を載せる", func(t *testing.T) {
			t.Parallel()

			n := decodeNotification(message("wakeup", `{"eventId":"e1","streamId":"s1","sequence":"42"}`))
			assert.Equal(t, rt.Notification{
				Kind: rt.KindWakeup, Wakeup: rt.Wakeup{EventID: "e1", StreamID: "s1", Sequence: 42}, Receipt: "receipt-1",
			}, n)
		})

		t.Run("revocation は Subject / Destination を復元する", func(t *testing.T) {
			t.Parallel()

			n := decodeNotification(message("revocation", `{"subject":"u1","destination":"s1"}`))
			assert.Equal(t, rt.Notification{
				Kind: rt.KindRevocation, Revocation: rt.Revocation{Subject: "u1", Destination: "s1"}, Receipt: "receipt-1",
			}, n)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		tests := map[string]sqstypes.Message{
			"種別属性が無い":                  message("", `{"eventId":"e1","streamId":"s1","sequence":"42"}`),
			"未知の種別":                    message("other", `{}`),
			"wakeup の本文が JSON でない":     message("wakeup", "not json"),
			"wakeup の sequence が整数でない": message("wakeup", `{"eventId":"e1","streamId":"s1","sequence":"x"}`),
			"revocation の本文が JSON でない": message("revocation", "not json"),
		}

		for name, m := range tests {
			t.Run(name+"なら Kind は空で Receipt だけ載る", func(t *testing.T) {
				t.Parallel()

				n := decodeNotification(m)
				assert.Equal(t, rt.Notification{Receipt: "receipt-1"}, n)
			})
		}
	})
}
