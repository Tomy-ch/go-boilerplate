package product

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	domainproduct "go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
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
					AfterPublishedAt: ptr.To(last.PublishedAt()), AfterID: ptr.To(last.ID()),
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
				require.Nil(t, actual)
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
				require.Nil(t, actual)
				require.ErrorIs(t, err, apperror.ErrInternal)
				require.NotErrorIs(t, err, domainproduct.ErrInvalidPrice)
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
				assert.Equal(t, 1999, got.Price())
				assert.Equal(t, 10, got.Quantity())
				assert.Nil(t, got.StockWarningThreshold())
				// DB は timestamptz をローカルタイムゾーンで返すため、格納した瞬間の一致で比較する。
				assert.True(t, base.Equal(got.PublishedAt()))
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未存在のIDはNotFoundを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.FindPublishedByID(ctx, uuid.NewTestFromSalt(t, "find_published_by_id_missing"))
				require.Nil(t, got)
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
				require.Nil(t, got)
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
				require.Nil(t, got)
				require.ErrorIs(t, err, apperror.ErrInternal)
				require.NotErrorIs(t, err, domainproduct.ErrInvalidPrice)
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

		t.Run("gen.Products をドメインエンティティへ変換する", func(t *testing.T) {
			t.Parallel()
			row := gen.Products{
				ID:                    id,
				Name:                  "商品",
				Description:           ptr.To("説明"),
				Price:                 1999,
				Quantity:              100,
				StockWarningThreshold: ptr.To(int32(10)),
				StatusID:              statusID,
				CategoryID:            categoryID,
				PublishedAt:           ptr.To(publishedAt),
			}
			got, err := rowToProduct(row)
			require.NoError(t, err)
			assert.Equal(t, id, got.ID())
			assert.Equal(t, 1999, got.Price())
			assert.Equal(t, publishedAt, got.PublishedAt())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不変条件違反(価格が負数)はデータ不整合としてErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()
			row := gen.Products{
				ID:          id,
				Name:        "商品",
				Price:       -1,
				Quantity:    100,
				StatusID:    statusID,
				CategoryID:  categoryID,
				PublishedAt: ptr.To(publishedAt),
			}
			got, err := rowToProduct(row)
			require.Nil(t, got)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}
