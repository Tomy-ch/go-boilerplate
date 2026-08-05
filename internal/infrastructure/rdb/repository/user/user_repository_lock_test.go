package user

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seededActiveUserID は、シードの未削除ユーザー（Ivy Clark）の ID です。
const seededActiveUserID = "eaabee3e-3b7a-4f61-8fa9-030944625e92"

// seededDeletedUserID は、シードの論理削除済みユーザー（Charlie Davis）の ID です。
const seededDeletedUserID = "d711970c-8e86-4875-8a34-e90bd79096a5"

func TestNewLockRepository(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	tf := observability.NewNoopTracerFactory(t)
	expected := &lockRepository{
		tracer: tf.Infra(),
		db:     testDB,
	}
	actual := NewLockRepository(testDB, tf)
	assert.Equal(t, expected, actual)
}

func Test_lockRepository_LockByID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)

	repo := &lockRepository{tracer: lt, db: testDB}

	activeID, err := uuid.Parse(seededActiveUserID)
	require.NoError(t, err)
	deletedID, err := uuid.Parse(seededDeletedUserID)
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未削除のユーザーをロックして取得できる", func(t *testing.T) {
			t.Parallel()
			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.LockByID(ctx, activeID)
				require.NoError(t, err)
				assert.Equal(t, activeID, got.ID())
				assert.Nil(t, got.DeletedAt())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("論理削除済みのユーザーの場合_NotFound", func(t *testing.T) {
			t.Parallel()
			txm.WithinTx(func(ctx context.Context) {
				_, err := repo.LockByID(ctx, deletedID)
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})

		t.Run("存在しないIDの場合_NotFound", func(t *testing.T) {
			t.Parallel()
			txm.WithinTx(func(ctx context.Context) {
				_, err := repo.LockByID(ctx, uuidtestkit.NewTestFromSalt(t, "lock-missing-user"))
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})
	})
}

func Test_lockRepository_LockActiveShareByID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)

	repo := &lockRepository{tracer: lt, db: testDB}

	activeID, err := uuid.Parse(seededActiveUserID)
	require.NoError(t, err)
	deletedID, err := uuid.Parse(seededDeletedUserID)
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未削除のユーザーの在籍を確認できる", func(t *testing.T) {
			t.Parallel()
			txm.WithinTx(func(ctx context.Context) {
				require.NoError(t, repo.LockActiveShareByID(ctx, activeID))
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("論理削除済みのユーザーの場合_NotFound", func(t *testing.T) {
			t.Parallel()
			txm.WithinTx(func(ctx context.Context) {
				require.ErrorIs(t, repo.LockActiveShareByID(ctx, deletedID), apperror.ErrNotFound)
			})
		})

		t.Run("存在しないIDの場合_NotFound", func(t *testing.T) {
			t.Parallel()
			txm.WithinTx(func(ctx context.Context) {
				err := repo.LockActiveShareByID(ctx, uuidtestkit.NewTestFromSalt(t, "share-missing-user"))
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})
	})
}
