package prefecture

import (
	"context"
	"testing"

	"boilerplate-go/internal/domain/prefecture"
	"boilerplate-go/internal/infrastructure/rdb/rdbtest"
	"boilerplate-go/internal/observability"
	"boilerplate-go/pkg/uuid"

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

func TestFindByName(t *testing.T) {
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

		t.Run("有効な都道府県名の場合、都道府県エンティティが取得できる", func(t *testing.T) {
			// t.Parallel()

			expectedID := "101caa1e-84e7-4ceb-9108-50d40b6be1a3"
			expectedName := "東京都"
			expectedCode := 8

			txm.WithinTx(func(ctx context.Context) {
				expected, err := prefecture.New(
					expectedID,
					expectedName,
					expectedCode,
				)
				require.NoError(t, err)

				actual, err := repo.FindByName(ctx, expectedName)
				require.NoError(t, err)
				require.Equal(t, expected, actual)
			})
		})
	})
}

func TestFindByID(t *testing.T) {
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

		t.Run("有効な都道府県IDの場合、都道府県エンティティが取得できる", func(t *testing.T) {
			// t.Parallel()

			expectedID := "101caa1e-84e7-4ceb-9108-50d40b6be1a3"
			expectedName := "東京都"
			expectedCode := 8

			txm.WithinTx(func(ctx context.Context) {
				expected, err := prefecture.New(
					expectedID,
					expectedName,
					expectedCode,
				)
				require.NoError(t, err)

				prefectureID, err := uuid.Parse(expectedID)
				require.NoError(t, err)
				actual, err := repo.FindByID(ctx, prefectureID)
				require.NoError(t, err)
				require.Equal(t, expected, actual)
			})
		})
	})
}

func TestFindByIDs(t *testing.T) {
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

		t.Run("有効な都道府県ID一覧の場合、都道府県エンティティ一覧が取得できる", func(t *testing.T) {
			// t.Parallel()

			expectedIDs := []string{
				"101caa1e-84e7-4ceb-9108-50d40b6be1a3",
				"d647fc85-ff46-4530-88cb-198f4a68a9d7",
			}
			expectedNames := []string{
				"東京都",
				"大阪府",
			}
			expectedCodes := []int{
				8,
				27,
			}

			txm.WithinTx(func(ctx context.Context) {
				expected := make(prefecture.Entities, len(expectedIDs))
				ids := make([]uuid.UUID, len(expectedIDs))
				for i := range expectedIDs {
					e, err := prefecture.New(
						expectedIDs[i],
						expectedNames[i],
						expectedCodes[i],
					)
					require.NoError(t, err)
					expected[i] = e

					id, err := uuid.Parse(expectedIDs[i])
					require.NoError(t, err)
					ids[i] = id
				}

				actual, err := repo.FindByIDs(ctx, ids)
				require.NoError(t, err)
				require.Equal(t, expected, actual)
			})
		})
	})
}
