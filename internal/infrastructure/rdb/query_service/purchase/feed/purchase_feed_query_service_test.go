package feed

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/purchase/query"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// statusCompletedID / statusCompletedName は、購入ステータスマスタ（seed 済み）の「完了」です。
	// JOIN によるステータス名解決の検証に用います。
	statusCompletedID   = "1904bf76-7d37-4288-bc15-359d2512ac91"
	statusCompletedName = "完了"
	// statusUnprocessedID / statusUnprocessedName は、購入ステータスマスタ（seed 済み）の「未処理」です。
	// ステータスごとに名称が正しく解決されることの検証に用います。
	statusUnprocessedID   = "a66c996c-86b2-41d8-9bdd-9b685fb7c47d"
	statusUnprocessedName = "未処理"
	// seedStatusInStock / seedCategory は、明細から結合する商品の FK を満たすための seed 済みマスタです。
	seedStatusInStock = "093170fb-83a2-4864-a2b3-53236eaf3597"
	seedCategory      = "5dd52d84-78eb-4a52-ba0b-2e11c95c2af2"
	// defaultProductID / defaultProductName は、要約を検証しないケースが使う既定の商品です。
	defaultProductID   = "ffffffff-3333-4000-8000-000000000001"
	defaultProductName = "ワイヤレスイヤホン"
	// statusUnprocessedCode は、purchase_statuses.code（SMALLINT）と同じ幅で「未処理」を表す値です。
	statusUnprocessedCode int16 = 1
)

// detailSeq は、insertPurchaseDetail が採番する明細 ID を一意に保つための連番です。
// テストは WithinTx で直列化されるため、単純な加算で足ります。
var detailSeq int

// insertFeedUser は、購入の FK 制約（user_id → users.id）を満たすためのユーザーを挿入するヘルパーです。
func insertFeedUser(ctx context.Context, t *testing.T, db driver.DBTX, id string) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO users "+
			"(id, first_name, last_name, email, phone, prefecture_id, city, street, postal_code, created_at, updated_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW())",
		id,
		"Feed",
		"User",
		"feed-"+id+"@example.com",
		"0000000000",
		"a03aaec4-3bd6-4bfb-8e47-2fbfa026d344", // 既存 seed の都道府県ID
		"City",
		"Street",
		"000-0000",
	)
	require.NoError(t, err)
}

// insertFeedProduct は、明細から結合される商品を挿入するヘルパーです。
func insertFeedProduct(ctx context.Context, t *testing.T, db driver.DBTX, id, name string) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO products "+
			"(id, name, description, price, quantity, stock_warning_threshold, status_id, category_id, published_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW()) ON CONFLICT (id) DO NOTHING",
		id, name, nil, 80000, 20, nil, seedStatusInStock, seedCategory,
	)
	require.NoError(t, err)
}

// insertPurchaseDetail は、購入へ明細を 1 件足すヘルパーです。明細 ID は seq の昇順で並ぶよう組み立てるため、
// 先頭商品（明細 ID 昇順の先頭）がどれになるかを呼び出し側が決められます。
func insertPurchaseDetail(ctx context.Context, t *testing.T, db driver.DBTX, purchaseID, productID string, seq int) {
	t.Helper()
	detailID := fmt.Sprintf("ffffffff-4444-4000-8000-%012d", seq)
	_, err := db.Exec(ctx,
		"INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price) VALUES ($1,$2,$3,$4,$5)",
		detailID, purchaseID, productID, 2, 80000,
	)
	require.NoError(t, err)
}

