package user

import (
	"testing"
	"time"

	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/require"
)

// newValidUser は、削除されていない有効なユーザーと基準時刻を返すテストヘルパー。
func newValidUser(t *testing.T) (*User, time.Time) {
	t.Helper()
	base := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	u, err := New(
		uuid.NewTestFromSalt(t, "user"),
		"John", "Doe", "hashed_password", "john@example.com", "1234567890",
		uuid.NewTestFromSalt(t, "prefecture"),
		"Shibuya", "1-2-3", ptr.To("Building A"), "150-0001",
		base, base, nil,
	)
	require.NoError(t, err)
	return u, base
}

func TestUser_UpdateProfile(t *testing.T) {
	t.Parallel()

	newPrefID := uuid.NewTestFromSalt(t, "prefecture2")

	t.Run("正常系_全フィールドと更新日時が置き換わる", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)
		newUpdatedAt := base.Add(time.Hour)

		err := u.UpdateProfile("Jane", "Smith", "jane@example.com", "0987654321",
			newPrefID, "200-0002", "Minato", "4-5-6", ptr.To("Tower"), newUpdatedAt)
		require.NoError(t, err)

		require.Equal(t, "Jane", u.firstName)
		require.Equal(t, "Smith", u.lastName)
		require.Equal(t, "jane@example.com", u.email)
		require.Equal(t, newPrefID, u.prefectureID)
		require.Equal(t, "Minato", u.city)
		require.Equal(t, newUpdatedAt, u.updatedAt)
	})

	t.Run("異常系_プロフィールフィールドが不正な場合_エラーを返す", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)

		err := u.UpdateProfile("", "Smith", "jane@example.com", "0987654321",
			newPrefID, "200-0002", "Minato", "4-5-6", nil, base.Add(time.Hour))
		require.ErrorIs(t, err, ErrInvalidFirstName)
	})

	t.Run("異常系_updatedAtがcreatedAtより前の場合_エラーを返す", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)

		err := u.UpdateProfile("Jane", "Smith", "jane@example.com", "0987654321",
			newPrefID, "200-0002", "Minato", "4-5-6", nil, base.Add(-time.Hour))
		require.ErrorIs(t, err, ErrInvalidUpdatedAt)
	})

	t.Run("異常系_削除済みユーザーでupdatedAtがdeletedAtより後の場合_エラーを返す", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)
		require.NoError(t, u.MarkAsDeleted(base.Add(time.Hour)))

		err := u.UpdateProfile("Jane", "Smith", "jane@example.com", "0987654321",
			newPrefID, "200-0002", "Minato", "4-5-6", nil, base.Add(2*time.Hour))
		require.ErrorIs(t, err, ErrInvalidDeletedAt)
	})
}

func TestUser_ChangePassword(t *testing.T) {
	t.Parallel()

	t.Run("正常系_パスワードハッシュと更新日時が置き換わる", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)
		newUpdatedAt := base.Add(time.Hour)

		err := u.ChangePassword("new_hashed_password", newUpdatedAt)
		require.NoError(t, err)
		require.Equal(t, "new_hashed_password", u.passwordHash)
		require.Equal(t, newUpdatedAt, u.updatedAt)
	})

	t.Run("異常系_パスワードハッシュが不正な場合_エラーを返す", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)

		err := u.ChangePassword("", base.Add(time.Hour))
		require.ErrorIs(t, err, ErrInvalidPasswordHash)
	})

	t.Run("異常系_updatedAtがcreatedAtより前の場合_エラーを返す", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)

		err := u.ChangePassword("new_hashed_password", base.Add(-time.Hour))
		require.ErrorIs(t, err, ErrInvalidUpdatedAt)
	})
}

func TestUser_MarkAsDeleted(t *testing.T) {
	t.Parallel()

	t.Run("正常系_deletedAtが設定される", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)
		deletedAt := base.Add(time.Hour)

		err := u.MarkAsDeleted(deletedAt)
		require.NoError(t, err)
		require.NotNil(t, u.deletedAt)
		require.Equal(t, deletedAt, *u.deletedAt)
	})

	t.Run("異常系_既に削除済みの場合_ErrAlreadyDeletedを返す", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)
		require.NoError(t, u.MarkAsDeleted(base.Add(time.Hour)))

		err := u.MarkAsDeleted(base.Add(2 * time.Hour))
		require.ErrorIs(t, err, ErrAlreadyDeleted)
	})

	t.Run("異常系_deletedAtがcreatedAtより前の場合_エラーを返す", func(t *testing.T) {
		t.Parallel()
		u, base := newValidUser(t)

		err := u.MarkAsDeleted(base.Add(-time.Hour))
		require.ErrorIs(t, err, ErrInvalidDeletedAt)
	})

	t.Run("異常系_deletedAtがupdatedAtより前の場合_エラーを返す", func(t *testing.T) {
		t.Parallel()
		base := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
		u, err := New(
			uuid.NewTestFromSalt(t, "user"),
			"John", "Doe", "hashed_password", "john@example.com", "1234567890",
			uuid.NewTestFromSalt(t, "prefecture"),
			"Shibuya", "1-2-3", ptr.To("Building A"), "150-0001",
			base, base.Add(2*time.Hour), nil,
		)
		require.NoError(t, err)

		err = u.MarkAsDeleted(base.Add(time.Hour))
		require.ErrorIs(t, err, ErrInvalidDeletedAt)
	})
}
