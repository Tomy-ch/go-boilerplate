package purchase

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
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
)

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
		"000-000-0000",
		"a03aaec4-3bd6-4bfb-8e47-2fbfa026d344", // 既存 seed の都道府県ID
		"City",
		"Street",
		"000-0000",
	)
	require.NoError(t, err)
}

// insertPurchase は、keyset 検証用に user_id / ordered_at / id / status_id / total_amount を明示した購入を挿入します。
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
}

func Test_repository_FindFeedByUserID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{tracer: lt, db: testDB}

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
				first, err := repo.FindFeedByUserID(ctx, mustParse(owner), purchase.ListFeedParams{Limit: 2})
				require.NoError(t, err)
				require.Len(t, first, 2)
				assert.Equal(t, mustParse(tieHigh), first[0].ID)
				assert.Equal(t, mustParse(tieLow), first[1].ID)

				// 次ページ: 先頭ページ末尾行(tieLow)を境界に keyset を進める。
				// (ordered_at, id) < (base, tieLow) のため同一 ordered_at の tieHigh は除外され mid → old が返る。
				last := first[len(first)-1]
				second, err := repo.FindFeedByUserID(ctx, mustParse(owner), purchase.ListFeedParams{
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
				page, err := repo.FindFeedByUserID(ctx, mustParse(owner), purchase.ListFeedParams{
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

				got, err := repo.FindFeedByUserID(ctx, mustParse(owner), purchase.ListFeedParams{Limit: 10})
				require.NoError(t, err)
				assert.Empty(t, got)
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

				got, err := repo.FindFeedByUserID(ctx, mustParse(owner), purchase.ListFeedParams{Limit: 10})
				require.NoError(t, err)
				require.Len(t, got, 2)
				// ordered_at 降順: 完了(base) → 未処理(base-1h)。ステータス ID / 名称ともに JOIN で解決される。
				assert.Equal(t, mustParse(statusCompletedID), got[0].StatusID)
				assert.Equal(t, statusCompletedName, got[0].StatusName)
				assert.Equal(t, 176500, got[0].TotalAmount)
				assert.Equal(t, "code-"+tieHigh, got[0].Code)
				assert.Equal(t, mustParse(statusUnprocessedID), got[1].StatusID)
				assert.Equal(t, statusUnprocessedName, got[1].StatusName)
			})
		})

		t.Run("購入ゼロのユーザーは空一覧になる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertFeedUser(ctx, t, drv, owner)

				got, err := repo.FindFeedByUserID(ctx, mustParse(owner), purchase.ListFeedParams{Limit: 10})
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
				got, err := repo.FindFeedByUserID(ctx, mustParse(owner), purchase.ListFeedParams{Limit: -1})
				require.Nil(t, got)
				require.ErrorIs(t, err, apperror.ErrInternal)
			})
		})

		t.Run("afterカーソル指定でlimitが負数の場合、ErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				orderedAt := base
				id := mustParse(tieHigh)
				got, err := repo.FindFeedByUserID(ctx, mustParse(owner), purchase.ListFeedParams{
					Limit:          -1,
					AfterOrderedAt: &orderedAt,
					AfterID:        &id,
				})
				require.Nil(t, got)
				require.ErrorIs(t, err, apperror.ErrInternal)
			})
		})
	})
}
