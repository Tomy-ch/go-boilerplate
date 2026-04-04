package prefecture

import (
	"context"
	"testing"

	"go-boilerplate/internal/domain/prefecture"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"

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

func TestFindByName(t *testing.T) {
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

		t.Run("有効な都道府県名の場合、都道府県エンティティが取得できる", func(t *testing.T) {
			t.Parallel()

			expectedID, err := uuid.Parse("101caa1e-84e7-4ceb-9108-50d40b6be1a3")
			require.NoError(t, err)

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

		t.Run("有効な都道府県IDの場合、都道府県エンティティが取得できる", func(t *testing.T) {
			t.Parallel()

			expectedID, err := uuid.Parse("101caa1e-84e7-4ceb-9108-50d40b6be1a3")
			require.NoError(t, err)

			expectedName := "東京都"
			expectedCode := 8

			txm.WithinTx(func(ctx context.Context) {
				expected, err := prefecture.New(
					expectedID,
					expectedName,
					expectedCode,
				)
				require.NoError(t, err)

				require.NoError(t, err)
				actual, err := repo.FindByID(ctx, expectedID)
				require.NoError(t, err)
				require.Equal(t, expected, actual)
			})
		})
	})
}

func TestFindByIDs(t *testing.T) {
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

		t.Run("有効な都道府県ID一覧の場合、都道府県エンティティ一覧が取得できる", func(t *testing.T) {
			t.Parallel()

			prefectureID1, err := uuid.Parse("101caa1e-84e7-4ceb-9108-50d40b6be1a3")
			require.NoError(t, err)
			prefectureID2, err := uuid.Parse("d647fc85-ff46-4530-88cb-198f4a68a9d7")
			require.NoError(t, err)

			expectedIDs := []uuid.UUID{
				prefectureID1,
				prefectureID2,
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
				expected := make(prefecture.Prefectures, len(expectedIDs))
				ids := make([]uuid.UUID, len(expectedIDs))
				for i := range expectedIDs {
					e, err := prefecture.New(
						expectedIDs[i],
						expectedNames[i],
						expectedCodes[i],
					)
					require.NoError(t, err)
					expected[i] = e

					ids[i] = expectedIDs[i]
				}

				actual, err := repo.FindByIDs(ctx, ids)
				require.NoError(t, err)
				require.Equal(t, expected, actual)
			})
		})
	})
}
