package product

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/lexicon/money"
	domainproduct "go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newProductWithImages は、指定した画像を持つ商品エンティティを構築します。
// 画像以外の属性は固定値で、画像の入れ替えだけを差分にできます。
func newProductWithImages(t *testing.T, id uuid.UUID, images []domainproduct.Image) *domainproduct.Product {
	t.Helper()

	statusRef, err := domainproduct.NewStatusRef(mustParse(t, statusInStock), "在庫あり")
	require.NoError(t, err)
	categoryRef, err := domainproduct.NewCategoryRef(mustParse(t, categoryElectronics), "電子機器")
	require.NoError(t, err)
	price, err := money.NewPrice(decimal.FromInt(1999))
	require.NoError(t, err)

	entity, err := domainproduct.New(id, domainproduct.Attributes{
		Name:     "画像付き商品",
		Price:    price,
		Quantity: 10,
		Status:   statusRef,
		Category: categoryRef,
		Images:   images,
	})
	require.NoError(t, err)

	return entity
}

func newImage(t *testing.T, salt, path string, sortKey int) domainproduct.Image {
	t.Helper()
	return domainproduct.NewImage(
		uuidtestkit.NewTestFromSalt(t, salt),
		domainproduct.ImageAttributes{ImagePath: path, SortKey: sortKey},
	)
}

// productsRow は、ドメインへの変換元となる商品行を構築します。
func productsRow(t *testing.T, id uuid.UUID) gen.Products {
	t.Helper()
	return gen.Products{
		ID:          id,
		Name:        "画像付き商品",
		Price:       decimal.FromInt(1999),
		Quantity:    10,
		StatusID:    mustParse(t, statusInStock),
		CategoryID:  mustParse(t, categoryElectronics),
		LockVersion: 1,
	}
}

// countProductImages は、商品に紐づく画像の行数を生存・論理削除の別に数えます。
func countProductImages(ctx context.Context, t *testing.T, db driver.DBTX, id uuid.UUID) (int, int) {
	t.Helper()

	var live, deleted int
	err := db.QueryRow(ctx,
		"SELECT "+
			"COUNT(*) FILTER (WHERE deleted_at IS NULL), "+
			"COUNT(*) FILTER (WHERE deleted_at IS NOT NULL) "+
			"FROM product_images WHERE product_id = $1",
		id,
	).Scan(&live, &deleted)
	require.NoError(t, err)

	return live, deleted
}

