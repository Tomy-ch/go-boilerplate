package product

import (
	"context"
	"sync"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/domain/kernel/money"
	domainproduct "go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/infrastructure/system"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 既存 seed に含まれるカテゴリ / ステータス ID（FK 制約を満たすために使用）。
const (
	categoryElectronics = "5dd52d84-78eb-4a52-ba0b-2e11c95c2af2"
	categoryBooks       = "b39be992-fe5a-4b4c-9f98-e695f0f5101e"
	statusInStock       = "093170fb-83a2-4864-a2b3-53236eaf3597"
	statusOutOfStock    = "f33654fe-1041-498d-be18-3a1384c10df4"
	// probeKeyword は、seed データと隔離してテスト挿入行のみを対象化するための一意なキーワードです。
	probeKeyword = "KEYSETPROBE563"
)

// insertProduct は、keyset 検証用に published_at / id を明示した商品を挿入するヘルパーです。
func insertProduct(
	ctx context.Context, t *testing.T, db driver.DBTX,
	id, name string, description *string, price int, statusID, categoryID string, publishedAt *time.Time,
) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO products "+
			"(id, name, description, price, quantity, stock_warning_threshold, status_id, category_id, published_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)",
		id, name, description, price, 10, nil, statusID, categoryID, publishedAt,
	)
	require.NoError(t, err)
}

func mustParse(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	require.NoError(t, err)
	return id
}

func Test_repository_FindPublishedList(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{tracer: lt, db: testDB}

	// seed より確実に新しい固定時刻。id の大小でタイブレークを検証するため tie ペアを含む。
	base := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)
	tieHigh := "eeeeeeee-0000-4000-8000-000000000002"
	tieLow := "eeeeeeee-0000-4000-8000-000000000001"
	mid := "eeeeeeee-0000-4000-8000-000000000003"  // base-1h
	old := "eeeeeeee-0000-4000-8000-000000000004"  // base-2h
	null := "eeeeeeee-0000-4000-8000-000000000005" // published_at=NULL

	// insertProbeSet は、tie ペア + 2 件 + 未公開(NULL) を probeKeyword 付きで挿入します。
	insertProbeSet := func(ctx context.Context, t *testing.T, drv driver.DBTX) {
		t.Helper()
		insertProduct(ctx, t, drv, tieHigh, probeKeyword+"-A", nil, 1000, statusInStock, categoryElectronics, ptr.To(base))
		insertProduct(ctx, t, drv, tieLow, probeKeyword+"-B", nil, 1000, statusInStock, categoryElectronics, ptr.To(base))
		insertProduct(ctx, t, drv, mid, probeKeyword+"-C", nil, 1000, statusInStock, categoryElectronics, ptr.To(base.Add(-time.Hour)))
		insertProduct(ctx, t, drv, old, probeKeyword+"-D", nil, 1000, statusInStock, categoryElectronics, ptr.To(base.Add(-2*time.Hour)))
		insertProduct(ctx, t, drv, null, probeKeyword+"-NULL", nil, 1000, statusInStock, categoryElectronics, nil)
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("降順で先頭ページとafter境界がkeyset安定順に次ページを返し未公開行は除外される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertProbeSet(ctx, t, drv)

				// 先頭ページ(limit=2): 公開日時降順・id降順で tieHigh, tieLow。
				firstPage, err := repo.FindPublishedList(ctx, domainproduct.ListParams{
					Limit: 2, Ascending: false, Keyword: ptr.To(probeKeyword),
				})
				require.NoError(t, err)
				require.Len(t, firstPage, 2)
				assert.Equal(t, mustParse(t, tieHigh), firstPage[0].ID())
				assert.Equal(t, mustParse(t, tieLow), firstPage[1].ID())

				// 次ページ: 末尾行(tieLow)を境界に keyset を進めると mid → old。
				last := firstPage[len(firstPage)-1]
				secondPage, err := repo.FindPublishedList(ctx, domainproduct.ListParams{
					Limit: 2, Ascending: false, Keyword: ptr.To(probeKeyword),
					AfterPublishedAt: last.PublishedAt(), AfterID: ptr.To(last.ID()),
				})
				require.NoError(t, err)
				require.Len(t, secondPage, 2)
				assert.Equal(t, mustParse(t, mid), secondPage[0].ID())
				assert.Equal(t, mustParse(t, old), secondPage[1].ID())

				// 未公開(NULL)行はどのページにも現れない。
				all, err := repo.FindPublishedList(ctx, domainproduct.ListParams{
					Limit: 100, Ascending: false, Keyword: ptr.To(probeKeyword),
				})
				require.NoError(t, err)
				for _, p := range all {
					assert.NotEqual(t, mustParse(t, null), p.ID())
				}
			})
		})

		t.Run("昇順では公開日時の古い順に返る", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertProbeSet(ctx, t, drv)

				page, err := repo.FindPublishedList(ctx, domainproduct.ListParams{
					Limit: 10, Ascending: true, Keyword: ptr.To(probeKeyword),
				})
				require.NoError(t, err)
				require.Len(t, page, 4)
				assert.Equal(t, mustParse(t, old), page[0].ID())
				assert.Equal(t, mustParse(t, mid), page[1].ID())
				assert.Equal(t, mustParse(t, tieLow), page[2].ID())
				assert.Equal(t, mustParse(t, tieHigh), page[3].ID())
			})
		})

		t.Run("category_idフィルタは該当カテゴリのみを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				bookID := "eeeeeeee-0000-4000-8000-0000000000b1"
				insertProduct(ctx, t, drv, tieHigh, probeKeyword+"-E", nil, 1000, statusInStock, categoryElectronics, ptr.To(base))
				insertProduct(ctx, t, drv, bookID, probeKeyword+"-BOOK", nil, 1000, statusInStock, categoryBooks, ptr.To(base))

				categoryID := mustParse(t, categoryBooks)
				page, err := repo.FindPublishedList(ctx, domainproduct.ListParams{
					Limit: 10, Ascending: false, Keyword: ptr.To(probeKeyword), CategoryID: &categoryID,
				})
				require.NoError(t, err)
				require.Len(t, page, 1)
				assert.Equal(t, mustParse(t, bookID), page[0].ID())
			})
		})

		t.Run("status_idフィルタは該当ステータスのみを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				outID := "eeeeeeee-0000-4000-8000-0000000000c1"
				insertProduct(ctx, t, drv, tieHigh, probeKeyword+"-F", nil, 1000, statusInStock, categoryElectronics, ptr.To(base))
				insertProduct(ctx, t, drv, outID, probeKeyword+"-OUT", nil, 1000, statusOutOfStock, categoryElectronics, ptr.To(base))

				statusID := mustParse(t, statusOutOfStock)
				page, err := repo.FindPublishedList(ctx, domainproduct.ListParams{
					Limit: 10, Ascending: false, Keyword: ptr.To(probeKeyword), StatusID: &statusID,
				})
				require.NoError(t, err)
				require.Len(t, page, 1)
				assert.Equal(t, mustParse(t, outID), page[0].ID())
			})
		})

		t.Run("keywordは名称と説明の両方に部分一致する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				nameHit := "eeeeeeee-0000-4000-8000-0000000000d1"
				descHit := "eeeeeeee-0000-4000-8000-0000000000d2"
				// name に一致。
				insertProduct(ctx, t, drv, nameHit, probeKeyword+"-NAME", nil, 1000, statusInStock, categoryElectronics, ptr.To(base))
				// description に一致（name は非一致）。
				insertProduct(
					ctx,
					t,
					drv,
					descHit,
					"無関係な商品名",
					ptr.To("これは "+probeKeyword+" を含む説明"),
					1000,
					statusInStock,
					categoryElectronics,
					ptr.To(base.Add(-time.Hour)),
				)

				page, err := repo.FindPublishedList(ctx, domainproduct.ListParams{
					Limit: 10, Ascending: false, Keyword: ptr.To(probeKeyword),
				})
				require.NoError(t, err)
				require.Len(t, page, 2)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("limitが負数の場合、ErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindPublishedList(ctx, domainproduct.ListParams{Limit: -1, Ascending: false})
				assert.Nil(t, actual)
				require.ErrorIs(t, err, apperror.ErrInternal)
			})
		})

		t.Run("不正な行(価格が負数)が含まれるとデータ不整合としてErrInternalを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				invalidID := "eeeeeeee-0000-4000-8000-0000000000e1"
				// price=-1 はドメイン不変条件違反(再構築エラー)を誘発する。
				insertProduct(ctx, t, drv, invalidID, probeKeyword+"-BADPRICE", nil, -1, statusInStock, categoryElectronics, ptr.To(base))

				actual, err := repo.FindPublishedList(ctx, domainproduct.ListParams{
					Limit: 10, Ascending: false, Keyword: ptr.To(probeKeyword),
				})
				assert.Nil(t, actual)
				require.ErrorIs(t, err, apperror.ErrInternal)
				require.NotErrorIs(t, err, money.ErrNegativePrice)
			})
		})
	})
}

