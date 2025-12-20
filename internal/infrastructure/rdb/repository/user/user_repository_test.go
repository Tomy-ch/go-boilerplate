package user

import (
	"context"
	"testing"
	"time"

	"boilerplate-go/internal/apperror"
	"boilerplate-go/internal/domain/user"
	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/infrastructure/rdb/rdbtest"
	"boilerplate-go/internal/infrastructure/rdb/sqlc/gen"
	"boilerplate-go/internal/observability"
	"boilerplate-go/pkg/ptr"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	db, provider := rdbtest.NewTestDBWithLoggingProvider(t)
	tf := observability.NewNoopTracerFactory(t)
	expected := &repository{
		db:       db,
		tracer:   tf.Infra(),
		provider: provider,
	}
	actual := New(db, provider, tf)
	require.Equal(t, expected, actual)
}

func TestFindAll(t *testing.T) {
	// t.Parallel()　// NOTE: 並列実行不可
	// 保存処理などが影響しあい、テストが不安定になるため並列実行不可とする。

	db, provider := rdbtest.NewTestDBWithLoggingProvider(t)
	lt := observability.NewMockInfraLayerTracer(t)

	txm := rdbtest.NewTestTransactionManager(t)

	repo := &repository{
		tracer:   lt,
		db:       db,
		provider: provider,
	}

	t.Run("正常系", func(t *testing.T) {
		// t.Parallel()

		t.Run("limitとoffsetを指定した場合、作成順で複数件が取得できる", func(t *testing.T) {
			// t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				limit := 100
				offset := 0

				expectedFirst, err := user.New(
					"eaabee3e-3b7a-4f61-8fa9-030944625e92",
					"Ivy",
					"Clark",
					"$2a$08$M1GUQfiCBgfhEWEirBko9.urJ3zFgU2McymmuDj3890PwPxSJRLdu6",
					"ivy.clark@example.com",
					"888-888-8888",
					"a03aaec4-3bd6-4bfb-8e47-2fbfa026d344",
					"鹿児島市",
					"7-7-7",
					ptr.To("Building G"),
					"890-0001",
					nil,
				)
				require.NoError(t, err)

				expectedLast, err := user.New(
					"550e8400-e29b-41d4-a716-446655440000",
					"John",
					"Doe",
					"$2a$08$e3DJxb7ZOfRkP2sDSmopw.Djw.PP.1GeY/ATp0Bbu6P7zksaWiEH26",
					"john.doe@example.com",
					"123-456-7890",
					"faba7bb2-f5a0-4a51-adae-1564929077b2",
					"札幌",
					"1-1",
					ptr.To("Building A"),
					"060-0001",
					nil,
				)
				require.NoError(t, err)

				actual, err := repo.FindAll(ctx, limit, offset)
				require.NoError(t, err)

				actualFirst := actual[0]
				actualLast := actual[len(actual)-1]

				require.Equal(t, expectedFirst, actualFirst)
				require.Equal(t, expectedLast, actualLast)
			})
		})

		t.Run("limit=1でoffset=0の場合先頭のユーザーが取得できる", func(t *testing.T) {
			// t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				limit := 1
				offset := 0

				expected, err := user.New(
					"eaabee3e-3b7a-4f61-8fa9-030944625e92",
					"Ivy", "Clark",
					"$2a$08$M1GUQfiCBgfhEWEirBko9.urJ3zFgU2McymmuDj3890PwPxSJRLdu6",
					"ivy.clark@example.com",
					"888-888-8888",
					"a03aaec4-3bd6-4bfb-8e47-2fbfa026d344",
					"鹿児島市",
					"7-7-7",
					ptr.To("Building G"),
					"890-0001",
					nil,
				)
				require.NoError(t, err)
				expectedLength := 1

				actual, err := repo.FindAll(ctx, limit, offset)
				require.NoError(t, err)
				require.Len(t, actual, expectedLength)

				require.Equal(t, expected, actual[0])
			})
		})

		t.Run("limit=1でoffset=9の場合、末尾のユーザーが取得できる", func(t *testing.T) {
			// t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				limit := 1
				offset := 9
				expected, getAllUsersErr := user.New(
					"550e8400-e29b-41d4-a716-446655440000",
					"John",
					"Doe",
					"$2a$08$e3DJxb7ZOfRkP2sDSmopw.Djw.PP.1GeY/ATp0Bbu6P7zksaWiEH26",
					"john.doe@example.com",
					"123-456-7890",
					"faba7bb2-f5a0-4a51-adae-1564929077b2",
					"札幌",
					"1-1",
					ptr.To("Building A"),
					"060-0001",
					nil,
				)
				require.NoError(t, getAllUsersErr)

				all, getAllUsersErr := repo.FindAll(ctx, limit, offset)
				require.NoError(t, getAllUsersErr)

				actual := all[len(all)-1]

				require.Equal(t, expected, actual)
			})
		})

		t.Run("limit=0の場合、空配列になる", func(t *testing.T) {
			// t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				limit := 0
				offset := 0
				actual, err := repo.FindAll(ctx, limit, offset)
				require.NoError(t, err)
				require.Empty(t, actual)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		// t.Parallel()

		t.Run("limitが負数の場合、エラーになる", func(t *testing.T) {
			// t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindAll(ctx, -1, 0)
				require.Nil(t, actual)
				require.Error(t, err)
			})
		})

		t.Run("offsetが負数の場合、エラーになる", func(t *testing.T) {
			// t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindAll(ctx, 10, -1)
				require.Nil(t, actual)
				require.Error(t, err)
			})
		})

		t.Run("無効なユーザーが挿入されていてもDomain化の時にエラーになる", func(t *testing.T) {
			// t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, db)
				_, execErr := drv.ExecContext(ctx,
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

func TestFindByKeyword(t *testing.T) {
	// t.Parallel()　// NOTE: 並列実行不可
	// 保存処理などが影響しあい、テストが不安定になるため並列実行不可とする。

	db, provider := rdbtest.NewTestDBWithLoggingProvider(t)
	lt := observability.NewMockInfraLayerTracer(t)

	txm := rdbtest.NewTestTransactionManager(t)

	repo := &repository{
		tracer:   lt,
		db:       db,
		provider: provider,
	}

	t.Run("正常系", func(t *testing.T) {
		// t.Parallel()

		t.Run("キーワードにマッチするユーザーが取得できる", func(t *testing.T) {
			// t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				keywords := []string{"Grace"}
				limit := 10
				offset := 0

				expected, err := user.New(
					"c688ffbc-731e-4257-82e9-d34b4712afd6",
					"Grace",
					"Lee",
					"$2a$08$TuXnmKZjCfyXhTw2Zh81POI1ZlaDTZzgCtf2SbC1MN64WSx0Nm6zi6",
					"grace.lee@example.com",
					"000-000-0000",
					"d647fc85-ff46-4530-88cb-198f4a68a9d7",
					"大阪市",
					"5-5-5",
					ptr.To("Building F"),
					"530-0001",
					nil,
				)
				require.NoError(t, err)
				expectedLength := 1

				actual, err := repo.FindByKeyword(ctx, keywords, nil, limit, offset)
				require.NoError(t, err)
				require.Len(t, actual, expectedLength)

				require.Equal(t, expected, actual[0])
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		// t.Parallel()

		t.Run("limitが負数の場合、エラーになる", func(t *testing.T) {
			// t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindByKeyword(ctx, nil, nil, -1, 0)
				require.Nil(t, actual)
				require.Error(t, err)
			})
		})

		t.Run("offsetが負数の場合、エラーになる", func(t *testing.T) {
			// t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindByKeyword(ctx, nil, nil, 10, -1)
				require.Nil(t, actual)
				require.Error(t, err)
			})
		})

		t.Run("無効なユーザーが挿入されていてもDomain化の時にエラーになる", func(t *testing.T) {
			// t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, db)
				_, execErr := drv.ExecContext(ctx,
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

				res, actualErr := repo.FindByKeyword(ctx, nil, nil, 100, 0)
				require.Nil(t, res)
				require.ErrorIs(t, actualErr, user.ErrInvalidLastName)
			})
		})
	})
}

