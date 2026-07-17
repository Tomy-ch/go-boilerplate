package user

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRoleRepository(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	tf := observability.NewNoopTracerFactory(t)
	expected := &roleRepository{
		tracer: tf.Infra(),
		db:     testDB,
	}
	actual := NewRoleRepository(testDB, tf)
	assert.Equal(t, expected, actual)
}

func Test_roleRepository_FindRolesByUserID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)

	txm := testkit.NewTestTransactionRunner(t)

	repo := &roleRepository{
		tracer: lt,
		db:     testDB,
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("管理者ロールを持つユーザーの場合、管理者を含むロール一覧が取得できる", func(t *testing.T) {
			t.Parallel()

			adminUserID, err := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
			require.NoError(t, err)

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindRolesByUserID(ctx, adminUserID)
				require.NoError(t, err)
				assert.True(t, actual.HasAdmin())
			})
		})

		t.Run("一般ロールのみのユーザーの場合、管理者を含まないロール一覧が取得できる", func(t *testing.T) {
			t.Parallel()

			generalUserID, err := uuid.Parse("a95a2dd3-2b37-4def-8041-23d2138faccc")
			require.NoError(t, err)

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindRolesByUserID(ctx, generalUserID)
				require.NoError(t, err)
				require.Len(t, actual, 1)
				assert.False(t, actual.HasAdmin())
				assert.Equal(t, user.RoleCodeGeneral, actual[0].Code())
			})
		})

		t.Run("ロール未割当のユーザーの場合、空のロール一覧を返す", func(t *testing.T) {
			t.Parallel()

			notAssignedID, err := uuid.Parse("00000000-0000-0000-0000-000000000000")
			require.NoError(t, err)

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindRolesByUserID(ctx, notAssignedID)
				require.NoError(t, err)
				assert.Empty(t, actual)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化される", func(t *testing.T) {
			t.Parallel()

			adminUserID, err := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
			require.NoError(t, err)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			actual, err := repo.FindRolesByUserID(ctx, adminUserID)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}