func Test_repository_FindPublishedByID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{tracer: lt, db: testDB}

	publishedID := "ffffffff-0000-4000-8000-000000000001"
	unpublishedID := "ffffffff-0000-4000-8000-000000000002"
	base := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("公開中の商品をIDで取得できる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertProduct(ctx, t, drv, publishedID, probeKeyword+"-PUB", ptr.To("説明"), 1999, statusInStock, categoryElectronics, ptr.To(base))

				got, err := repo.FindPublishedByID(ctx, mustParse(t, publishedID))
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, mustParse(t, publishedID), got.ID())
				require.NotNil(t, got.Description())
				assert.Equal(t, "説明", *got.Description())
				assert.Equal(t, "1999", got.Price().String())
				assert.Equal(t, 10, got.Quantity())
				assert.Nil(t, got.StockWarningThreshold())
				// 固定参照マスタとの結合で解決した status / category を ID・名称とも実 DB 経由で固定する
				// （SQL の結合エイリアス取り違え＝status/category の名称入れ替わりを検出するため、異なる 2 値で検証）。
				assert.Equal(t, mustParse(t, statusInStock), got.Status().ID())
				assert.Equal(t, "在庫あり", got.Status().Name())
				assert.Equal(t, mustParse(t, categoryElectronics), got.Category().ID())
				assert.Equal(t, "電子機器", got.Category().Name())
				// DB は timestamptz をローカルタイムゾーンで返すため、格納した瞬間の一致で比較する。
				require.NotNil(t, got.PublishedAt())
				assert.True(t, base.Equal(*got.PublishedAt()))
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未存在のIDはNotFoundを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.FindPublishedByID(ctx, uuid.NewTestFromSalt(t, "find_published_by_id_missing"))
				assert.Nil(t, got)
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})

		t.Run("非公開(published_atがNULL)の商品はNotFoundを返し存在を秘匿する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				// published_at が NULL の行は公開述語に合致しないため、未存在と同じ NotFound が返る（存在秘匿）。
				insertProduct(ctx, t, drv, unpublishedID, probeKeyword+"-UNPUB", nil, 1999, statusInStock, categoryElectronics, nil)

				got, err := repo.FindPublishedByID(ctx, mustParse(t, unpublishedID))
				assert.Nil(t, got)
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})

		t.Run("不正な行(価格が負数)が取得されるとデータ不整合としてErrInternalを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				invalidID := "ffffffff-0000-4000-8000-000000000003"
				// price=-1 はドメイン不変条件違反(再構築エラー)を誘発する。公開述語を通すため published_at は設定する。
				insertProduct(ctx, t, drv, invalidID, probeKeyword+"-BADPRICE", nil, -1, statusInStock, categoryElectronics, ptr.To(base))

				got, err := repo.FindPublishedByID(ctx, mustParse(t, invalidID))
				assert.Nil(t, got)
				require.ErrorIs(t, err, apperror.ErrInternal)
				require.NotErrorIs(t, err, money.ErrNegativePrice)
			})
		})
	})
}

