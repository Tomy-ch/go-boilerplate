package ranking

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	clocktestkit "go-boilerplate/internal/usecase/boundary/clock/testkit"
	"go-boilerplate/internal/usecase/product/ranking/query"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 既存 seed の FK 対象（products の status_id / category_id、purchases の user_id / status_id）。
const (
	seedStatusInStock  = "093170fb-83a2-4864-a2b3-53236eaf3597"
	seedCategory       = "5dd52d84-78eb-4a52-ba0b-2e11c95c2af2"
	seedUserID         = "550e8400-e29b-41d4-a716-446655440000"
	seedUnprocessedSID = "a66c996c-86b2-41d8-9bdd-9b685fb7c47d"
)

// canceledContext は、キャンセル済みのコンテキストを返します。
func canceledContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	return ctx
}

func mustParse(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	require.NoError(t, err)
	return id
}

// insertProduct は、FK を満たす公開商品を price（十進文字列）と name 指定で挿入します。
func insertProduct(ctx context.Context, t *testing.T, db driver.DBTX, id uuid.UUID, price, name string) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO products "+
			"(id, name, description, price, quantity, stock_warning_threshold, status_id, category_id, published_at) "+
			"VALUES ($1,$2,$3,$4::numeric,$5,$6,$7,$8,NOW())",
		id, name, nil, price, 100, nil, seedStatusInStock, seedCategory,
	)
	require.NoError(t, err)
}

// insertUnpublishedProduct は、published_at が NULL（非公開）の商品を挿入します。
func insertUnpublishedProduct(ctx context.Context, t *testing.T, db driver.DBTX, id uuid.UUID, price, name string) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO products "+
			"(id, name, description, price, quantity, stock_warning_threshold, status_id, category_id, published_at) "+
			"VALUES ($1,$2,$3,$4::numeric,$5,$6,$7,$8,NULL)",
		id, name, nil, price, 100, nil, seedStatusInStock, seedCategory,
	)
	require.NoError(t, err)
}

// insertPurchase は、集計対象となる購入を注文日時とキャンセル日時（nil 可）指定で挿入します。
func insertPurchase(ctx context.Context, t *testing.T, db driver.DBTX, id uuid.UUID, orderedAt time.Time, canceledAt *time.Time) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO purchases "+
			"(id, code, user_id, status_id, subtotal_amount, tax_amount, shipping_fee, total_amount, ordered_at, canceled_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)",
		id, "ranking-test-"+id.String(), mustParse(t, seedUserID), mustParse(t, seedUnprocessedSID),
		1000, 100, 0, 1100, orderedAt, canceledAt,
	)
	require.NoError(t, err)
}

// insertDetail は、購入明細を数量指定で挿入します。
func insertDetail(ctx context.Context, t *testing.T, db driver.DBTX, id, purchaseID, productID uuid.UUID, quantity int) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price) VALUES ($1,$2,$3,$4,$5::numeric)",
		id, purchaseID, productID, quantity, "10",
	)
	require.NoError(t, err)
}

