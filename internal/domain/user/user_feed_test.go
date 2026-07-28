package user

import (
	"testing"
	"time"

	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
)

func TestNewFeedCursor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("作成日時とIDが保存され、ゲッターから取得できる", func(t *testing.T) {
			t.Parallel()

			createdAt := time.Date(2025, time.January, 2, 3, 4, 5, 600000000, time.UTC)
			id := uuid.NewTestFromSalt(t, "feed_cursor_id")

			actual := NewFeedCursor(createdAt, id)

			assert.Equal(t, createdAt, actual.CreatedAt())
			assert.Equal(t, id, actual.ID())
		})
	})
}

func TestFeedCursor_CreatedAt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("構築時の作成日時をナノ秒精度のまま返す", func(t *testing.T) {
			t.Parallel()

			createdAt := time.Date(2025, time.January, 2, 3, 4, 5, 600000000, time.UTC)

			actual := NewFeedCursor(createdAt, uuid.NewTestFromSalt(t, "feed_cursor_id"))

			assert.Equal(t, createdAt, actual.CreatedAt())
		})
	})
}

func TestFeedCursor_ID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("構築時のユーザーIDを返す", func(t *testing.T) {
			t.Parallel()

			id := uuid.NewTestFromSalt(t, "feed_cursor_id")

			actual := NewFeedCursor(time.Date(2025, time.January, 2, 3, 4, 5, 600000000, time.UTC), id)

			assert.Equal(t, id, actual.ID())
		})
	})
}
