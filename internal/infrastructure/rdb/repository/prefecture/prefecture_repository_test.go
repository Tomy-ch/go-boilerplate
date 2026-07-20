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

func Test_repository_FindByName(t *testing.T) {
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

func Test_repository_FindByID(t *testing.T) {
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

func Test_repository_FindAll(t *testing.T) {
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

		t.Run("全都道府県が47件code昇順で取得できる", func(t *testing.T) {
			t.Parallel()

			tokyoID, err := uuid.Parse("101caa1e-84e7-4ceb-9108-50d40b6be1a3")
			require.NoError(t, err)

			txm.WithinTx(func(ctx context.Context) {
				expectedTokyo, err := prefecture.New(tokyoID, "東京都", 8)
				require.NoError(t, err)

				actual, err := repo.FindAll(ctx)
				require.NoError(t, err)
				assert.Len(t, actual, 47)

				// code は UNIQUE のため厳密昇順が正。
				for i := 1; i < len(actual); i++ {
					assert.Less(t, actual[i-1].Code(), actual[i].Code())
				}

				var actualTokyo *prefecture.Prefecture
				for _, p := range actual {
					if p.ID() == tokyoID {
						actualTokyo = p
						break
					}
				}
				require.NotNil(t, actualTokyo)
				assert.Equal(t, expectedTokyo, actualTokyo)
			})
		})

		t.Run("テーブルが空の場合、nilではない空一覧を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				// prefectures を参照する依存行（users など）ごと空にする（tx はロールバックされる）。
				_, execErr := driver.New(ctx, testDB).Exec(ctx, "TRUNCATE prefectures CASCADE")
				require.NoError(t, execErr)

				actual, err := repo.FindAll(ctx)
				require.NoError(t, err)
				assert.NotNil(t, actual)
				assert.Empty(t, actual)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化される", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			actual, err := repo.FindAll(ctx)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})

		t.Run("取得行のドメイン化に失敗した場合、データ不整合としてErrInternalを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				invalidID, err := uuid.Parse("00000000-0000-0000-0000-0000000000fe")
				require.NoError(t, err)

				// code=99 は都道府県コードの有効範囲(1..47)外のため、ドメイン化に失敗する。
				_, execErr := driver.New(ctx, testDB).Exec(ctx,
					"INSERT INTO prefectures (id, name, code) VALUES ($1,$2,$3)",
					invalidID, "テスト無効県", 99,
				)
				require.NoError(t, execErr)

				actual, err := repo.FindAll(ctx)
				assert.Nil(t, actual)
				require.ErrorIs(t, err, apperror.ErrInternal)
				require.NotErrorIs(t, err, prefecture.ErrInvalidCode)
			})
		})
	})
}

func Test_repository_FindByIDs(t *testing.T) {
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

		t.Run("取得行のドメイン化に失敗した場合、データ不整合としてErrInternalを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				invalidID, err := uuid.Parse("00000000-0000-0000-0000-0000000000ff")
				require.NoError(t, err)

				// code=99 は都道府県コードの有効範囲(1..47)外のため、ドメイン化に失敗する。
				_, execErr := driver.New(ctx, testDB).Exec(ctx,
					"INSERT INTO prefectures (id, name, code) VALUES ($1,$2,$3)",
					invalidID, "テスト無効県", 99,
				)
				require.NoError(t, execErr)

				actual, err := repo.FindByIDs(ctx, []uuid.UUID{invalidID})
				assert.Nil(t, actual)
				require.ErrorIs(t, err, apperror.ErrInternal)
				require.NotErrorIs(t, err, prefecture.ErrInvalidCode)
			})
		})
	})
}