func Test_int32PtrToIntPtr(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilの場合はnilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, int32PtrToIntPtr(nil))
		})

		t.Run("非nilの場合は値を保持したintポインタを返す", func(t *testing.T) {
			t.Parallel()
			v := int32(7)
			got := int32PtrToIntPtr(&v)
			require.NotNil(t, got)
			assert.Equal(t, 7, *got)
		})
	})
}

func Test_rowToProduct(t *testing.T) {
	t.Parallel()

	publishedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	id := uuid.NewTestFromSalt(t, "row_to_product_id")
	statusID := uuid.NewTestFromSalt(t, "row_to_product_status")
	categoryID := uuid.NewTestFromSalt(t, "row_to_product_category")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("商品行(マスタ JOIN 込み)をドメインエンティティへ変換する", func(t *testing.T) {
			t.Parallel()
			row := gen.Products{
				ID:                    id,
				Name:                  "商品",
				Description:           ptr.To("説明"),
				Price:                 decimal.FromInt(1999),
				Quantity:              100,
				StockWarningThreshold: ptr.To(int32(10)),
				StatusID:              statusID,
				CategoryID:            categoryID,
				PublishedAt:           ptr.To(publishedAt),
				ImagePath:             ptr.To("products/earphone.png"),
				LockVersion:           3,
			}
			got, err := rowToProduct(productRow{p: row, statusName: "在庫あり", categoryName: "電子機器"})
			require.NoError(t, err)
			assert.Equal(t, id, got.ID())
			assert.Equal(t, "商品", got.Name())
			assert.Equal(t, "1999", got.Price().String())
			assert.Equal(t, 100, got.Quantity())
			require.NotNil(t, got.StockWarningThreshold())
			assert.Equal(t, 10, *got.StockWarningThreshold())
			assert.Equal(t, statusID, got.Status().ID())
			assert.Equal(t, "在庫あり", got.Status().Name())
			assert.Equal(t, categoryID, got.Category().ID())
			assert.Equal(t, "電子機器", got.Category().Name())
			require.NotNil(t, got.PublishedAt())
			assert.Equal(t, publishedAt, *got.PublishedAt())
			// 同型の description / image_path は、取り違えを検出できるよう異なる値で対応を固定する。
			require.NotNil(t, got.Description())
			assert.Equal(t, "説明", *got.Description())
			require.NotNil(t, got.ImagePath())
			assert.Equal(t, "products/earphone.png", *got.ImagePath())
			assert.Equal(t, 3, got.Version())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不変条件違反(価格が負数)はデータ不整合としてErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()
			row := gen.Products{
				ID:          id,
				Name:        "商品",
				Price:       decimal.FromInt(-1),
				Quantity:    100,
				StatusID:    statusID,
				CategoryID:  categoryID,
				PublishedAt: ptr.To(publishedAt),
			}
			got, err := rowToProduct(productRow{p: row, statusName: "在庫あり", categoryName: "電子機器"})
			assert.Nil(t, got)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("JOIN で解決したステータス名が空の場合はデータ不整合としてErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()
			row := gen.Products{
				ID:          id,
				Name:        "商品",
				Price:       decimal.FromInt(1999),
				Quantity:    100,
				StatusID:    statusID,
				CategoryID:  categoryID,
				PublishedAt: ptr.To(publishedAt),
			}
			got, err := rowToProduct(productRow{p: row, statusName: "", categoryName: "電子機器"})
			assert.Nil(t, got)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("JOIN で解決したカテゴリ名が空の場合はデータ不整合としてErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()
			row := gen.Products{
				ID:          id,
				Name:        "商品",
				Price:       decimal.FromInt(1999),
				Quantity:    100,
				StatusID:    statusID,
				CategoryID:  categoryID,
				PublishedAt: ptr.To(publishedAt),
			}
			got, err := rowToProduct(productRow{p: row, statusName: "在庫あり", categoryName: ""})
			assert.Nil(t, got)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("バージョンが下限未満の行はエンティティ再構築の失敗としてErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()
			row := gen.Products{
				ID:          id,
				Name:        "商品",
				Price:       decimal.FromInt(1999),
				Quantity:    100,
				StatusID:    statusID,
				CategoryID:  categoryID,
				PublishedAt: ptr.To(publishedAt),
				LockVersion: 0,
			}
			got, err := rowToProduct(productRow{p: row, statusName: "在庫あり", categoryName: "電子機器"})
			assert.Nil(t, got)
			require.ErrorIs(t, err, apperror.ErrInternal)
			require.NotErrorIs(t, err, domainproduct.ErrInvalidVersion)
		})
	})
}

func Test_repository_Create(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{tracer: lt, db: testDB}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("商品を登録し image_path / published_at / 説明を往復して取得できる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				id := uuid.NewTestFromSalt(t, "create_roundtrip_id")
				publishedAt := time.Date(2099, time.June, 1, 0, 0, 0, 0, time.UTC)
				statusRef, err := domainproduct.NewStatusRef(mustParse(t, statusInStock), "在庫あり")
				require.NoError(t, err)
				categoryRef, err := domainproduct.NewCategoryRef(mustParse(t, categoryElectronics), "電子機器")
				require.NoError(t, err)
				price, err := money.NewPrice(decimal.FromInt(1999))
				require.NoError(t, err)
				entity, err := domainproduct.New(id, domainproduct.Attributes{
					Name:                  "作成商品",
					Description:           ptr.To("<p>リッチテキスト説明</p>"),
					Price:                 price,
					Quantity:              100,
					StockWarningThreshold: ptr.To(10),
					Status:                statusRef,
					Category:              categoryRef,
					PublishedAt:           ptr.To(publishedAt),
					ImagePath:             ptr.To("products/created.png"),
				})
				require.NoError(t, err)

				require.NoError(t, repo.Create(ctx, entity))

				got, err := repo.FindPublishedByID(ctx, id)
				require.NoError(t, err)
				assert.Equal(t, "作成商品", got.Name())
				require.NotNil(t, got.Description())
				assert.Equal(t, "<p>リッチテキスト説明</p>", *got.Description())
				require.NotNil(t, got.ImagePath())
				assert.Equal(t, "products/created.png", *got.ImagePath())
				require.NotNil(t, got.PublishedAt())
				assert.True(t, publishedAt.Equal(*got.PublishedAt()))
			})
		})

		t.Run("image_path / published_at が nil の場合、DB へ NULL として登録される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				id := uuid.NewTestFromSalt(t, "create_nil_optional_id")
				statusRef, err := domainproduct.NewStatusRef(mustParse(t, statusInStock), "在庫あり")
				require.NoError(t, err)
				categoryRef, err := domainproduct.NewCategoryRef(mustParse(t, categoryElectronics), "電子機器")
				require.NoError(t, err)
				price, err := money.NewPrice(decimal.FromInt(500))
				require.NoError(t, err)
				entity, err := domainproduct.New(id, domainproduct.Attributes{
					Name:     "未公開商品",
					Price:    price,
					Quantity: 0,
					Status:   statusRef,
					Category: categoryRef,
				})
				require.NoError(t, err)

				require.NoError(t, repo.Create(ctx, entity))

				// 公開述語(published_at IS NOT NULL)で除外されるため FindPublishedByID では読み戻せない。
				// 実際に NULL 列として書き込まれたことを生 SQL で直接検証する。
				var publishedAt *time.Time
				var imagePath *string
				err = driver.New(ctx, testDB).
					QueryRow(ctx, "SELECT published_at, image_path FROM products WHERE id = $1", id).
					Scan(&publishedAt, &imagePath)
				require.NoError(t, err)
				assert.Nil(t, publishedAt)
				assert.Nil(t, imagePath)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化される", func(t *testing.T) {
			t.Parallel()

			id := uuid.NewTestFromSalt(t, "create_canceled_id")
			statusRef, err := domainproduct.NewStatusRef(mustParse(t, statusInStock), "在庫あり")
			require.NoError(t, err)
			categoryRef, err := domainproduct.NewCategoryRef(mustParse(t, categoryElectronics), "電子機器")
			require.NoError(t, err)
			price, err := money.NewPrice(decimal.FromInt(100))
			require.NoError(t, err)
			entity, err := domainproduct.New(id, domainproduct.Attributes{
				Name:     "キャンセル商品",
				Price:    price,
				Quantity: 1,
				Status:   statusRef,
				Category: categoryRef,
			})
			require.NoError(t, err)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			// db.CreateProduct が context.Canceled を返し、pgerror.NormalizeError が ErrCanceled へ正規化する。
			err = repo.Create(ctx, entity)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

func Test_repository_FindByID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{tracer: lt, db: testDB}

	unpublishedID := "aaaaaaaa-0000-4000-8000-000000000001"
	versionedID := "aaaaaaaa-0000-4000-8000-000000000002"
	base := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非公開(published_atがNULL)の商品も取得できる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertProduct(ctx, t, drv, unpublishedID, probeKeyword+"-UNPUB-ANY", nil, 1999, statusInStock, categoryElectronics, nil)

				got, err := repo.FindByID(ctx, mustParse(t, unpublishedID))
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, mustParse(t, unpublishedID), got.ID())
				assert.Equal(t, probeKeyword+"-UNPUB-ANY", got.Name())
				assert.Nil(t, got.PublishedAt())
			})
		})

		t.Run("DBに永続化されたversionがエンティティに反映される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertProduct(ctx, t, drv, versionedID, probeKeyword+"-VER", nil, 1999, statusInStock, categoryElectronics, ptr.To(base))
				// 初期値(1)との区別が付くよう、DB 側のバージョンだけを進めた行を用意する。
				_, err := drv.Exec(ctx, "UPDATE products SET lock_version = 3 WHERE id = $1", mustParse(t, versionedID))
				require.NoError(t, err)

				got, err := repo.FindByID(ctx, mustParse(t, versionedID))
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, 3, got.Version())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未存在のIDはNotFoundを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.FindByID(ctx, uuid.NewTestFromSalt(t, "find_by_id_missing"))
				assert.Nil(t, got)
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})
	})
}

