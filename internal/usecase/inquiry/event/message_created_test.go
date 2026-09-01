package event_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/domain/inquirymessage"
	"go-boilerplate/internal/usecase/inquiry/event"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
)

// newTestMessage は、payload の組み立て元となるメッセージを作ります。
func newTestMessage(t *testing.T, kind inquirymessage.AuthorKind) *inquirymessage.Message {
	t.Helper()
	author, err := inquirymessage.NewAuthor(kind, uuidtestkit.NewTestFromSalt(t, "subject"))
	require.NoError(t, err)
	m, err := inquirymessage.Reconstruct(uuidtestkit.NewTestFromSalt(t, "message"), inquirymessage.Attributes{
		InquiryID: uuidtestkit.NewTestFromSalt(t, "inquiry"),
		Author:    author,
		Body:      "本文",
		Sequence:  3,
		CreatedAt: time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	return m
}

func TestBuildMessageCreated(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("集約から自己完結のpayloadを組み立てる", func(t *testing.T) {
			t.Parallel()
			m := newTestMessage(t, inquirymessage.AuthorKindUser)

			payload, err := event.BuildMessageCreated(m)
			require.NoError(t, err)

			var got map[string]any
			require.NoError(t, json.Unmarshal(payload, &got))
			assert.Equal(t, m.ID().String(), got["messageId"])
			assert.Equal(t, m.InquiryID().String(), got["inquiryId"])
			assert.Equal(t, "本文", got["body"])
			assert.InDelta(t, float64(3), got["sequence"], 0)
			assert.Equal(t, "2026-09-01T10:00:00Z", got["createdAt"])
		})

		t.Run("送り手は種別だけを載せ主体IDを出さない", func(t *testing.T) {
			t.Parallel()
			m := newTestMessage(t, inquirymessage.AuthorKindOperator)

			payload, err := event.BuildMessageCreated(m)
			require.NoError(t, err)

			var got struct {
				Author map[string]any `json:"author"`
			}
			require.NoError(t, json.Unmarshal(payload, &got))
			assert.Equal(t, "operator", got.Author["kind"])
			assert.NotContains(t, got.Author, "subjectId")
			assert.NotContains(t, string(payload), m.Author().SubjectID().String())
		})
	})
}
