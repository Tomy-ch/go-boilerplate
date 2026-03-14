package user

import (
	"context"
	"testing"

	"boilerplate-go/internal/domain/user"
	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/infrastructure/rdb/testkit"
	"boilerplate-go/internal/observability"
	"boilerplate-go/pkg/uuid"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	provider := testkit.NewTestLoggingProvider(t)
	tf := observability.NewNoopTracerFactory(t)
	expected := &service{
		tracer:   tf.Infra(),
		provider: provider,
	}
	actual := New(provider, tf)
	require.Equal(t, expected, actual)
}

func TestFindByKeyword(t *testing.T) {
	t.Parallel()
	// 保存処理などが影響しあい、テストが不安定になるため並列実行不可とする。

	provider := testkit.NewTestLoggingProvider(t)
	db := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)

	txm := testkit.NewTestTransactionManager(t)

	repo := &service{
		tracer:   lt,
		provider: provider,
	}

	t.Run("正常系", func(t *testing.T) {
		// t.Parallel()

		t.Run("キーワードにマッチするユーザーが取得できる", func(t *testing.T) {
			// t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				keywords := []string{"Grace"}
				limit := int32(10)
				offset := int32(0)

				userID, err := uuid.Parse("c688ffbc-731e-4257-82e9-d34b4712afd6")
				require.NoError(t, err)

				firstName := "Grace"
				lastName := "Lee"

				expectedLength := 1

				actual, err := repo.FindByKeyword(ctx, keywords, nil, limit, offset)
				require.NoError(t, err)
				require.Len(t, actual, expectedLength)

				require.Equal(t, userID, actual[0].ID())
				require.Equal(t, firstName, actual[0].FirstName())
				require.Equal(t, lastName, actual[0].LastName())
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