func Test_repository_Update(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{tracer: lt, db: testDB}

	updatedID := "bbbbbbbb-0000-4000-8000-000000000001"
	conflictID := "bbbbbbbb-0000-4000-8000-000000000002"
	base := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("更新に成功すると採番後のversionを返しDBの行が更新される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertProduct(ctx, t, drv, updatedID, probeKeyword+"-UPD", nil, 1999, statusInStock, categoryElectronics, ptr.To(base))

				entity, err := repo.FindByID(ctx, mustParse(t, updatedID))
				require.NoError(t, err)
				require.Equal(t, 1, entity.Version())

				statusRef, err := domainproduct.NewStatusRef(mustParse(t, statusOutOfStock), "在庫切れ")
				require.NoError(t, err)
				categoryRef, err := domainproduct.NewCategoryRef(mustParse(t, categoryBooks), "書籍")
				require.NoError(t, err)
				price, err := money.NewPrice(decimal.FromInt(2500))
				require.NoError(t, err)
				publishedAt := base.Add(24 * time.Hour)
				require.NoError(t, entity.Update(domainproduct.Attributes{
					Name:                  "更新後商品",
					Description:           ptr.To("更新後の説明"),
					Price:                 price,
					Quantity:              5,
					StockWarningThreshold: ptr.To(3),
					Status:                statusRef,
					Category:              categoryRef,
					PublishedAt:           ptr.To(publishedAt),
					ImagePath:             ptr.To("products/updated.png"),
				}))

				version, err := repo.Update(ctx, entity)
				require.NoError(t, err)
				assert.Equal(t, 2, version)

				got, err := repo.FindByID(ctx, mustParse(t, updatedID))
				require.NoError(t, err)
				assert.Equal(t, "更新後商品", got.Name())
				require.NotNil(t, got.Description())
				assert.Equal(t, "更新後の説明", *got.Description())
				assert.Equal(t, "2500", got.Price().String())
				assert.Equal(t, 5, got.Quantity())
				require.NotNil(t, got.StockWarningThreshold())
				assert.Equal(t, 3, *got.StockWarningThreshold())
				assert.Equal(t, mustParse(t, statusOutOfStock), got.Status().ID())
				assert.Equal(t, mustParse(t, categoryBooks), got.Category().ID())
				require.NotNil(t, got.PublishedAt())
				assert.True(t, publishedAt.Equal(*got.PublishedAt()))
				require.NotNil(t, got.ImagePath())
				assert.Equal(t, "products/updated.png", *got.ImagePath())
				assert.Equal(t, 2, got.Version())
			})
		})

		t.Run("description / stock_warning_threshold / published_at / image_path を nil で更新するとDBへNULLとして書き込まれる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				id := uuid.NewTestFromSalt(t, "update_clear_nullable_id")
				statusRef, err := domainproduct.NewStatusRef(mustParse(t, statusInStock), "在庫あり")
				require.NoError(t, err)
				categoryRef, err := domainproduct.NewCategoryRef(mustParse(t, categoryElectronics), "電子機器")
				require.NoError(t, err)
				price, err := money.NewPrice(decimal.FromInt(1999))
				require.NoError(t, err)
				entity, err := domainproduct.New(id, domainproduct.Attributes{
					Name:                  "クリア対象商品",
					Description:           ptr.To("<p>クリア前の説明</p>"),
					Price:                 price,
					Quantity:              5,
					StockWarningThreshold: ptr.To(10),
					Status:                statusRef,
					Category:              categoryRef,
					PublishedAt:           ptr.To(base),
					ImagePath:             ptr.To("products/cleared.png"),
				})
				require.NoError(t, err)
				require.NoError(t, repo.Create(ctx, entity))

				loaded, err := repo.FindByID(ctx, id)
				require.NoError(t, err)
				// クリア前が非 NULL であることを確かめないと、最初から NULL だった場合と区別が付かない。
				require.NotNil(t, loaded.Description())
				require.NotNil(t, loaded.StockWarningThreshold())
				require.NotNil(t, loaded.PublishedAt())
				require.NotNil(t, loaded.ImagePath())

				require.NoError(t, loaded.Update(domainproduct.Attributes{
					Name:     loaded.Name(),
					Price:    loaded.Price(),
					Quantity: loaded.Quantity(),
					Status:   loaded.Status(),
					Category: loaded.Category(),
				}))

				version, err := repo.Update(ctx, loaded)
				require.NoError(t, err)
				assert.Equal(t, 2, version)

				// FindByID は公開述語を持たないため、published_at をクリアした後も読み戻せる。
				got, err := repo.FindByID(ctx, id)
				require.NoError(t, err)
				assert.Nil(t, got.Description())
				assert.Nil(t, got.StockWarningThreshold())
				assert.Nil(t, got.PublishedAt())
				assert.Nil(t, got.ImagePath())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("古いversionを持つエンティティの更新はErrVersionConflictを返しDBの行を変更しない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertProduct(ctx, t, drv, conflictID, probeKeyword+"-CONFLICT", nil, 1999, statusInStock, categoryElectronics, ptr.To(base))

				stale, err := repo.FindByID(ctx, mustParse(t, conflictID))
				require.NoError(t, err)

				// 読み込み後に他トランザクションが更新し、DB 側のバージョンだけが進んだ状態を再現する。
				_, err = drv.Exec(ctx,
					"UPDATE products SET name = $2, lock_version = lock_version + 1 WHERE id = $1",
					mustParse(t, conflictID), probeKeyword+"-BY-OTHER",
				)
				require.NoError(t, err)

				require.NoError(t, stale.Update(domainproduct.Attributes{
					Name:                  "衝突する更新",
					Description:           stale.Description(),
					Price:                 stale.Price(),
					Quantity:              stale.Quantity(),
					StockWarningThreshold: stale.StockWarningThreshold(),
					Status:                stale.Status(),
					Category:              stale.Category(),
					PublishedAt:           stale.PublishedAt(),
					ImagePath:             stale.ImagePath(),
				}))

				version, err := repo.Update(ctx, stale)
				require.ErrorIs(t, err, domainproduct.ErrVersionConflict)
				assert.Equal(t, 0, version)

				got, err := repo.FindByID(ctx, mustParse(t, conflictID))
				require.NoError(t, err)
				assert.Equal(t, probeKeyword+"-BY-OTHER", got.Name())
				assert.Equal(t, 2, got.Version())
			})
		})

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化され衝突とは区別される", func(t *testing.T) {
			t.Parallel()

			id := uuid.NewTestFromSalt(t, "update_canceled_id")
			statusRef, err := domainproduct.NewStatusRef(mustParse(t, statusInStock), "在庫あり")
			require.NoError(t, err)
			categoryRef, err := domainproduct.NewCategoryRef(mustParse(t, categoryElectronics), "電子機器")
			require.NoError(t, err)
			price, err := money.NewPrice(decimal.FromInt(100))
			require.NoError(t, err)
			entity, err := domainproduct.New(id, domainproduct.Attributes{
				Name:     "キャンセル商品",
				Price:    price,
				Quantity: 1,
				Status:   statusRef,
				Category: categoryRef,
			})
			require.NoError(t, err)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			version, err := repo.Update(ctx, entity)
			assert.Equal(t, 0, version)
			require.ErrorIs(t, err, apperror.ErrCanceled)
			require.NotErrorIs(t, err, domainproduct.ErrVersionConflict)
		})
	})
}

