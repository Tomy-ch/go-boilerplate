package user

import (
	"context"
	"testing"
	"time"

	"boilerplate-go/internal/apperror"
	"boilerplate-go/internal/domain/user"
	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/infrastructure/rdb/sqlc/gen"
	"boilerplate-go/internal/infrastructure/rdb/testkit"
	"boilerplate-go/internal/observability"
	"boilerplate-go/pkg/ptr"
	"boilerplate-go/pkg/uuid"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	loggingDB := testkit.NewTestLoggingProvider(t)
	tf := observability.NewNoopTracerFactory(t)
	expected := &repository{
		tracer: tf.Infra(),
		db:     loggingDB,
	}
	actual := New(loggingDB, tf)
	require.Equal(t, expected, actual)
}

func TestFindAll(t *testing.T) {
	t.Parallel()

	loggingDB := testkit.NewTestLoggingProvider(t)
	db := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)

	txm := testkit.NewTestTransactionManager(t)

	repo := &repository{
		tracer: lt,
		db:     loggingDB,
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("limitとoffsetを指定した場合、作成順で複数件が取得できる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				limit := int32(100)
				offset := int32(0)

				firstUserID, err := uuid.Parse("eaabee3e-3b7a-4f61-8fa9-030944625e92")
				require.NoError(t, err)

				lastUserID, err := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
				require.NoError(t, err)

				actual, err := repo.FindAll(ctx, limit, offset)
				require.NoError(t, err)

				actualFirst := actual[0]
				actualLast := actual[len(actual)-1]

				require.Equal(t, firstUserID, actualFirst.ID())
				require.Equal(t, lastUserID, actualLast.ID())
			})
		})

		t.Run("limit=1でoffset=0の場合先頭のユーザーが取得できる", func(t *testing.T) {
			t.Parallel()

			firstUserID, err := uuid.Parse("eaabee3e-3b7a-4f61-8fa9-030944625e92")
			require.NoError(t, err)
			expectedLength := 1

			txm.WithinTx(func(ctx context.Context) {
				limit := int32(1)
				offset := int32(0)

				actual, err := repo.FindAll(ctx, limit, offset)
				require.NoError(t, err)
				require.Len(t, actual, expectedLength)

				require.Equal(t, firstUserID, actual[0].ID())
			})
		})

		t.Run("limit=1でoffset=9の場合、末尾のユーザーが取得できる", func(t *testing.T) {
			t.Parallel()

			lastUserID, err := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
			require.NoError(t, err)

			txm.WithinTx(func(ctx context.Context) {
				limit := int32(1)
				offset := int32(9)

				all, getAllUsersErr := repo.FindAll(ctx, limit, offset)
				require.NoError(t, getAllUsersErr)

				actual := all[len(all)-1]

				require.Equal(t, lastUserID, actual.ID())
			})
		})

		t.Run("limit=0の場合、空配列になる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				limit := int32(0)
				offset := int32(0)
				actual, err := repo.FindAll(ctx, limit, offset)
				require.NoError(t, err)
				require.Empty(t, actual)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("limitが負数の場合、エラーになる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindAll(ctx, -1, 0)
				require.Nil(t, actual)
				require.Error(t, err)
			})
		})

		t.Run("offsetが負数の場合、エラーになる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindAll(ctx, 10, -1)
				require.Nil(t, actual)
				require.Error(t, err)
			})
		})

		t.Run("無効なユーザーが挿入されていてもDomain化の時にエラーになる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, db)
				_, execErr := drv.Exec(ctx,
					"INSERT INTO users "+
						"(id, first_name, last_name, password_hash, email, phone, prefecture_id, city, street, postal_code) "+
						"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)",
					"07e5b6d3-0000-4000-8000-000000000000",
					"Tx",
					"",
					"$2a$10$dummy",
					"tx-insert@example.com",
					"000-000-0000",
					"a03aaec4-3bd6-4bfb-8e47-2fbfa026d344",
					"City",
					"Street",
					"000-0000",
				)
				require.NoError(t, execErr)

				res, actualErr := repo.FindAll(ctx, 100, 0)
				require.Nil(t, res)
				require.ErrorIs(t, actualErr, user.ErrInvalidLastName)
			})
		})
	})
}

func TestCreateUser(t *testing.T) {
	t.Parallel()

	loggingDB := testkit.NewTestLoggingProvider(t)
	db := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionManager(t)

	repo := &repository{
		tracer: lt,
		db:     loggingDB,
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効なユーザーエンティティの場合、ユーザーが作成できる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				now := time.Now()

				userID, err := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
				require.NoError(t, err)
				prefectureID, err := uuid.Parse("a03aaec4-3bd6-4bfb-8e47-2fbfa026d344")
				require.NoError(t, err)

				userEntity, err := user.New(
					userID,
					"Alice",
					"Smith",
					"password",
					"alice.smith@example.com",
					"555-555-5555",
					prefectureID,
					"新宿区",
					"5-5-5",
					ptr.To("Building X"),
					"160-0022",
					now,
					now,
					nil,
				)
				require.NoError(t, err)

				createErr := repo.Create(ctx, userEntity)
				require.NoError(t, createErr)

				user, err := gen.New(driver.New(ctx, db)).GetUserByID(ctx, userEntity.ID())
				require.NoError(t, err)
				require.Equal(t, userEntity.ID(), user.Users.ID)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("重複するメールアドレスの場合、エラーになる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				now := time.Now()
				userID, err := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
				require.NoError(t, err)
				prefectureID, err := uuid.Parse("a03aaec4-3bd6-4bfb-8e47-2fbfa026d344")
				require.NoError(t, err)

				userEntity, err := user.New(
					userID,
					"John",
					"Doe",
					"password",
					"john.doe@example.com",
					"555-555-5555",
					prefectureID,
					"新宿区",
					"5-5-5",
					ptr.To("Building X"),
					"160-0022",
					now,
					now,
					nil,
				)
				require.NoError(t, err)

				createErr := repo.Create(ctx, userEntity)
				require.ErrorIs(t, createErr, apperror.ErrConflict)
			})
		})
	})
}

func TestCountByActive(t *testing.T) {
	t.Parallel()

	loggingDB := testkit.NewTestLoggingProvider(t)
	lt := observability.NewMockInfraLayerTracer(t)

	txm := testkit.NewTestTransactionManager(t)

	repo := &repository{
		tracer: lt,
		db:     loggingDB,
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("active=trueの場合、アクティブなユーザーの件数が取得できる", func(t *testing.T) {
			t.Parallel()
			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.CountByActive(ctx, ptr.To(true))
				require.NoError(t, err)
				require.Equal(t, int64(8), got)
			})
		})
		t.Run("active=falseの場合、非アクティブなユーザーの件数が取得できる", func(t *testing.T) {
			t.Parallel()
			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.CountByActive(ctx, ptr.To(false))
				require.NoError(t, err)
				require.Equal(t, int64(2), got)
			})
		})
		t.Run("active=nilの場合、全ユーザーの件数が取得できる", func(t *testing.T) {
			t.Parallel()
			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.CountByActive(ctx, nil)
				require.NoError(t, err)
				require.Equal(t, int64(10), got)
			})
		})
	})
}
