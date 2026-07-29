package productstatus

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/product/status"
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

		t.Run("全商品ステータスが10件sortKey昇順で取得できる", func(t *testing.T) {
			t.Parallel()

			// seed の sortKey は code と非連動。sortKey=1 は「検討中」(code=8)。
			reviewingID, err := uuid.Parse("bdf44f06-227c-4549-b2c8-e57b32f06321")
			require.NoError(t, err)

			txm.WithinTx(func(ctx context.Context) {
				expectedReviewing, err := status.New(reviewingID, "検討中", 8, 1)
				require.NoError(t, err)

				actual, err := repo.FindAll(ctx)
				require.NoError(t, err)
				assert.Len(t, actual, 10)

				// sort_key は UNIQUE のため厳密昇順が正。
				for i := 1; i < len(actual); i++ {
					assert.Less(t, actual[i-1].SortKey(), actual[i].SortKey())
				}

				// sortKey 昇順の先頭は「検討中」(sortKey=1)。
				assert.Equal(t, expectedReviewing, actual[0])
			})
		})

		t.Run("テーブルが空の場合、nilではない空一覧を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				// product_statuses を参照する依存行（products など）ごと空にする（tx はロールバックされる）。
				_, execErr := driver.New(ctx, testDB).Exec(ctx, "TRUNCATE product_statuses CASCADE")
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

				// sort_key=0 は有効範囲(1..32767)外のため、ドメイン化に失敗する。
				_, execErr := driver.New(ctx, testDB).Exec(ctx,
					"INSERT INTO product_statuses (id, name, code, sort_key) VALUES ($1,$2,$3,$4)",
					invalidID, "テスト無効ステータス", 99, 0,
				)
				require.NoError(t, execErr)

				actual, err := repo.FindAll(ctx)
				assert.Nil(t, actual)
				require.ErrorIs(t, err, apperror.ErrInternal)
				require.NotErrorIs(t, err, status.ErrInvalidSortKey)
			})
		})
	})
}

func Test_rowToProductStatus(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な行からエンティティを再構築する", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			entity, err := rowToProductStatus(id, "在庫あり", int16(1), int16(5))
			require.NoError(t, err)
			require.NotNil(t, entity)
			assert.Equal(t, id, entity.ID())
			assert.Equal(t, "在庫あり", entity.Name())
			assert.Equal(t, 1, entity.Code())
			assert.Equal(t, 5, entity.SortKey())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("再構築時の検証失敗はErrInternalへ正規化され元の分類は露出しない", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			// sort_key=0 は有効範囲(1〜32767)外のため domain 構築が失敗する。
			entity, err := rowToProductStatus(id, "在庫あり", int16(1), int16(0))
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

		t.Run("IDから商品ステータスを取得できる", func(t *testing.T) {
			t.Parallel()

			inStockID, err := uuid.Parse("093170fb-83a2-4864-a2b3-53236eaf3597")
			require.NoError(t, err)

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindByID(ctx, inStockID)
				require.NoError(t, err)
				require.NotNil(t, actual)
				assert.Equal(t, inStockID, actual.ID())
				assert.Equal(t, "在庫あり", actual.Name())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないIDはNotFoundを返す", func(t *testing.T) {
			t.Parallel()

			missingID := uuid.NewTestFromSalt(t, "status_missing")

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindByID(ctx, missingID)
				assert.Nil(t, actual)
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})
	})
}