func Test_repository_LockByID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{tracer: lt, db: testDB}

	unpublishedID := "cccccccc-0000-4000-8000-000000000001"
	versionedID := "cccccccc-0000-4000-8000-000000000002"
	base := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非公開(published_atがNULL)の商品も取得できる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertProduct(ctx, t, drv, unpublishedID, probeKeyword+"-LOCK-UNPUB", nil, 1999, statusInStock, categoryElectronics, nil)

				got, err := repo.LockByID(ctx, mustParse(t, unpublishedID))
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, mustParse(t, unpublishedID), got.ID())
				assert.Equal(t, probeKeyword+"-LOCK-UNPUB", got.Name())
				assert.Nil(t, got.PublishedAt())
				assert.Equal(t, "在庫あり", got.Status().Name())
				assert.Equal(t, "電子機器", got.Category().Name())
			})
		})

		t.Run("DBに永続化されたversionがエンティティに反映される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertProduct(ctx, t, drv, versionedID, probeKeyword+"-LOCK-VER", nil, 1999, statusInStock, categoryElectronics, ptr.To(base))
				// 初期値(1)との区別が付くよう、DB 側のバージョンだけを進めた行を用意する。
				_, err := drv.Exec(ctx, "UPDATE products SET lock_version = 4 WHERE id = $1", mustParse(t, versionedID))
				require.NoError(t, err)

				got, err := repo.LockByID(ctx, mustParse(t, versionedID))
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, 4, got.Version())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未存在のIDはNotFoundを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.LockByID(ctx, uuid.NewTestFromSalt(t, "lock_by_id_missing"))
				assert.Nil(t, got)
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})
	})
}

