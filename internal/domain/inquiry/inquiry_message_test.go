package inquiry

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
)

func newTestMessageAttributes(t *testing.T) MessageAttributes {
	t.Helper()
	return MessageAttributes{
		Author:   newTestAuthor(t),
		Body:     "問い合わせ本文",
		Sequence: 1,
	}
}

func newTestMessage(t *testing.T) *Message {
	t.Helper()
	m, err := ReconstructMessage(uuidtestkit.NewTestFromSalt(t, "message"), newTestMessageAttributes(t))
	require.NoError(t, err)
	return m
}

func TestReconstructMessage(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("作成日時を含めて全属性を保持して再構築する", func(t *testing.T) {
			t.Parallel()
			id := uuidtestkit.NewTestFromSalt(t, "message")
			attrs := newTestMessageAttributes(t)
			attrs.Sequence = 7
			attrs.CreatedAt = time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)

			m, err := ReconstructMessage(id, attrs)
			require.NoError(t, err)
			assert.Equal(t, id, m.ID())
			assert.Equal(t, attrs.Author, m.Author())
			assert.Equal(t, attrs.Body, m.Body())
			assert.Equal(t, attrs.Sequence, m.Sequence())
			assert.Equal(t, attrs.CreatedAt, m.CreatedAt())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保存済みでも不変条件を緩めずErrInvalidSequenceを返す", func(t *testing.T) {
			t.Parallel()
			attrs := newTestMessageAttributes(t)
			attrs.Sequence = 0
			attrs.CreatedAt = time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)

			_, err := ReconstructMessage(uuidtestkit.NewTestFromSalt(t, "message"), attrs)
			require.ErrorIs(t, err, ErrInvalidSequence)
		})
	})
}

func Test_newMessage(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("2つの入口が共有する検証を通ると属性を保持した集約を返す", func(t *testing.T) {
			t.Parallel()
			id := uuidtestkit.NewTestFromSalt(t, "message")
			attrs := newTestMessageAttributes(t)

			m, err := newMessage(id, attrs)
			require.NoError(t, err)
			assert.Equal(t, id, m.ID())
			assert.Equal(t, attrs.Body, m.Body())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDが未設定ならErrInvalidMessageIDを返す", func(t *testing.T) {
			t.Parallel()
			_, err := newMessage(uuid.UUID{}, newTestMessageAttributes(t))
			require.ErrorIs(t, err, ErrInvalidMessageID)
		})

		t.Run("送り手の主体が未設定ならErrInvalidAuthorSubjectを返す", func(t *testing.T) {
			t.Parallel()
			attrs := newTestMessageAttributes(t)
			attrs.Author = Author{kind: AuthorKindUser}

			_, err := newMessage(uuidtestkit.NewTestFromSalt(t, "message"), attrs)
			require.ErrorIs(t, err, ErrInvalidAuthorSubject)
		})
	})
}

func Test_validateBody(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("下限文字数ちょうどの本文を受け入れる", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateBody(strings.Repeat("a", minBodyLength)))
		})

		t.Run("上限文字数ちょうどの本文を受け入れる", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateBody(strings.Repeat("a", maxBodyLength)))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空の本文はErrEmptyBodyを返す", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateBody(""), ErrEmptyBody)
		})

		t.Run("上限文字数を超える本文はErrBodyTooLongを返す", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateBody(strings.Repeat("a", maxBodyLength+1)), ErrBodyTooLong)
		})
	})
}

func TestMessage_ID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成に用いたIDを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, uuidtestkit.NewTestFromSalt(t, "message"), newTestMessage(t).ID())
		})
	})
}

func TestMessage_Author(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成に用いた送り手を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, newTestAuthor(t), newTestMessage(t).Author())
		})
	})
}

func TestMessage_Body(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成に用いた本文を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "問い合わせ本文", newTestMessage(t).Body())
		})
	})
}

func TestMessage_Sequence(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成に用いた位置を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, int64(1), newTestMessage(t).Sequence())
		})
	})
}

func TestMessage_CreatedAt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("作成日時を渡さずに組み立てるとゼロ値を返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, newTestMessage(t).CreatedAt().IsZero())
		})

		t.Run("再構築時に設定した作成日時を返す", func(t *testing.T) {
			t.Parallel()
			attrs := newTestMessageAttributes(t)
			attrs.CreatedAt = time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)

			m, err := ReconstructMessage(uuidtestkit.NewTestFromSalt(t, "message"), attrs)
			require.NoError(t, err)
			assert.Equal(t, attrs.CreatedAt, m.CreatedAt())
		})
	})
}

func TestMessage_IsFrom(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		subjectID := func(t *testing.T) uuid.UUID {
			t.Helper()
			return uuidtestkit.NewTestFromSalt(t, "subject")
		}

		t.Run("種別と主体の双方が一致すればtrueを返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, newTestMessage(t).IsFrom(AuthorKindUser, subjectID(t)))
		})

		t.Run("種別だけ一致し主体が異なればfalseを返す", func(t *testing.T) {
			t.Parallel()
			other := uuidtestkit.NewTestFromSalt(t, "other")
			assert.False(t, newTestMessage(t).IsFrom(AuthorKindUser, other))
		})

		t.Run("主体だけ一致し種別が異なればfalseを返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, newTestMessage(t).IsFrom(AuthorKindOperator, subjectID(t)))
		})

		t.Run("種別も主体も異なればfalseを返す", func(t *testing.T) {
			t.Parallel()
			other := uuidtestkit.NewTestFromSalt(t, "other")
			assert.False(t, newTestMessage(t).IsFrom(AuthorKindOperator, other))
		})
	})
}
