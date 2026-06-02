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

	"github.com/stretchr/testify/require"
)

func Test_repository_FindByID(t *testing.T) {
	t.Parallel()

	loggingDB := testkit.NewTestLoggingProvider(t)
	_ = testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionManager(t)

	repo := &repository{tracer: lt, db: loggingDB}

	seededID, err := uuid.Parse("eaabee3e-3b7a-4f61-8fa9-030944625e92")
	require.NoError(t, err)

	t.Run("正常系_存在するIDでユーザーが取得できる", func(t *testing.T) {
		t.Parallel()
		txm.WithinTx(func(ctx context.Context) {
			got, err := repo.FindByID(ctx, seededID)
			require.NoError(t, err)
			require.Equal(t, seededID, got.ID())
		})
	})

	t.Run("異常系_存在しないIDの場合_NotFoundが返る", func(t *testing.T) {
		t.Parallel()
		txm.WithinTx(func(ctx context.Context) {
			missing := uuid.NewTestFromSalt(t, "missing-user")
			_, err := repo.FindByID(ctx, missing)
			require.ErrorIs(t, err, apperror.ErrNotFound)
		})
	})
}

func Test_repository_Update(t *testing.T) {
	t.Parallel()

	loggingDB := testkit.NewTestLoggingProvider(t)
	_ = testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionManager(t)

	repo := &repository{tracer: lt, db: loggingDB}

	firstID, err := uuid.Parse("eaabee3e-3b7a-4f61-8fa9-030944625e92")
	require.NoError(t, err)
	lastID, err := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)

	// 同一行（firstID）を更新するため、ロック競合回避にサブテストは直列実行する。
	t.Run("正常系_プロフィール更新が永続化される", func(t *testing.T) {
		txm.WithinTx(func(ctx context.Context) {
			u, err := repo.FindByID(ctx, firstID)
			require.NoError(t, err)

			err = u.UpdateProfile("UpdatedFirst", u.LastName(), u.Email(), u.Phone(),
				u.PrefectureID(), u.PostalCode(), u.City(), u.Street(), u.Building(), time.Now())
			require.NoError(t, err)
			require.NoError(t, repo.Update(ctx, u))

			got, err := repo.FindByID(ctx, firstID)
			require.NoError(t, err)
			require.Equal(t, "UpdatedFirst", got.FirstName())
		})
	})

	t.Run("正常系_論理削除（deletedAt設定）が永続化される", func(t *testing.T) {
		txm.WithinTx(func(ctx context.Context) {
			u, err := repo.FindByID(ctx, firstID)
			require.NoError(t, err)
			require.NoError(t, u.MarkAsDeleted(time.Now()))
			require.NoError(t, repo.Update(ctx, u))

			got, err := repo.FindByID(ctx, firstID)
			require.NoError(t, err)
			require.NotNil(t, got.DeletedAt())
		})
	})

	t.Run("異常系_メールアドレス重複でErrConflictが返る", func(t *testing.T) {
		txm.WithinTx(func(ctx context.Context) {
			target, err := repo.FindByID(ctx, firstID)
			require.NoError(t, err)
			other, err := repo.FindByID(ctx, lastID)
			require.NoError(t, err)

			// 別ユーザーのメールアドレスへ更新 → unique 制約違反
			err = target.UpdateProfile(target.FirstName(), target.LastName(), other.Email(), target.Phone(),
				target.PrefectureID(), target.PostalCode(), target.City(), target.Street(), target.Building(), time.Now())
			require.NoError(t, err)

			err = repo.Update(ctx, target)
			require.ErrorIs(t, err, apperror.ErrConflict)
		})
	})

	t.Run("異常系_存在しないIDのUpdateはErrNotFoundを返す", func(t *testing.T) {
		txm.WithinTx(func(ctx context.Context) {
			prefID, err := uuid.Parse("a03aaec4-3bd6-4bfb-8e47-2fbfa026d344") // シード済み都道府県
			require.NoError(t, err)
			now := time.Now()

			// DB に存在しない ID のエンティティ
			u, err := user.New(
				uuid.NewTestFromSalt(t, "missing-update"),
				"X", "Y", "hashed_password", "missing-update@example.com", "09000000000",
				prefID, "City", "Street", nil, "100-0001", now, now, nil,
			)
			require.NoError(t, err)

			err = repo.Update(ctx, u)
			require.ErrorIs(t, err, apperror.ErrNotFound)
		})
	})
}