func Test_repository_UpdateStock(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{tracer: lt, db: testDB}

	adjustedID := "dddddddd-0000-4000-8000-000000000001"
	conflictID := "dddddddd-0000-4000-8000-000000000002"
	untouchedID := "dddddddd-0000-4000-8000-000000000003"
	base := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("在庫を更新すると採番後のversionを返しDBの行が更新される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertProduct(ctx, t, drv, adjustedID, probeKeyword+"-STOCK", nil, 1999, statusInStock, categoryElectronics, ptr.To(base))

				entity, err := repo.LockByID(ctx, mustParse(t, adjustedID))
				require.NoError(t, err)
				before := entity.Quantity()
				require.Equal(t, 1, entity.Version())
				require.NoError(t, entity.AdjustStock(7))

				version, err := repo.UpdateStock(ctx, entity)
				require.NoError(t, err)
				assert.Equal(t, 2, version)

				got, err := repo.FindByID(ctx, mustParse(t, adjustedID))
				require.NoError(t, err)
				assert.Equal(t, before+7, got.Quantity())
				assert.Equal(t, 2, got.Version())
			})
		})

		t.Run("在庫以外の列は更新されない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertProduct(
					ctx,
					t,
					drv,
					untouchedID,
					probeKeyword+"-STOCK-KEEP",
					ptr.To("据え置きの説明"),
					1999,
					statusInStock,
					categoryElectronics,
					ptr.To(base),
				)

				entity, err := repo.LockByID(ctx, mustParse(t, untouchedID))
				require.NoError(t, err)
				require.NoError(t, entity.AdjustStock(1))
				_, err = repo.UpdateStock(ctx, entity)
				require.NoError(t, err)

				got, err := repo.FindByID(ctx, mustParse(t, untouchedID))
				require.NoError(t, err)
				assert.Equal(t, probeKeyword+"-STOCK-KEEP", got.Name())
				require.NotNil(t, got.Description())
				assert.Equal(t, "据え置きの説明", *got.Description())
				assert.Equal(t, "1999", got.Price().String())
				assert.Equal(t, mustParse(t, statusInStock), got.Status().ID())
				require.NotNil(t, got.PublishedAt())
				assert.True(t, base.Equal(*got.PublishedAt()))
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("古いversionを持つエンティティの在庫更新はErrVersionConflictを返しDBの行を変更しない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertProduct(ctx, t, drv, conflictID, probeKeyword+"-STOCK-CONFLICT", nil, 1999, statusInStock, categoryElectronics, ptr.To(base))

				stale, err := repo.FindByID(ctx, mustParse(t, conflictID))
				require.NoError(t, err)
				require.NoError(t, stale.AdjustStock(3))

				// 取得後に他トランザクションが更新し、DB 側のバージョンだけが進んだ状態を再現する。
				_, err = drv.Exec(ctx,
					"UPDATE products SET quantity = quantity + 100, lock_version = lock_version + 1 WHERE id = $1",
					mustParse(t, conflictID),
				)
				require.NoError(t, err)

				version, err := repo.UpdateStock(ctx, stale)
				require.ErrorIs(t, err, domainproduct.ErrVersionConflict)
				assert.Equal(t, 0, version)

				got, err := repo.FindByID(ctx, mustParse(t, conflictID))
				require.NoError(t, err)
				assert.Equal(t, 110, got.Quantity())
				assert.Equal(t, 2, got.Version())
			})
		})

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化され衝突とは区別される", func(t *testing.T) {
			t.Parallel()

			id := uuid.NewTestFromSalt(t, "update_stock_canceled_id")
			statusRef, err := domainproduct.NewStatusRef(mustParse(t, statusInStock), "在庫あり")
			require.NoError(t, err)
			categoryRef, err := domainproduct.NewCategoryRef(mustParse(t, categoryElectronics), "電子機器")
			require.NoError(t, err)
			price, err := money.NewPrice(decimal.FromInt(100))
			require.NoError(t, err)
			entity, err := domainproduct.New(id, domainproduct.Attributes{
				Name:     "キャンセル商品",
				Price:    price,
				Quantity: 1,
				Status:   statusRef,
				Category: categoryRef,
			})
			require.NoError(t, err)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			version, err := repo.UpdateStock(ctx, entity)
			assert.Equal(t, 0, version)
			require.ErrorIs(t, err, apperror.ErrCanceled)
			require.NotErrorIs(t, err, domainproduct.ErrVersionConflict)
		})
	})
}