// insertPurchase は、keyset 検証用に user_id / ordered_at / id / status_id / total_amount を明示した購入を挿入します。
// 一覧は明細の要約を INNER LATERAL で解決するため、購入は必ず明細を 1 件伴います
// （明細を持たない購入は購入集約の不変条件により存在しません）。
func insertPurchase(ctx context.Context, t *testing.T, db driver.DBTX, id, userID, statusID string, total int64, orderedAt time.Time) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO purchases "+
			"(id, code, user_id, status_id, subtotal_amount, tax_amount, shipping_fee, total_amount, ordered_at, created_at, updated_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW())",
		id,
		"code-"+id,
		userID,
		statusID,
		total,
		0,
		0,
		total,
		orderedAt,
	)
	require.NoError(t, err)

	insertFeedProduct(ctx, t, db, defaultProductID, defaultProductName)
	detailSeq++
	insertPurchaseDetail(ctx, t, db, id, defaultProductID, detailSeq)
}

func Test_service_FindFeedByUserID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)

	svc := &service{tracer: lt, db: testDB}

	// seed データより必ず後ろ（新しい）に来るよう十分未来を基準にする。
	base := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)

	// 所有者と、所有権フィルタ検証用の別ユーザー。
	owner := "ffffffff-1111-4000-8000-000000000001"
	other := "ffffffff-1111-4000-8000-0000000000ff"

	// 所有者の購入: tie ペア（同一 ordered_at・id 差）+ より古い 2 件。
	// ORDER BY ordered_at DESC, id DESC のため、同一 ordered_at では id が大きい tieHigh が先に来る。
	tieHigh := "ffffffff-2222-4000-8000-000000000002"
	tieLow := "ffffffff-2222-4000-8000-000000000001"
	mid := "ffffffff-2222-4000-8000-000000000003" // base-1h
	old := "ffffffff-2222-4000-8000-000000000004" // base-2h

	mustParse := func(s string) uuid.UUID {
		id, err := uuid.Parse(s)
		require.NoError(t, err)
		return id
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("先頭ページとafterカーソルで所有者の購入がkeyset安定順に次ページを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertFeedUser(ctx, t, drv, owner)
				insertPurchase(ctx, t, drv, tieHigh, owner, statusCompletedID, 100, base)
				insertPurchase(ctx, t, drv, tieLow, owner, statusCompletedID, 200, base)
				insertPurchase(ctx, t, drv, mid, owner, statusCompletedID, 300, base.Add(-time.Hour))
				insertPurchase(ctx, t, drv, old, owner, statusCompletedID, 400, base.Add(-2*time.Hour))

				// 先頭ページ（after=nil, limit=2）: 最新の tie ペア。id DESC で tieHigh が先。
				first, err := svc.FindFeedByUserID(ctx, mustParse(owner), query.ListFeedParams{Limit: 2})
				require.NoError(t, err)
				require.Len(t, first, 2)
				assert.Equal(t, mustParse(tieHigh), first[0].ID)
				assert.Equal(t, mustParse(tieLow), first[1].ID)

				// 次ページ: 先頭ページ末尾行(tieLow)を境界に keyset を進める。
				// (ordered_at, id) < (base, tieLow) のため同一 ordered_at の tieHigh は除外され mid → old が返る。
				last := first[len(first)-1]
				second, err := svc.FindFeedByUserID(ctx, mustParse(owner), query.ListFeedParams{
					Limit:          2,
					AfterOrderedAt: &last.OrderedAt,
					AfterID:        &last.ID,
				})
				require.NoError(t, err)
				require.Len(t, second, 2)
				assert.Equal(t, mustParse(mid), second[0].ID)
				assert.Equal(t, mustParse(old), second[1].ID)
			})
		})

		t.Run("afterの境界がtieペアの先頭行の場合、同一ordered_atのもう一方が次ページ先頭に来る", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertFeedUser(ctx, t, drv, owner)
				insertPurchase(ctx, t, drv, tieHigh, owner, statusCompletedID, 100, base)
				insertPurchase(ctx, t, drv, tieLow, owner, statusCompletedID, 200, base)

				// 境界を tieHigh（同一 ordered_at の大きい id）にすると tieLow のみが残る = id タイブレークの検証。
				orderedAt := base
				id := mustParse(tieHigh)
				page, err := svc.FindFeedByUserID(ctx, mustParse(owner), query.ListFeedParams{
					Limit:          10,
					AfterOrderedAt: &orderedAt,
					AfterID:        &id,
				})
				require.NoError(t, err)
				require.Len(t, page, 1)
				assert.Equal(t, mustParse(tieLow), page[0].ID)
			})
		})

		t.Run("別ユーザーの購入は所有権フィルタで返らない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertFeedUser(ctx, t, drv, owner)
				insertFeedUser(ctx, t, drv, other)
				// 別ユーザーの購入のみ存在する状況。
				insertPurchase(ctx, t, drv, tieHigh, other, statusCompletedID, 100, base)

				got, err := svc.FindFeedByUserID(ctx, mustParse(owner), query.ListFeedParams{Limit: 10})
				require.NoError(t, err)
				assert.Empty(t, got)
			})
		})

		t.Run("先頭ページで期間外に注文された購入は返らない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertFeedUser(ctx, t, drv, owner)
				insertPurchase(ctx, t, drv, tieHigh, owner, statusCompletedID, 100, base)
				insertPurchase(ctx, t, drv, mid, owner, statusCompletedID, 300, base.Add(-24*time.Hour))

				// 半開区間 [base-1h, base+1h) は base の 1 件だけを含む。
				after := base.Add(-time.Hour)
				before := base.Add(time.Hour)
				got, err := svc.FindFeedByUserID(ctx, mustParse(owner), query.ListFeedParams{
					Limit:         10,
					OrderedAfter:  &after,
					OrderedBefore: &before,
				})
				require.NoError(t, err)
				require.Len(t, got, 1)
				assert.Equal(t, mustParse(tieHigh), got[0].ID)
			})
		})

		t.Run("afterページでも期間の絞り込みが効く", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertFeedUser(ctx, t, drv, owner)
				insertPurchase(ctx, t, drv, tieHigh, owner, statusCompletedID, 100, base)
				insertPurchase(ctx, t, drv, mid, owner, statusCompletedID, 300, base.Add(-time.Hour))
				insertPurchase(ctx, t, drv, old, owner, statusCompletedID, 400, base.Add(-24*time.Hour))

				// keyset 境界は tieHigh。期間を [base-2h, base+1h) に絞ると old は範囲外で mid だけが残る。
				orderedAt := base
				id := mustParse(tieHigh)
				after := base.Add(-2 * time.Hour)
				before := base.Add(time.Hour)
				got, err := svc.FindFeedByUserID(ctx, mustParse(owner), query.ListFeedParams{
					Limit:          10,
					AfterOrderedAt: &orderedAt,
					AfterID:        &id,
					OrderedAfter:   &after,
					OrderedBefore:  &before,
				})
				require.NoError(t, err)
				require.Len(t, got, 1)
				assert.Equal(t, mustParse(mid), got[0].ID)
			})
		})

		t.Run("開始日だけを指定した場合は絞り込まない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertFeedUser(ctx, t, drv, owner)
				insertPurchase(ctx, t, drv, tieHigh, owner, statusCompletedID, 100, base)
				insertPurchase(ctx, t, drv, old, owner, statusCompletedID, 400, base.Add(-24*time.Hour))

				// 片側だけの指定は絞り込みなしとして扱う契約。境界が NULL のまま述語へ入ると全件が消える。
				after := base.Add(-time.Hour)
				got, err := svc.FindFeedByUserID(ctx, mustParse(owner), query.ListFeedParams{
					Limit:        10,
					OrderedAfter: &after,
				})
				require.NoError(t, err)
				assert.Len(t, got, 2)
			})
		})

		t.Run("終了日だけを指定した場合も絞り込まない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertFeedUser(ctx, t, drv, owner)
				insertPurchase(ctx, t, drv, tieHigh, owner, statusCompletedID, 100, base)
				insertPurchase(ctx, t, drv, old, owner, statusCompletedID, 400, base.Add(-24*time.Hour))

				// 開始日側と対称であることを固定する。片側判定が非対称に壊れてもここが赤くなる。
				before := base.Add(-time.Hour)
				got, err := svc.FindFeedByUserID(ctx, mustParse(owner), query.ListFeedParams{
					Limit:         10,
					OrderedBefore: &before,
				})
				require.NoError(t, err)
				assert.Len(t, got, 2)
			})
		})

		t.Run("statusが購入ごとにマスタ名称で解決され金額とコードも一致して返る", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertFeedUser(ctx, t, drv, owner)
				// 異なるステータスの購入を 2 件（新: 完了 / 旧: 未処理）挿入し、それぞれ名称解決されることを検証する。
				insertPurchase(ctx, t, drv, tieHigh, owner, statusCompletedID, 176500, base)
				insertPurchase(ctx, t, drv, mid, owner, statusUnprocessedID, 500, base.Add(-time.Hour))

				got, err := svc.FindFeedByUserID(ctx, mustParse(owner), query.ListFeedParams{Limit: 10})
				require.NoError(t, err)
				require.Len(t, got, 2)
				// ordered_at 降順: 完了(base) → 未処理(base-1h)。ステータス ID / 名称ともに JOIN で解決される。
				assert.Equal(t, mustParse(statusCompletedID), got[0].StatusID)
				assert.Equal(t, statusCompletedName, got[0].StatusName)
				assert.Equal(t, domainpurchase.StatusCompleted.Code(), got[0].StatusCode)
				assert.Equal(t, 176500, got[0].TotalAmount)
				assert.Equal(t, "code-"+tieHigh, got[0].Code)
				assert.Equal(t, mustParse(statusUnprocessedID), got[1].StatusID)
				assert.Equal(t, statusUnprocessedName, got[1].StatusName)
				assert.Equal(t, domainpurchase.StatusUnprocessed.Code(), got[1].StatusCode)
			})
		})

		t.Run("明細が1件の購入は先頭商品名を返しitemCountが1になる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertFeedUser(ctx, t, drv, owner)
				// insertPurchase が既定商品の明細を 1 件だけ伴わせる。
				insertPurchase(ctx, t, drv, tieHigh, owner, statusCompletedID, 176500, base)

				got, err := svc.FindFeedByUserID(ctx, mustParse(owner), query.ListFeedParams{Limit: 10})
				require.NoError(t, err)
				require.Len(t, got, 1)
				assert.Equal(t, defaultProductName, got[0].FirstItemName)
				assert.Equal(t, 1, got[0].ItemCount)
			})
		})

		t.Run("明細が複数の購入は明細IDの先頭の商品名と行数を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertFeedUser(ctx, t, drv, owner)
				insertPurchase(ctx, t, drv, tieHigh, owner, statusCompletedID, 176500, base)

				// 既定明細（先頭）の後ろへ 2 件足す。先頭商品名は既定商品のままで、点数だけが 3 になる。
				secondProduct := "ffffffff-3333-4000-8000-000000000002"
				thirdProduct := "ffffffff-3333-4000-8000-000000000003"
				insertFeedProduct(ctx, t, drv, secondProduct, "モバイルバッテリー")
				insertFeedProduct(ctx, t, drv, thirdProduct, "USB-C ケーブル")
				detailSeq++
				insertPurchaseDetail(ctx, t, drv, tieHigh, secondProduct, detailSeq)
				detailSeq++
				insertPurchaseDetail(ctx, t, drv, tieHigh, thirdProduct, detailSeq)

				got, err := svc.FindFeedByUserID(ctx, mustParse(owner), query.ListFeedParams{Limit: 10})
				require.NoError(t, err)
				require.Len(t, got, 1)
				assert.Equal(t, defaultProductName, got[0].FirstItemName)
				assert.Equal(t, 3, got[0].ItemCount)
			})
		})

		t.Run("複数購入の要約が購入ごとに解決される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertFeedUser(ctx, t, drv, owner)
				insertPurchase(ctx, t, drv, tieHigh, owner, statusCompletedID, 100, base)
				insertPurchase(ctx, t, drv, mid, owner, statusCompletedID, 300, base.Add(-time.Hour))

				// 新しい方にだけ明細を足し、要約が購入をまたいで混ざらないことを確かめる。
				extraProduct := "ffffffff-3333-4000-8000-000000000004"
				insertFeedProduct(ctx, t, drv, extraProduct, "スマホスタンド")
				detailSeq++
				insertPurchaseDetail(ctx, t, drv, tieHigh, extraProduct, detailSeq)

				got, err := svc.FindFeedByUserID(ctx, mustParse(owner), query.ListFeedParams{Limit: 10})
				require.NoError(t, err)
				require.Len(t, got, 2)
				assert.Equal(t, 2, got[0].ItemCount)
				assert.Equal(t, 1, got[1].ItemCount)
				assert.Equal(t, defaultProductName, got[0].FirstItemName)
				assert.Equal(t, defaultProductName, got[1].FirstItemName)
			})
		})

		t.Run("購入ゼロのユーザーは空一覧になる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertFeedUser(ctx, t, drv, owner)

				got, err := svc.FindFeedByUserID(ctx, mustParse(owner), query.ListFeedParams{Limit: 10})
				require.NoError(t, err)
				assert.Empty(t, got)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("先頭ページでlimitが負数の場合、ErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				// 負数 LIMIT は PostgreSQL の 2201W（map 未定義）となり ErrInternal へ写像される。
				got, err := svc.FindFeedByUserID(ctx, mustParse(owner), query.ListFeedParams{Limit: -1})
				assert.Nil(t, got)
				require.ErrorIs(t, err, apperror.ErrInternal)
			})
		})

		t.Run("afterカーソル指定でlimitが負数の場合、ErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				orderedAt := base
				id := mustParse(tieHigh)
				got, err := svc.FindFeedByUserID(ctx, mustParse(owner), query.ListFeedParams{
					Limit:          -1,
					AfterOrderedAt: &orderedAt,
					AfterID:        &id,
				})
				assert.Nil(t, got)
				require.ErrorIs(t, err, apperror.ErrInternal)
			})
		})
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("依存を注入したクエリサービス実装を生成する", func(t *testing.T) {
			t.Parallel()

			testDB := testkit.NewTestDB(t)
			tf := observability.NewNoopTracerFactory(t)

			svc, ok := New(testDB, tf).(*service)
			require.True(t, ok)
			assert.Equal(t, testDB, svc.db)
			assert.NotNil(t, svc.tracer)
		})
	})
}

