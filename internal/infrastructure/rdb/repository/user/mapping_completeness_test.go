package user

import (
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_fetchListUsersRows(t *testing.T) {
	t.Parallel()
	t.Skip("Test_repository_FindByActive（active=nil 経路）の実 DB テストでカバー")
}

func Test_fetchListUsersRowsByActive(t *testing.T) {
	t.Parallel()
	t.Skip("Test_repository_FindByActive（active=true 経路）の実 DB テストでカバー")
}

func Test_fetchListUsersRowsByDeleted(t *testing.T) {
	t.Parallel()
	t.Skip("Test_repository_FindByActive（active=false 経路）の実 DB テストでカバー")
}

func Test_rowToUser(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("再構築時の検証失敗はErrInternalへ正規化され元の分類は露出しない", func(t *testing.T) {
			t.Parallel()

			// ゼロ値の行は ID が nil のため domain 構築が失敗する。
			// 成功経路は Test_repository_FindByID / Test_repository_FindByActive の実 DB テストでカバー。
			entity, err := rowToUser(gen.Users{})
			require.Error(t, err)
			require.Nil(t, entity)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func Test_rowToRole(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な行からロールエンティティを再構築する", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			role, err := rowToRole(id, "admin", int16(1))
			require.NoError(t, err)
			require.NotNil(t, role)
			assert.Equal(t, id, role.ID())
			assert.Equal(t, "admin", role.Name())
			assert.True(t, role.IsAdmin())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("再構築時の検証失敗はErrInternalへ正規化され元の分類は露出しない", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			// code=0 は未知のロールコードのため domain 構築が失敗する。
			role, err := rowToRole(id, "admin", int16(0))
			require.Error(t, err)
			require.Nil(t, role)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}