//nolint:paralleltest // 両 tx から見える commit 済みの行を使うため非並列
func Test_repository_UpdateStock_concurrentRowLock(t *testing.T) {
	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	repo := &repository{tracer: lt, db: testDB}

	dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
	txm := driver.NewTransactionManager(testDB, dbCfg, logging.NewTestLogger(t), system.NewSleeper())

	// commit する専用行のため、rollback 前提の他テストが使う ID・keyword とは重ならない値にする。
	productID := mustParse(t, "9a592000-0000-4000-8000-000000000001")

	t.Cleanup(func() {
		_ = txm.Do(context.Background(), func(ctx context.Context) error {
			_, err := driver.New(ctx, testDB).Exec(ctx, "DELETE FROM products WHERE id = $1", productID)
			return err
		})
	})

	// 両 tx から見える commit 済みの在庫 10・バージョン 1 の行を用意する。
	require.NoError(t, txm.Do(context.Background(), func(ctx context.Context) error {
		_, err := driver.New(ctx, testDB).Exec(ctx, "DELETE FROM products WHERE id = $1", productID)
		require.NoError(t, err)
		insertProduct(ctx, t, driver.New(ctx, testDB), productID.String(), "STOCKCONCURRENT592",
			nil, 1999, statusInStock, categoryElectronics, nil)
		return nil
	}))

	holderLocked := make(chan struct{})
	release := make(chan struct{})
	holderDone := make(chan error, 1)

	var once sync.Once
	rel := func() { once.Do(func() { close(release) }) }
	t.Cleanup(rel) // 失敗時も保持側 goroutine をリークさせない

	go func() {
		holderDone <- txm.Do(context.Background(), func(ctx context.Context) error {
			entity, err := repo.LockByID(ctx, productID)
			if err != nil {
				return err
			}
			close(holderLocked)
			<-release
			if err = entity.AdjustStock(5); err != nil {
				return err
			}
			_, err = repo.UpdateStock(ctx, entity)
			return err
		})
	}()

	select {
	case <-holderLocked:
	case err := <-holderDone:
		require.NoError(t, err, "保持側 tx がロック取得前に失敗した")
		return
	}

	timeoutErr := txm.Do(context.Background(), func(ctx context.Context) error {
		if _, err := driver.New(ctx, testDB).Exec(ctx, "SET LOCAL lock_timeout = '50ms'"); err != nil {
			return err
		}
		_, err := repo.LockByID(ctx, productID)
		return err
	})
	require.ErrorIs(t, timeoutErr, apperror.ErrUnavailable, "ロック待ちのタイムアウトは一時的な失敗として扱う")

	contenderDone := make(chan error, 1)
	go func() {
		contenderDone <- txm.Do(context.Background(), func(ctx context.Context) error {
			entity, err := repo.LockByID(ctx, productID)
			if err != nil {
				return err
			}
			assert.Equal(t, 15, entity.Quantity(), "後続 tx は先行 tx の更新後の在庫を読む")
			if err = entity.AdjustStock(-3); err != nil {
				return err
			}
			_, err = repo.UpdateStock(ctx, entity)
			return err
		})
	}()

	rel()
	require.NoError(t, <-holderDone)
	require.NoError(t, <-contenderDone)

	got, err := repo.FindByID(context.Background(), productID)
	require.NoError(t, err)
	assert.Equal(t, 12, got.Quantity(), "両 tx の増減が失われず合成される")
	assert.Equal(t, 3, got.Version(), "在庫更新のたびにバージョンが進む")
}

