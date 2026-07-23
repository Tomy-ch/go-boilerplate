package user

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_repository_FindByID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{tracer: lt, db: testDB}

	seededID, err := uuid.Parse("eaabee3e-3b7a-4f61-8fa9-030944625e92")
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("存在するIDでユーザーが取得できる", func(t *testing.T) {
			t.Parallel()
			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.FindByID(ctx, seededID)
				require.NoError(t, err)
				assert.Equal(t, seededID, got.ID())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()
		t.Run("存在しないIDの場合_NotFoundが返る", func(t *testing.T) {
			t.Parallel()
			txm.WithinTx(func(ctx context.Context) {
				missing := uuid.NewTestFromSalt(t, "missing-user")
				_, err := repo.FindByID(ctx, missing)
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})
	})
}

//nolint:paralleltest,tparallel // 同一行のロック競合回避のためサブテストを直列実行する
func Test_repository_Update(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	_ = testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{tracer: lt, db: testDB}

	firstID, err := uuid.Parse("eaabee3e-3b7a-4f61-8fa9-030944625e92")
	require.NoError(t, err)
	lastID, err := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)

	// 同一行（firstID）を更新するため、ロック競合回避にサブテストは直列実行する（t.Parallel を付けない）。
	t.Run("正常系", func(t *testing.T) {
		t.Run("プロフィール更新が永続化される", func(t *testing.T) {
			txm.WithinTx(func(ctx context.Context) {
				u, err := repo.FindByID(ctx, firstID)
				require.NoError(t, err)

				err = u.UpdateProfile("UpdatedFirst", u.LastName(), u.Email(), u.Phone(),
					u.PrefectureID(), u.City(), u.Street(), u.Building(), u.PostalCode(), time.Now())
				require.NoError(t, err)
				require.NoError(t, repo.Update(ctx, u))

				got, err := repo.FindByID(ctx, firstID)
				require.NoError(t, err)
				assert.Equal(t, "UpdatedFirst", got.FirstName())
			})
		})

		t.Run("論理削除すると以降FindByIDの対象外になる", func(t *testing.T) {
			txm.WithinTx(func(ctx context.Context) {
				u, err := repo.FindByID(ctx, firstID)
				require.NoError(t, err)
				require.NoError(t, u.MarkAsDeleted(time.Now()))
				require.NoError(t, repo.Update(ctx, u))

				// FindByID / Update は deleted_at IS NULL でフィルタするため、削除後は NotFound
				_, err = repo.FindByID(ctx, firstID)
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("メールアドレス重複でErrConflictが返る", func(t *testing.T) {
			txm.WithinTx(func(ctx context.Context) {
				target, err := repo.FindByID(ctx, firstID)
				require.NoError(t, err)
				other, err := repo.FindByID(ctx, lastID)
				require.NoError(t, err)

				// 別ユーザーのメールアドレスへ更新 → unique 制約違反
				err = target.UpdateProfile(target.FirstName(), target.LastName(), other.Email(), target.Phone(),
					target.PrefectureID(), target.City(), target.Street(), target.Building(), target.PostalCode(), time.Now())
				require.NoError(t, err)

				err = repo.Update(ctx, target)
				require.ErrorIs(t, err, apperror.ErrConflict)
			})
		})

		t.Run("存在しないIDのUpdateはErrNotFoundを返す", func(t *testing.T) {
			txm.WithinTx(func(ctx context.Context) {
				prefID, err := uuid.Parse("a03aaec4-3bd6-4bfb-8e47-2fbfa026d344") // シード済み都道府県
				require.NoError(t, err)
				now := time.Now()

				// DB に存在しない ID のエンティティ
				u, err := user.New(
					uuid.NewTestFromSalt(t, "missing-update"),
					"X", "Y", "missing-update@example.com", "09000000000",
					prefID, "City", "Street", nil, "100-0001", now, now, nil,
				)
				require.NoError(t, err)

				err = repo.Update(ctx, u)
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})
	})
}