func Test_repository_findImagesByProductIDs(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{db: testDB, tracer: observability.NewMockInfraLayerTracer(t)}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("商品IDごとの画像を表示順の昇順で返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				id := uuidtestkit.NewTestFromSalt(t, "find_images_product")
				entity := newProductWithImages(t, id, []domainproduct.Image{
					newImage(t, "find_images_2", "products/b.png", 2),
					newImage(t, "find_images_1", "products/a.png", 1),
				})
				require.NoError(t, repo.Create(ctx, entity))

				got, err := repo.findImagesByProductIDs(ctx, []uuid.UUID{id})

				require.NoError(t, err)
				require.Len(t, got[id], 2)
				assert.Equal(t, "products/a.png", got[id][0].ImagePath())
				assert.Equal(t, "products/b.png", got[id][1].ImagePath())
			})
		})

		t.Run("論理削除された画像は返さない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				id := uuidtestkit.NewTestFromSalt(t, "find_images_deleted_product")
				entity := newProductWithImages(t, id, []domainproduct.Image{
					newImage(t, "find_images_deleted", "products/old.png", 1),
				})
				require.NoError(t, repo.Create(ctx, entity))
				db := gen.New(driver.New(ctx, repo.db))
				require.NoError(t, repo.syncImages(ctx, db, newProductWithImages(t, id, nil)))

				got, err := repo.findImagesByProductIDs(ctx, []uuid.UUID{id})

				require.NoError(t, err)
				assert.Empty(t, got[id])
			})
		})

		t.Run("IDが空なら問い合わせず空を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.findImagesByProductIDs(ctx, nil)

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

			got, err := repo.findImagesByProductIDs(ctx, []uuid.UUID{uuidtestkit.NewTestFromSalt(t, "canceled")})

			assert.Nil(t, got)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

func Test_repository_syncImages(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{db: testDB, tracer: observability.NewMockInfraLayerTracer(t)}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("集合から外れた画像を論理削除し、集合の新しい画像を登録する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				id := uuidtestkit.NewTestFromSalt(t, "sync_images_product")
				require.NoError(t, repo.Create(ctx, newProductWithImages(t, id, []domainproduct.Image{
					newImage(t, "sync_before", "products/before.png", 1),
				})))

				db := gen.New(driver.New(ctx, repo.db))
				require.NoError(t, repo.syncImages(ctx, db, newProductWithImages(t, id, []domainproduct.Image{
					newImage(t, "sync_after", "products/after.png", 1),
				})))

				got, err := repo.FindByID(ctx, id)
				require.NoError(t, err)
				require.Len(t, got.Images(), 1)
				assert.Equal(t, "products/after.png", got.Images()[0].ImagePath())

				// 外れた行は消えず、論理削除として残る（差し替え履歴）。
				live, deleted := countProductImages(ctx, t, driver.New(ctx, testDB), id)
				assert.Equal(t, 1, live)
				assert.Equal(t, 1, deleted)
			})
		})

		t.Run("空の集合を渡すと現在の画像がすべて外れる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				id := uuidtestkit.NewTestFromSalt(t, "sync_images_clear_product")
				require.NoError(t, repo.Create(ctx, newProductWithImages(t, id, []domainproduct.Image{
					newImage(t, "sync_clear", "products/clear.png", 1),
				})))

				db := gen.New(driver.New(ctx, repo.db))
				require.NoError(t, repo.syncImages(ctx, db, newProductWithImages(t, id, nil)))

				got, err := repo.FindByID(ctx, id)
				require.NoError(t, err)
				assert.Empty(t, got.Images())
			})
		})

		t.Run("現在の画像と同じ集合を渡すと行が一切変化しない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				id := uuidtestkit.NewTestFromSalt(t, "sync_images_noop_product")
				require.NoError(t, repo.Create(ctx, newProductWithImages(t, id, []domainproduct.Image{
					newImage(t, "sync_noop_1", "products/keep1.png", 1),
					newImage(t, "sync_noop_2", "products/keep2.png", 2),
				})))

				loaded, err := repo.FindByID(ctx, id)
				require.NoError(t, err)

				db := gen.New(driver.New(ctx, repo.db))
				require.NoError(t, repo.syncImages(ctx, db, loaded))

				live, deleted := countProductImages(ctx, t, driver.New(ctx, testDB), id)
				assert.Equal(t, 2, live)
				assert.Equal(t, 0, deleted)

				got, err := repo.FindByID(ctx, id)
				require.NoError(t, err)
				require.Len(t, got.Images(), 2)
				assert.Equal(t, loaded.Images()[0].ID(), got.Images()[0].ID())
				assert.Equal(t, loaded.Images()[1].ID(), got.Images()[1].ID())
			})
		})

		t.Run("同じ表示順へ差し替えても部分UNIQUEに衝突しない", func(t *testing.T) {
			t.Parallel()

			// 論理削除行を残したまま同じ (product_id, sort_key) を入れ直すのが差し替えの通常経路なので、
			// ここが衝突すると 2 回目以降の差し替えが一切できなくなる。
			txm.WithinTx(func(ctx context.Context) {
				id := uuidtestkit.NewTestFromSalt(t, "sync_images_same_sort_key_product")
				require.NoError(t, repo.Create(ctx, newProductWithImages(t, id, []domainproduct.Image{
					newImage(t, "same_sort_key_1", "products/first.png", 1),
				})))

				db := gen.New(driver.New(ctx, repo.db))
				require.NoError(t, repo.syncImages(ctx, db, newProductWithImages(t, id, []domainproduct.Image{
					newImage(t, "same_sort_key_2", "products/second.png", 1),
				})))
				require.NoError(t, repo.syncImages(ctx, db, newProductWithImages(t, id, []domainproduct.Image{
					newImage(t, "same_sort_key_3", "products/third.png", 1),
				})))

				got, err := repo.FindByID(ctx, id)
				require.NoError(t, err)
				require.Len(t, got.Images(), 1)
				assert.Equal(t, "products/third.png", got.Images()[0].ImagePath())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生存行の表示順が重複する場合は衝突として返す", func(t *testing.T) {
			t.Parallel()

			// 衝突判定を主キーに限定しているため、表示順の重複は握り潰されず 23505 のまま返る。
			txm.WithinTx(func(ctx context.Context) {
				id := uuidtestkit.NewTestFromSalt(t, "sync_images_duplicate_product")
				require.NoError(t, repo.Create(ctx, newProductWithImages(t, id, []domainproduct.Image{
					newImage(t, "sync_duplicate_1", "products/first.png", 1),
				})))

				db := gen.New(driver.New(ctx, repo.db))
				err := db.CreateProductImagesIfAbsent(ctx, &gen.CreateProductImagesIfAbsentParams{
					ProductID:  id,
					Ids:        []uuid.UUID{uuidtestkit.NewTestFromSalt(t, "sync_duplicate_2")},
					ImagePaths: []string{"products/second.png"},
					SortKeys:   []int16{1},
				})

				require.ErrorIs(t, pgerror.NormalizeError(err), apperror.ErrConflict)
			})
		})

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			db := gen.New(driver.New(ctx, repo.db))
			err := repo.syncImages(ctx, db, newProductWithImages(t, uuidtestkit.NewTestFromSalt(t, "canceled"), nil))

			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

func Test_repository_insertImages(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{db: testDB, tracer: observability.NewMockInfraLayerTracer(t)}

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生存行の表示順が重複する場合は衝突として返す", func(t *testing.T) {
			t.Parallel()

			// ドメインが同一集約内の重複を弾くため通常は到達しないが、DB 側の部分 UNIQUE が
			// 最後の砦として効いていることを固定する。
			txm.WithinTx(func(ctx context.Context) {
				id := uuidtestkit.NewTestFromSalt(t, "insert_images_duplicate_product")
				require.NoError(t, repo.Create(ctx, newProductWithImages(t, id, []domainproduct.Image{
					newImage(t, "duplicate_1", "products/first.png", 1),
				})))

				db := gen.New(driver.New(ctx, repo.db))
				err := repo.insertImages(ctx, db, newProductWithImages(t, id, []domainproduct.Image{
					newImage(t, "duplicate_2", "products/second.png", 1),
				}))

				require.ErrorIs(t, err, apperror.ErrConflict)
			})
		})
	})
}

