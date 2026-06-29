package user

import (
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeFeedCursor(t *testing.T) {
	t.Parallel()

	validID := uuid.NewTestFromSalt(t, "feed_cursor_id")
	validTime := time.Date(2025, time.January, 2, 3, 4, 5, 600000000, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な2キーのカーソルはFeedCursorへ復号される", func(t *testing.T) {
			t.Parallel()
			encoded := paging.EncodeCursor(validTime.Format(time.RFC3339Nano), validID.String())
			cursor, err := paging.NewCursor(&encoded, nil)
			require.NoError(t, err)

			actual, decErr := decodeFeedCursor(cursor)
			require.NoError(t, decErr)
			require.NotNil(t, actual)
			assert.True(t, actual.CreatedAt().Equal(validTime))
			assert.Equal(t, validID, actual.ID())
		})

		t.Run("先頭ページ(カーソル無し)の場合はnilを返す", func(t *testing.T) {
			t.Parallel()
			cursor, err := paging.NewCursor(nil, nil)
			require.NoError(t, err)

			actual, decErr := decodeFeedCursor(cursor)
			require.NoError(t, decErr)
			assert.Nil(t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キー数が2でない場合はErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()
			encoded := paging.EncodeCursor("only-one-key")
			cursor, err := paging.NewCursor(&encoded, nil)
			require.NoError(t, err)

			actual, decErr := decodeFeedCursor(cursor)
			require.ErrorIs(t, decErr, apperror.ErrInvalidArgument)
			require.Nil(t, actual)
		})

		t.Run("created_atがRFC3339Nanoでない場合はErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()
			encoded := paging.EncodeCursor("not-a-time", validID.String())
			cursor, err := paging.NewCursor(&encoded, nil)
			require.NoError(t, err)

			actual, decErr := decodeFeedCursor(cursor)
			require.ErrorIs(t, decErr, apperror.ErrInvalidArgument)
			require.Nil(t, actual)
		})

		t.Run("idがUUIDでない場合はErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()
			encoded := paging.EncodeCursor(validTime.Format(time.RFC3339Nano), "not-a-uuid")
			cursor, err := paging.NewCursor(&encoded, nil)
			require.NoError(t, err)

			actual, decErr := decodeFeedCursor(cursor)
			require.ErrorIs(t, decErr, apperror.ErrInvalidArgument)
			require.Nil(t, actual)
		})
	})
}

func TestEncodeFeedCursor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("末尾行のソートキーから生成したカーソルは同じcreated_at/idへ復号できる", func(t *testing.T) {
			t.Parallel()

			id := uuid.NewTestFromSalt(t, "encode_feed_cursor_id")
			createdAt := time.Date(2025, time.March, 4, 5, 6, 7, 800000000, time.UTC)
			last, err := user.New(
				id,
				"first_name", "last_name", "password", "email_address", "phone_number",
				uuid.NewTestFromSalt(t, "encode_feed_cursor_pref"), "city_name", "town_address", nil, "p_code", createdAt, createdAt, nil,
			)
			require.NoError(t, err)

			encoded := encodeFeedCursor(last)
			require.NotEmpty(t, encoded)

			cursor, cErr := paging.NewCursor(&encoded, nil)
			require.NoError(t, cErr)

			decoded, decErr := decodeFeedCursor(cursor)
			require.NoError(t, decErr)
			require.NotNil(t, decoded)
			assert.True(t, decoded.CreatedAt().Equal(createdAt))
			assert.Equal(t, id, decoded.ID())
		})
	})
}
