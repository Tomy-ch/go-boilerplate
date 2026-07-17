package user

import (
	"testing"

	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRole(t *testing.T) {
	t.Parallel()

	validID := uuid.NewTestFromSalt(t, "role")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("管理者コードの場合、Role が構築され IsAdmin が true を返す", func(t *testing.T) {
			t.Parallel()

			actual, err := NewRole(validID, "管理者", int(RoleCodeAdmin))
			require.NoError(t, err)
			assert.Equal(t, validID, actual.ID())
			assert.Equal(t, "管理者", actual.Name())
			assert.Equal(t, RoleCodeAdmin, actual.Code())
			assert.True(t, actual.IsAdmin())
		})

		t.Run("一般コードの場合、Role が構築され IsAdmin が false を返す", func(t *testing.T) {
			t.Parallel()

			actual, err := NewRole(validID, "一般", int(RoleCodeGeneral))
			require.NoError(t, err)
			assert.Equal(t, RoleCodeGeneral, actual.Code())
			assert.False(t, actual.IsAdmin())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("id が nil の場合、ErrInvalidRoleID を返す", func(t *testing.T) {
			t.Parallel()

			_, err := NewRole(uuid.UUID{}, "管理者", int(RoleCodeAdmin))
			require.ErrorIs(t, err, ErrInvalidRoleID)
		})

		t.Run("名前が空文字の場合、ErrInvalidRoleName を返す", func(t *testing.T) {
			t.Parallel()

			_, err := NewRole(validID, "", int(RoleCodeAdmin))
			require.ErrorIs(t, err, ErrInvalidRoleName)
		})

		t.Run("未知のコードの場合、ErrInvalidRoleCode を返す", func(t *testing.T) {
			t.Parallel()

			_, err := NewRole(validID, "不明", 99)
			require.ErrorIs(t, err, ErrInvalidRoleCode)
		})
	})
}

func TestRoles_HasAdmin(t *testing.T) {
	t.Parallel()

	validID := uuid.NewTestFromSalt(t, "role")

	adminRole, err := NewRole(validID, "管理者", int(RoleCodeAdmin))
	require.NoError(t, err)
	generalRole, err := NewRole(validID, "一般", int(RoleCodeGeneral))
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("管理者ロールを含む場合、true を返す", func(t *testing.T) {
			t.Parallel()

			assert.True(t, Roles{generalRole, adminRole}.HasAdmin())
		})

		t.Run("管理者ロールを含まない場合、false を返す", func(t *testing.T) {
			t.Parallel()

			assert.False(t, Roles{generalRole}.HasAdmin())
		})

		t.Run("空スライスの場合、false を返す", func(t *testing.T) {
			t.Parallel()

			assert.False(t, Roles{}.HasAdmin())
		})
	})
}