func Test_repository_buildProducts(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{db: testDB, tracer: observability.NewMockInfraLayerTracer(t)}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("行が空なら空の列を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.buildProducts(ctx, nil)

				require.NoError(t, err)
				assert.Empty(t, got)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("変換失敗があれば nil とエラーを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				invalid := productRow{
					p:            productsRow(t, uuidtestkit.NewTestFromSalt(t, "build_products_invalid")),
					statusName:   "", // status 名が空だと rowToProduct が検証失敗する
					categoryName: "電子機器",
				}

				got, err := repo.buildProducts(ctx, []productRow{invalid})

				require.Error(t, err)
				assert.Nil(t, got)
			})
		})
	})
}

func Test_repository_buildProduct(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{db: testDB, tracer: observability.NewMockInfraLayerTracer(t)}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("単一の行を画像込みのエンティティへ変換する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				id := uuidtestkit.NewTestFromSalt(t, "build_product_single")
				require.NoError(t, repo.Create(ctx, newProductWithImages(t, id, []domainproduct.Image{
					newImage(t, "build_product_image", "products/single.png", 1),
				})))

				row := productRow{
					p:            productsRow(t, id),
					statusName:   "在庫あり",
					categoryName: "電子機器",
				}

				got, err := repo.buildProduct(ctx, row)

				require.NoError(t, err)
				require.Len(t, got.Images(), 1)
				assert.Equal(t, "products/single.png", got.Images()[0].ImagePath())
			})
		})
	})
}
