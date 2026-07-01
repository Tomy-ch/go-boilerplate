package prefecture

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/prefecture"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	tf := observability.NewNoopTracerFactory(t)
	expected := &repository{
		tracer: tf.Infra(),
		db:     testDB,
	}
	actual := New(testDB, tf)
	assert.Equal(t, expected, actual)
}

func TestFindByName(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)

	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{
		tracer: lt,
		db:     testDB,
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
				assert.Equal(t, expected, actual)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しない都道府県名の場合、ErrNotFoundを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindByName(ctx, "存在しない県")
				require.ErrorIs(t, err, apperror.ErrNotFound)
				assert.Nil(t, actual)
			})
		})
	})
}

func TestFindByID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)

	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{
		tracer: lt,
		db:     testDB,
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

				actual, err := repo.FindByID(ctx, expectedID)
				require.NoError(t, err)
				assert.Equal(t, expected, actual)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しない都道府県IDの場合、ErrNotFoundを返す", func(t *testing.T) {
			t.Parallel()

			notExistID, err := uuid.Parse("00000000-0000-0000-0000-000000000000")
			require.NoError(t, err)

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindByID(ctx, notExistID)
				require.ErrorIs(t, err, apperror.ErrNotFound)
				assert.Nil(t, actual)
			})
		})
	})
}

func TestFindByIDs(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)

	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{
		tracer: lt,
		db:     testDB,
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
				assert.Equal(t, expected, actual)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化される", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.Parse("101caa1e-84e7-4ceb-9108-50d40b6be1a3")
			require.NoError(t, err)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			actual, err := repo.FindByIDs(ctx, []uuid.UUID{id})
			assert.Nil(t, actual)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})

		t.Run("取得行のドメイン化に失敗した場合、そのエラーを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				invalidID, err := uuid.Parse("00000000-0000-0000-0000-0000000000ff")
				require.NoError(t, err)

				// code=99 は都道府県コードの有効範囲(1..47)外のため、ドメイン化で ErrInvalidCode となる。
				_, execErr := driver.New(ctx, testDB).Exec(ctx,
					"INSERT INTO prefectures (id, name, code) VALUES ($1,$2,$3)",
					invalidID, "テスト無効県", 99,
				)
				require.NoError(t, execErr)

				actual, err := repo.FindByIDs(ctx, []uuid.UUID{invalidID})
				assert.Nil(t, actual)
				require.ErrorIs(t, err, prefecture.ErrInvalidCode)
			})
		})
	})
}