func Test_intPtrToInt32Ptr(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilの場合はnilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, intPtrToInt32Ptr(nil))
		})

		t.Run("非nilの場合は値を保持したint32ポインタを返す", func(t *testing.T) {
			t.Parallel()
			got := intPtrToInt32Ptr(ptr.To(42))
			require.NotNil(t, got)
			assert.Equal(t, int32(42), *got)
		})
	})
}

func Test_rowsToProducts(t *testing.T) {
	t.Parallel()

	newRow := func(t *testing.T, salt string) productRow {
		t.Helper()
		return productRow{
			p: gen.Products{
				ID:          uuid.NewTestFromSalt(t, salt),
				Name:        "商品",
				Description: ptr.To("説明"),
				Price:       decimal.FromInt(1999),
				Quantity:    100,
				StatusID:    uuid.NewTestFromSalt(t, salt+"_status"),
				CategoryID:  uuid.NewTestFromSalt(t, salt+"_category"),
				LockVersion: 3,
			},
			statusName:   "在庫あり",
			categoryName: "電子機器",
		}
	}
	identity := func(r productRow) productRow { return r }

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("行スライスを順序を保ってエンティティ列へ変換する", func(t *testing.T) {
			t.Parallel()

			r1 := newRow(t, "rows_a")
			r2 := newRow(t, "rows_b")

			got, err := rowsToProducts([]productRow{r1, r2}, identity)

			require.NoError(t, err)
			require.Len(t, got, 2)
			assert.Equal(t, r1.p.ID, got[0].ID())
			assert.Equal(t, r2.p.ID, got[1].ID())
		})

		t.Run("空スライスは空の列を返す", func(t *testing.T) {
			t.Parallel()

			got, err := rowsToProducts([]productRow{}, identity)

			require.NoError(t, err)
			assert.Empty(t, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("変換失敗があれば先頭で打ち切り nil とエラーを返す", func(t *testing.T) {
			t.Parallel()

			invalid := newRow(t, "rows_invalid")
			invalid.statusName = "" // status 名が空だと rowToProduct が検証失敗する

			got, err := rowsToProducts([]productRow{invalid, newRow(t, "rows_valid")}, identity)

			require.Error(t, err)
			assert.Nil(t, got)
		})
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡したドライバとinfra層トレーサーを保持した実装を返す", func(t *testing.T) {
			t.Parallel()

			testDB := testkit.NewTestDB(t)
			tf := observability.NewNoopTracerFactory(t)

			actual := New(testDB, tf)

			assert.Equal(t, &repository{db: testDB, tracer: tf.Infra()}, actual)
		})
	})
}