func Test_service_ListRanking(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	now := time.Now()
	svc := &service{db: testDB, clk: clocktestkit.NewMockClock(t, now), tracer: lt}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("複数購入にまたがる同一商品の数量を合算し販売数量の降順でname/priceと共に返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productA := mustParse(t, "a1000000-0000-4000-8000-000000000001")
				productB := mustParse(t, "a1000000-0000-4000-8000-000000000002")
				insertProduct(ctx, t, drv, productA, "19.99", "商品A")
				insertProduct(ctx, t, drv, productB, "5.50", "商品B")

				purchase1 := mustParse(t, "a1000000-0000-4000-8000-0000000000f1")
				purchase2 := mustParse(t, "a1000000-0000-4000-8000-0000000000f2")
				insertPurchase(ctx, t, drv, purchase1, now, nil)
				insertPurchase(ctx, t, drv, purchase2, now, nil)
				insertDetail(ctx, t, drv, mustParse(t, "a1000000-0000-4000-8000-0000000000d1"), purchase1, productA, 3)
				insertDetail(ctx, t, drv, mustParse(t, "a1000000-0000-4000-8000-0000000000d2"), purchase2, productA, 5)
				insertDetail(ctx, t, drv, mustParse(t, "a1000000-0000-4000-8000-0000000000d3"), purchase1, productB, 4)

				got, err := svc.ListRanking(ctx, query.RankingQueryParams{Period: query.PeriodAll, Limit: 10})
				require.NoError(t, err)

				require.Len(t, got, 2)
				assert.Equal(t, productA, got[0].ProductID)
				assert.Equal(t, "商品A", got[0].Name)
				assert.Equal(t, "19.99", got[0].Price.String())
				assert.Equal(t, int64(8), got[0].SoldQuantity)
				assert.Equal(t, productB, got[1].ProductID)
				assert.Equal(t, "5.5", got[1].Price.String())
				assert.Equal(t, int64(4), got[1].SoldQuantity)
			})
		})

		t.Run("キャンセル済みの購入は除外し未払いの購入は含める", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productC := mustParse(t, "b1000000-0000-4000-8000-000000000001")
				productD := mustParse(t, "b1000000-0000-4000-8000-000000000002")
				insertProduct(ctx, t, drv, productC, "10", "商品C")
				insertProduct(ctx, t, drv, productD, "10", "商品D")

				canceledAt := now
				canceledPurchase := mustParse(t, "b1000000-0000-4000-8000-0000000000f1")
				unpaidPurchase := mustParse(t, "b1000000-0000-4000-8000-0000000000f2")
				normalPurchase := mustParse(t, "b1000000-0000-4000-8000-0000000000f3")
				insertPurchase(ctx, t, drv, canceledPurchase, now, &canceledAt)
				insertPurchase(ctx, t, drv, unpaidPurchase, now, nil)
				insertPurchase(ctx, t, drv, normalPurchase, now, nil)
				insertDetail(ctx, t, drv, mustParse(t, "b1000000-0000-4000-8000-0000000000d1"), canceledPurchase, productC, 100)
				insertDetail(ctx, t, drv, mustParse(t, "b1000000-0000-4000-8000-0000000000d2"), unpaidPurchase, productC, 2)
				insertDetail(ctx, t, drv, mustParse(t, "b1000000-0000-4000-8000-0000000000d3"), normalPurchase, productD, 1)

				got, err := svc.ListRanking(ctx, query.RankingQueryParams{Period: query.PeriodAll, Limit: 10})
				require.NoError(t, err)

				require.Len(t, got, 2)
				assert.Equal(t, productC, got[0].ProductID)
				assert.Equal(t, int64(2), got[0].SoldQuantity)
				assert.Equal(t, productD, got[1].ProductID)
				assert.Equal(t, int64(1), got[1].SoldQuantity)
			})
		})

		t.Run("同一販売数量は商品IDの昇順で安定的に並ぶ", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productLow := mustParse(t, "c1000000-0000-4000-8000-000000000001")
				productHigh := mustParse(t, "c1000000-0000-4000-8000-000000000002")
				insertProduct(ctx, t, drv, productLow, "10", "商品Low")
				insertProduct(ctx, t, drv, productHigh, "10", "商品High")

				purchase := mustParse(t, "c1000000-0000-4000-8000-0000000000f1")
				insertPurchase(ctx, t, drv, purchase, now, nil)
				insertDetail(ctx, t, drv, mustParse(t, "c1000000-0000-4000-8000-0000000000d1"), purchase, productLow, 5)
				insertDetail(ctx, t, drv, mustParse(t, "c1000000-0000-4000-8000-0000000000d2"), purchase, productHigh, 5)

				got, err := svc.ListRanking(ctx, query.RankingQueryParams{Period: query.PeriodAll, Limit: 10})
				require.NoError(t, err)

				require.Len(t, got, 2)
				assert.Equal(t, productLow, got[0].ProductID)
				assert.Equal(t, productHigh, got[1].ProductID)
			})
		})

		t.Run("period=30dは注文日時が直近30日以内の購入のみ集計する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				product := mustParse(t, "d1000000-0000-4000-8000-000000000001")
				insertProduct(ctx, t, drv, product, "10", "商品E")

				recentPurchase := mustParse(t, "d1000000-0000-4000-8000-0000000000f1")
				oldPurchase := mustParse(t, "d1000000-0000-4000-8000-0000000000f2")
				insertPurchase(ctx, t, drv, recentPurchase, now, nil)
				insertPurchase(ctx, t, drv, oldPurchase, now.Add(-60*24*time.Hour), nil)
				insertDetail(ctx, t, drv, mustParse(t, "d1000000-0000-4000-8000-0000000000d1"), recentPurchase, product, 7)
				insertDetail(ctx, t, drv, mustParse(t, "d1000000-0000-4000-8000-0000000000d2"), oldPurchase, product, 50)

				gotAll, err := svc.ListRanking(ctx, query.RankingQueryParams{Period: query.PeriodAll, Limit: 10})
				require.NoError(t, err)
				require.Len(t, gotAll, 1)
				assert.Equal(t, int64(57), gotAll[0].SoldQuantity)

				got30d, err := svc.ListRanking(ctx, query.RankingQueryParams{Period: query.Period30d, Limit: 10})
				require.NoError(t, err)
				require.Len(t, got30d, 1)
				assert.Equal(t, int64(7), got30d[0].SoldQuantity)
			})
		})

		t.Run("period=30dは境界ちょうどの注文を含み境界より古い注文を除外する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				product := mustParse(t, "aa100000-0000-4000-8000-000000000001")
				insertProduct(ctx, t, drv, product, "10", "境界商品")

				// サービスは固定 clock の now を基準に境界を算出するため、同じ now から境界を厳密に再現する。
				boundary := now.Add(-rankingWindow30d)
				insidePurchase := mustParse(t, "aa100000-0000-4000-8000-0000000000f1")
				outsidePurchase := mustParse(t, "aa100000-0000-4000-8000-0000000000f2")
				insertPurchase(ctx, t, drv, insidePurchase, boundary, nil)
				insertPurchase(ctx, t, drv, outsidePurchase, boundary.Add(-time.Second), nil)
				insertDetail(ctx, t, drv, mustParse(t, "aa100000-0000-4000-8000-0000000000d1"), insidePurchase, product, 3)
				insertDetail(ctx, t, drv, mustParse(t, "aa100000-0000-4000-8000-0000000000d2"), outsidePurchase, product, 40)

				got, err := svc.ListRanking(ctx, query.RankingQueryParams{Period: query.Period30d, Limit: 10})
				require.NoError(t, err)

				require.Len(t, got, 1)
				assert.Equal(t, int64(3), got[0].SoldQuantity)
			})
		})

		t.Run("limitで販売数量上位N件に絞る", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productTop := mustParse(t, "e1000000-0000-4000-8000-000000000001")
				productMid := mustParse(t, "e1000000-0000-4000-8000-000000000002")
				productLow := mustParse(t, "e1000000-0000-4000-8000-000000000003")
				insertProduct(ctx, t, drv, productTop, "10", "商品Top")
				insertProduct(ctx, t, drv, productMid, "10", "商品Mid")
				insertProduct(ctx, t, drv, productLow, "10", "商品Low")

				purchase := mustParse(t, "e1000000-0000-4000-8000-0000000000f1")
				insertPurchase(ctx, t, drv, purchase, now, nil)
				insertDetail(ctx, t, drv, mustParse(t, "e1000000-0000-4000-8000-0000000000d1"), purchase, productTop, 9)
				insertDetail(ctx, t, drv, mustParse(t, "e1000000-0000-4000-8000-0000000000d2"), purchase, productMid, 5)
				insertDetail(ctx, t, drv, mustParse(t, "e1000000-0000-4000-8000-0000000000d3"), purchase, productLow, 1)

				got, err := svc.ListRanking(ctx, query.RankingQueryParams{Period: query.PeriodAll, Limit: 2})
				require.NoError(t, err)

				require.Len(t, got, 2)
				assert.Equal(t, productTop, got[0].ProductID)
				assert.Equal(t, productMid, got[1].ProductID)
			})
		})

		t.Run("非公開(published_atがNULL)の商品は購入済みでもランキングに出ない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				publishedProduct := mustParse(t, "f1000000-0000-4000-8000-000000000001")
				unpublishedProduct := mustParse(t, "f1000000-0000-4000-8000-000000000002")
				insertProduct(ctx, t, drv, publishedProduct, "10", "公開商品")
				insertUnpublishedProduct(ctx, t, drv, unpublishedProduct, "10", "非公開商品")

				purchase := mustParse(t, "f1000000-0000-4000-8000-0000000000f1")
				insertPurchase(ctx, t, drv, purchase, now, nil)
				insertDetail(ctx, t, drv, mustParse(t, "f1000000-0000-4000-8000-0000000000d1"), purchase, publishedProduct, 1)
				insertDetail(ctx, t, drv, mustParse(t, "f1000000-0000-4000-8000-0000000000d2"), purchase, unpublishedProduct, 99)

				got, err := svc.ListRanking(ctx, query.RankingQueryParams{Period: query.PeriodAll, Limit: 10})
				require.NoError(t, err)

				require.Len(t, got, 1)
				assert.Equal(t, publishedProduct, got[0].ProductID)
				assert.Equal(t, int64(1), got[0].SoldQuantity)
			})
		})

		t.Run("集計対象の購入が無い場合は空を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				got, err := svc.ListRanking(ctx, query.RankingQueryParams{Period: query.PeriodAll, Limit: 10})
				require.NoError(t, err)
				assert.Empty(t, got)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			got, err := svc.ListRanking(canceledContext(t), query.RankingQueryParams{Period: query.PeriodAll, Limit: 10})
			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.Nil(t, got)
		})
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	tf := observability.NewNoopTracerFactory(t)
	clk := clocktestkit.NewMockClock(t, time.Now())
	expected := &service{
		db:     testDB,
		clk:    clk,
		tracer: tf.Infra(),
	}
	actual := New(testDB, clk, tf)
	assert.Equal(t, expected, actual)
}

func Test_resolvePeriod(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 25, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全期間は期間フィルタ無効かつ境界時刻nilを返す", func(t *testing.T) {
			t.Parallel()

			filter, after := resolvePeriod(query.PeriodAll, now)
			assert.False(t, filter)
			assert.Nil(t, after)
		})

		t.Run("直近30日は期間フィルタ有効かつnowから30日前の境界を返す", func(t *testing.T) {
			t.Parallel()

			filter, after := resolvePeriod(query.Period30d, now)
			assert.True(t, filter)
			require.NotNil(t, after)
			assert.Equal(t, now.Add(-30*24*time.Hour), *after)
		})
	})
}
