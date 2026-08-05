package productcategory

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/product/category"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

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

		t.Run("全商品カテゴリが5件sortKey昇順で取得できる", func(t *testing.T) {
			t.Parallel()

			electronicsID, err := uuid.Parse("5dd52d84-78eb-4a52-ba0b-2e11c95c2af2")
			require.NoError(t, err)

			txm.WithinTx(func(ctx context.Context) {
				expectedElectronics, err := category.New(electronicsID, category.Attributes{Name: "電子機器", Code: 1, SortKey: 1})
				require.NoError(t, err)

				actual, err := repo.FindAll(ctx)
				require.NoError(t, err)
				assert.Len(t, actual, 5)

				// sort_key は UNIQUE のため厳密昇順が正。
				for i := 1; i < len(actual); i++ {
					assert.Less(t, actual[i-1].SortKey(), actual[i].SortKey())
				}

				assert.Equal(t, expectedElectronics, actual[0])
			})
		})

		t.Run("テーブルが空の場合、nilではない空一覧を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				// product_categories を参照する依存行（products など）ごと空にする（tx はロールバックされる）。
				_, execErr := driver.New(ctx, testDB).Exec(ctx, "TRUNCATE product_categories CASCADE")
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

				// sort_key=0 は表示順の有効範囲(1..32767)外のため、ドメイン化に失敗する。
				_, execErr := driver.New(ctx, testDB).Exec(ctx,
					"INSERT INTO product_categories (id, name, code, sort_key) VALUES ($1,$2,$3,$4)",
					invalidID, "テスト無効カテゴリ", 99, 0,
				)
				require.NoError(t, execErr)

				actual, err := repo.FindAll(ctx)
				assert.Nil(t, actual)
				require.ErrorIs(t, err, apperror.ErrInternal)
				require.NotErrorIs(t, err, category.ErrInvalidSortKey)
			})
		})
	})
}

func Test_rowToProductCategory(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な行からエンティティを再構築する", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			entity, err := rowToProductCategory(id, "電子機器", int16(1), int16(1))
			require.NoError(t, err)
			require.NotNil(t, entity)
			assert.Equal(t, id, entity.ID())
			assert.Equal(t, "電子機器", entity.Name())
			assert.Equal(t, 1, entity.Code())
			assert.Equal(t, 1, entity.SortKey())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("再構築時の検証失敗はErrInternalへ正規化され元の分類は露出しない", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			// sort_key=0 は有効範囲(1〜32767)外のため domain 構築が失敗する。
			entity, err := rowToProductCategory(id, "電子機器", int16(1), int16(0))
			require.Error(t, err)
			require.Nil(t, entity)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func Test_repository_FindByID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{tracer: lt, db: testDB}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDから商品カテゴリを取得できる", func(t *testing.T) {
			t.Parallel()

			electronicsID, err := uuid.Parse("5dd52d84-78eb-4a52-ba0b-2e11c95c2af2")
			require.NoError(t, err)

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindByID(ctx, electronicsID)
				require.NoError(t, err)
				require.NotNil(t, actual)
				assert.Equal(t, electronicsID, actual.ID())
				assert.Equal(t, "電子機器", actual.Name())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないIDはNotFoundを返す", func(t *testing.T) {
			t.Parallel()

			missingID := uuidtestkit.NewTestFromSalt(t, "category_missing")

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindByID(ctx, missingID)
				assert.Nil(t, actual)
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})
	})
}
