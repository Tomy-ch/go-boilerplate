package userrepo

import (
	"context"
	"testing"

	"boilerplate-go/internal/domain/user"
	rdbdriver "boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/infrastructure/rdb/rdbtest"
	"boilerplate-go/pkg/ptr"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	db, _, z, _ := rdbtest.NewTestInstances(t)
	expected := &repository{
		db: db,
		z:  z,
	}
	actual := New(db, z)

	require.Equal(t, expected, actual)
}

func TestGetAllUsers(t *testing.T) {
	t.Parallel()

	db, txm, z, _ := rdbtest.NewTestInstances(t)

	repo := New(db, z)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("limitとoffsetを指定した場合、作成順で複数件が取得できる", func(t *testing.T) {
			t.Parallel()

			err := txm.Do(func(ctx context.Context) error {
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
					"鹿児島県",
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
					"北海道",
					"札幌",
					"1-1",
					ptr.To("Building A"),
					"060-0001",
					nil,
				)
				require.NoError(t, err)

				actual, err := repo.GetAllUsers(ctx, limit, offset)
				require.NoError(t, err)

				actualFirst := actual[0]
				actualLast := actual[len(actual)-1]

				require.Equal(t, expectedFirst, &actualFirst)
				require.Equal(t, expectedLast, &actualLast)

				return nil
			})
			require.NoError(t, err)
		})

		t.Run("limit=1でoffset=0の場合先頭のユーザーが取得できる", func(t *testing.T) {
			t.Parallel()
			err := txm.Do(func(ctx context.Context) error {
				limit := 1
				offset := 0

				expected, err := user.New(
					"eaabee3e-3b7a-4f61-8fa9-030944625e92",
					"Ivy", "Clark",
					"$2a$08$M1GUQfiCBgfhEWEirBko9.urJ3zFgU2McymmuDj3890PwPxSJRLdu6",
					"ivy.clark@example.com",
					"888-888-8888",
					"a03aaec4-3bd6-4bfb-8e47-2fbfa026d344",
					"鹿児島県",
					"鹿児島市",
					"7-7-7",
					ptr.To("Building G"),
					"890-0001",
					nil,
				)
				require.NoError(t, err)
				expectedLength := 1

				actual, err := repo.GetAllUsers(ctx, limit, offset)
				require.NoError(t, err)
				require.Len(t, actual, expectedLength)

				require.Equal(t, expected, &actual[0])
				return nil
			})
			require.NoError(t, err)
		})

		t.Run("limit=1でoffset=9の場合、末尾のユーザーが取得できる", func(t *testing.T) {
			err := txm.Do(func(ctx context.Context) error {
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
					"北海道",
					"札幌",
					"1-1",
					ptr.To("Building A"),
					"060-0001",
					nil,
				)
				require.NoError(t, getAllUsersErr)

				all, getAllUsersErr := repo.GetAllUsers(ctx, limit, offset)
				require.NoError(t, getAllUsersErr)

				actual := all[len(all)-1]

				require.Equal(t, expected, &actual)
				return nil
			})
			require.NoError(t, err)
		})

		t.Run("limit=0の場合、空配列になる", func(t *testing.T) {
			t.Parallel()
			err := txm.Do(func(ctx context.Context) error {
				limit := 0
				offset := 0
				actual, err := repo.GetAllUsers(ctx, limit, offset)
				require.NoError(t, err)
				require.Empty(t, actual)
				return nil
			})
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("limitが負数の場合、エラーになる", func(t *testing.T) {
			t.Parallel()
			err := txm.Do(func(ctx context.Context) error {
				actual, err := repo.GetAllUsers(ctx, -1, 0)
				require.Nil(t, actual)
				require.Error(t, err)
				return err
			})
			require.Error(t, err)
		})

		t.Run("offsetが負数の場合、エラーになる", func(t *testing.T) {
			t.Parallel()
			err := txm.Do(func(ctx context.Context) error {
				actual, err := repo.GetAllUsers(ctx, 10, -1)
				require.Nil(t, actual)
				require.Error(t, err)
				return err
			})
			require.Error(t, err)
		})

		t.Run("無効なユーザーが挿入されていてもDomain化の時にエラーになる", func(t *testing.T) {
			t.Parallel()
			err := txm.Do(func(ctx context.Context) error {
				drv := rdbdriver.ResolveDriverWithLog(ctx, db, z)
				_, execErr := drv.ExecContext(ctx,
					"INSERT INTO users "+
						"(id, first_name, last_name, password_hash, email, phone, prefecture_id, city, street, postal_code)"+
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

				res, actualErr := repo.GetAllUsers(ctx, 100, 0)
				require.Nil(t, res)
				require.ErrorIs(t, actualErr, user.ErrInvalidLastName)

				return actualErr
			})
			require.Error(t, err)
		})
	})
}
