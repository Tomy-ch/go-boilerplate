package product

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// insertProductWithImage は、画像を持つ商品を挿入するヘルパーです。imagePath=nil で画像なしになります。
// deletedAt に値を渡すと、その画像は論理削除済みとして挿入されます。
func insertProductWithImage(
	ctx context.Context, t *testing.T, db driver.DBTX, id, name string, imagePath *string, deletedAt *time.Time,
) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO products "+
			"(id, name, description, price, quantity, stock_warning_threshold, status_id, category_id) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
		id, name, nil, 100, 10, nil, statusInStock, categoryElectronics,
	)
	require.NoError(t, err)

	if imagePath == nil {
		return
	}

	_, err = db.Exec(ctx,
		"INSERT INTO product_images (id, product_id, image_path, display_sort, deleted_at) VALUES ($1,$2,$3,$4,$5)",
		uuidtestkit.NewTestFromSalt(t, "image_of_"+id), id, *imagePath, 1, deletedAt,
	)
	require.NoError(t, err)
}

func Test_repository_FilterExistingImagePaths(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{db: testDB, tracer: observability.NewMockInfraLayerTracer(t)}

	referenced := "products/f0000000-0000-4000-8000-000000000001.png"
	orphan := "products/f0000000-0000-4000-8000-000000000002.png"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("商品が参照しているパスだけを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertProductWithImage(ctx, t, drv, "f1000000-0000-4000-8000-000000000001", "画像あり商品", &referenced, nil)

				got, err := repo.FilterExistingImagePaths(ctx, []string{referenced, orphan})

				require.NoError(t, err)
				assert.Equal(t, []string{referenced}, got)
			})
		})

		t.Run("どの商品も参照していなければ空を返す", func(t *testing.T) {
			t.Parallel()

			// ここが「参照あり」に倒れると、生きている画像を孤児と誤判定して不可逆に消すことになる。
			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.FilterExistingImagePaths(ctx, []string{orphan})

				require.NoError(t, err)
				assert.Empty(t, got)
			})
		})

		t.Run("複数商品が同じパスを参照していても重複させない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertProductWithImage(ctx, t, drv, "f1000000-0000-4000-8000-000000000002", "同一画像A", &referenced, nil)
				insertProductWithImage(ctx, t, drv, "f1000000-0000-4000-8000-000000000003", "同一画像B", &referenced, nil)

				got, err := repo.FilterExistingImagePaths(ctx, []string{referenced})

				require.NoError(t, err)
				assert.Equal(t, []string{referenced}, got)
			})
		})

		t.Run("画像を持たない商品は結果に現れない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertProductWithImage(ctx, t, drv, "f1000000-0000-4000-8000-000000000004", "画像なし商品", nil, nil)

				got, err := repo.FilterExistingImagePaths(ctx, []string{orphan})

				require.NoError(t, err)
				assert.Empty(t, got)
			})
		})

		t.Run("論理削除された画像のパスは参照とみなさない", func(t *testing.T) {
			t.Parallel()

			// ここが「参照あり」に倒れると、差し替えで外れた画像がストレージに残り続けて回収されない。
			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				deletedAt := time.Now()
				insertProductWithImage(
					ctx, t, drv, "f1000000-0000-4000-8000-000000000005", "差し替え済み商品", &orphan, &deletedAt,
				)

				got, err := repo.FilterExistingImagePaths(ctx, []string{orphan})

				require.NoError(t, err)
				assert.Empty(t, got)
			})
		})

		t.Run("パスが空なら問い合わせず空を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.FilterExistingImagePaths(ctx, nil)

				require.NoError(t, err)
				assert.Empty(t, got)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			got, err := repo.FilterExistingImagePaths(ctx, []string{orphan})

			assert.Nil(t, got)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}