func TestCreateUser(t *testing.T) {
	// t.Parallel()　// NOTE: 並列実行不可
	// 保存処理などが影響しあい、テストが不安定になるため並列実行不可とする。

	db, provider := rdbtest.NewTestDBWithLoggingProvider(t)
	lt := observability.NewMockInfraLayerTracer(t)

	txm := rdbtest.NewTestTransactionManager(t)

	repo := &repository{
		tracer:   lt,
		db:       db,
		provider: provider,
	}

	t.Run("正常系", func(t *testing.T) {
		// t.Parallel()

		t.Run("有効なユーザーエンティティの場合、ユーザーが作成できる", func(t *testing.T) {
			// t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				now := time.Now()
				userEntity, err := user.New(
					"123e4567-e89b-12d3-a456-426614174000",
					"Alice",
					"Smith",
					"password",
					"alice.smith@example.com",
					"555-555-5555",
					"a03aaec4-3bd6-4bfb-8e47-2fbfa026d344",
					"新宿区",
					"5-5-5",
					ptr.To("Building X"),
					"160-0022",
					nil,
				)
				require.NoError(t, err)

				createErr := repo.CreateUser(ctx, now, userEntity)
				require.NoError(t, createErr)

				user, err := gen.New(driver.New(ctx, db)).GetUserByID(ctx, userEntity.ID().ToPrimitive())
				require.NoError(t, err)
				require.Equal(t, userEntity.ID().String(), user.Users.ID.String())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		// t.Parallel()

		t.Run("重複するメールアドレスの場合、エラーになる", func(t *testing.T) {
			// t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				now := time.Now()
				userEntity, err := user.New(
					"123e4567-e89b-12d3-a456-426614174000",
					"John",
					"Doe",
					"password",
					"john.doe@example.com",
					"555-555-5555",
					"a03aaec4-3bd6-4bfb-8e47-2fbfa026d344",
					"新宿区",
					"5-5-5",
					ptr.To("Building X"),
					"160-0022",
					nil,
				)
				require.NoError(t, err)

				createErr := repo.CreateUser(ctx, now, userEntity)
				require.ErrorIs(t, createErr, apperror.ErrConflict)
			})
		})
	})
}
