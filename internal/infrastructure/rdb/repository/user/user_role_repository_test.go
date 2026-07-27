package user

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
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

		t.Run("管理者ロールを持つユーザーの場合、管理者と一般を併せ持つロール一覧が取得できる", func(t *testing.T) {
			t.Parallel()

			adminUserID, err := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
			require.NoError(t, err)

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindRolesByUserID(ctx, adminUserID)
				require.NoError(t, err)
				require.Len(t, actual, 2)
				assert.True(t, actual.HasAdmin())
				// ORDER BY r.code のため管理者(1)→一般(2)の順で返る。
				assert.Equal(t, user.RoleCodeAdmin, actual[0].Code())
				assert.Equal(t, user.RoleCodeGeneral, actual[1].Code())
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

		t.Run("ドメイン不変条件に反する行が存在する場合、ErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()

			generalUserID, err := uuid.Parse("a95a2dd3-2b37-4def-8041-23d2138faccc")
			require.NoError(t, err)

			txm.WithinTx(func(ctx context.Context) {
				// 既知ロール外の code を持つ行を割り当て、行→エンティティ再構築の失敗経路を誘発する。
				insertInvalidRole(ctx, t, driver.New(ctx, testDB), generalUserID, uuid.NewTestFromSalt(t, "invalid_role"))

				actual, err := repo.FindRolesByUserID(ctx, generalUserID)
				require.Nil(t, actual)
				require.ErrorIs(t, err, apperror.ErrInternal)
			})
		})
	})
}

// insertInvalidRole は、既知ロール外の code(99) を持つロールを対象ユーザーへ割り当てるヘルパーです。
// FindRolesByUserID が行→エンティティ変換で再構築エラーになる経路を検証するために使用します。
func insertInvalidRole(ctx context.Context, t *testing.T, db driver.DBTX, userID, roleID uuid.UUID) {
	t.Helper()

	_, err := db.Exec(ctx,
		"INSERT INTO roles (id, name, code) VALUES ($1, $2, $3)",
		roleID, "invalid-role", int16(99), // code=99 は RoleCode.valid() を満たさずドメイン不変条件違反となる。
	)
	require.NoError(t, err)

	_, err = db.Exec(ctx,
		"INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)",
		userID, roleID,
	)
	require.NoError(t, err)
}

func Test_rowToRole(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な行からロールエンティティを再構築する", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			role, err := rowToRole(&gen.GetUserRolesByUserIDRow{ID: id, Name: "admin", Code: int16(1)})
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
			role, err := rowToRole(&gen.GetUserRolesByUserIDRow{ID: id, Name: "admin", Code: int16(0)})
			require.Error(t, err)
			require.Nil(t, role)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}