func Test_toFeedReadModel(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入履歴フィードの行を読み取りモデルの各フィールドへ写像する", func(t *testing.T) {
			t.Parallel()

			id := mustParseUUID(t, "e3000000-0000-4000-8000-000000000001")
			statusID := mustParseUUID(t, "e3000000-0000-4000-8000-0000000000a1")
			orderedAt := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)

			item := toFeedReadModel(feedRow{
				ID:            id,
				Code:          "feed-code-1",
				TotalAmount:   176500,
				OrderedAt:     orderedAt,
				StatusID:      statusID,
				StatusCode:    statusUnprocessedCode,
				StatusName:    statusUnprocessedName,
				FirstItemName: "ワイヤレスイヤホン",
				ItemCount:     3,
			})

			assert.Equal(t, query.PurchaseFeedReadModel{
				Code:          "feed-code-1",
				TotalAmount:   176500,
				StatusID:      statusID,
				StatusCode:    domainpurchase.StatusUnprocessed.Code(),
				StatusName:    statusUnprocessedName,
				FirstItemName: "ワイヤレスイヤホン",
				ItemCount:     3,
				OrderedAt:     orderedAt,
				ID:            id,
			}, item)
		})
	})
}

// mustParseUUID は、テスト定数の UUID 文字列を解決するヘルパーです。
func mustParseUUID(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	require.NoError(t, err)
	return id
}
