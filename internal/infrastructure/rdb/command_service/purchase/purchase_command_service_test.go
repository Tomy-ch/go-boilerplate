package purchase

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 既存 seed の FK 対象（products の status_id / category_id、purchases の user_id）。
const (
	seedStatusInStock  = "093170fb-83a2-4864-a2b3-53236eaf3597"
	seedCategory       = "5dd52d84-78eb-4a52-ba0b-2e11c95c2af2"
	seedUserID         = "550e8400-e29b-41d4-a716-446655440000"
	seedUnprocessedSID = "a66c996c-86b2-41d8-9bdd-9b685fb7c47d"
)

func mustParse(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	require.NoError(t, err)
	return id
}

// insertTestProduct は、FK を満たす公開商品を price / quantity 指定で挿入します。
func insertTestProduct(ctx context.Context, t *testing.T, db driver.DBTX, id uuid.UUID, price, quantity int) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO products "+
			"(id, name, description, price, quantity, stock_warning_threshold, status_id, category_id, published_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())",
		id, "purchase-cmd-test-"+id.String(), nil, price, quantity, nil, seedStatusInStock, seedCategory,
	)
	require.NoError(t, err)
}

func Test_commandService_LockProducts(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	svc := &commandService{tracer: lt, db: testDB}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定商品をロックし価格と在庫を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				pid := mustParse(t, "c1000000-0000-4000-8000-000000000001")
				insertTestProduct(ctx, t, drv, pid, 80000, 20)

				locked, err := svc.LockProducts(ctx, []uuid.UUID{pid})
				require.NoError(t, err)
				require.Len(t, locked, 1)
				assert.Equal(t, pid, locked[0].ID())
				assert.Equal(t, 80000, locked[0].Price())
				assert.Equal(t, 20, locked[0].Quantity())
			})
		})
	})
}

func Test_commandService_CreatePurchase(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	svc := &commandService{tracer: lt, db: testDB}
	userID := mustParse(t, seedUserID)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("在庫を減算し購入と明細を書き込む", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				pid := mustParse(t, "c2000000-0000-4000-8000-000000000001")
				insertTestProduct(ctx, t, drv, pid, 80000, 20)

				locked, err := svc.LockProducts(ctx, []uuid.UUID{pid})
				require.NoError(t, err)

				entity := newPurchase(t, userID, pid, 2, locked)
				require.NoError(t, svc.CreatePurchase(ctx, entity))

				var quantity int
				require.NoError(t, drv.QueryRow(ctx, "SELECT quantity FROM products WHERE id=$1", pid).Scan(&quantity))
				assert.Equal(t, 18, quantity)

				var purchaseCount, detailCount int
				require.NoError(t, drv.QueryRow(ctx, "SELECT count(*) FROM purchases WHERE id=$1", entity.ID()).Scan(&purchaseCount))
				require.NoError(t, drv.QueryRow(ctx, "SELECT count(*) FROM purchase_details WHERE purchase_id=$1", entity.ID()).Scan(&detailCount))
				assert.Equal(t, 1, purchaseCount)
				assert.Equal(t, 1, detailCount)

				// status_id は code=1（未処理）へ解決される。
				var statusID uuid.UUID
				require.NoError(t, drv.QueryRow(ctx, "SELECT status_id FROM purchases WHERE id=$1", entity.ID()).Scan(&statusID))
				assert.Equal(t, mustParse(t, seedUnprocessedSID), statusID)

				// 金額列が集約の値どおり書き込まれる（toInt32 の取り違え検知）。
				var subtotal, tax, shipping, total int
				require.NoError(t, drv.QueryRow(ctx,
					"SELECT subtotal_amount, tax_amount, shipping_fee, total_amount FROM purchases WHERE id=$1", entity.ID(),
				).Scan(&subtotal, &tax, &shipping, &total))
				assert.Equal(t, entity.SubtotalAmount(), subtotal)
				assert.Equal(t, entity.TaxAmount(), tax)
				assert.Equal(t, entity.ShippingFee(), shipping)
				assert.Equal(t, entity.TotalAmount(), total)

				var unitPrice int
				require.NoError(t, drv.QueryRow(ctx,
					"SELECT unit_price FROM purchase_details WHERE purchase_id=$1", entity.ID(),
				).Scan(&unitPrice))
				assert.Equal(t, 80000, unitPrice)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("実在庫が不足する場合はErrInsufficientStockを返し在庫を変更しない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				pid := mustParse(t, "c3000000-0000-4000-8000-000000000001")
				// 実在庫は 1 だが、古いロック値（20）で集約を作り防御 UPDATE の 0 行検知を検証する。
				insertTestProduct(ctx, t, drv, pid, 80000, 1)
				stale := []domainpurchase.LockedProduct{domainpurchase.NewLockedProduct(pid, 80000, 20)}

				entity := newPurchase(t, userID, pid, 2, stale)
				err := svc.CreatePurchase(ctx, entity)
				require.ErrorIs(t, err, domainpurchase.ErrInsufficientStock)

				var quantity int
				require.NoError(t, drv.QueryRow(ctx, "SELECT quantity FROM products WHERE id=$1", pid).Scan(&quantity))
				assert.Equal(t, 1, quantity)
			})
		})

		t.Run("2件目の明細の在庫が不足する場合はErrInsufficientStockを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				pid1 := mustParse(t, "c4000000-0000-4000-8000-000000000001")
				pid2 := mustParse(t, "c4000000-0000-4000-8000-000000000002")
				insertTestProduct(ctx, t, drv, pid1, 80000, 20)
				insertTestProduct(ctx, t, drv, pid2, 1500, 1) // 2 件目は実在庫 1

				id, err := uuid.New()
				require.NoError(t, err)
				code, err := uuid.New()
				require.NoError(t, err)
				d1, err := uuid.New()
				require.NoError(t, err)
				d2, err := uuid.New()
				require.NoError(t, err)
				// 古いロック値で両方を通し、2 件目の減算で 0 行検知させる。
				stale := []domainpurchase.LockedProduct{
					domainpurchase.NewLockedProduct(pid1, 80000, 20),
					domainpurchase.NewLockedProduct(pid2, 1500, 20),
				}
				entity, err := domainpurchase.New(id, code.String(), userID, []domainpurchase.DetailInput{
					{ID: d1, ProductID: pid1, Quantity: 1},
					{ID: d2, ProductID: pid2, Quantity: 2},
				}, stale)
				require.NoError(t, err)

				require.ErrorIs(t, svc.CreatePurchase(ctx, entity), domainpurchase.ErrInsufficientStock)
			})
		})

		t.Run("存在しないuser_idの場合はFK違反をErrInvalidArgumentへ正規化する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				pid := mustParse(t, "c5000000-0000-4000-8000-000000000001")
				insertTestProduct(ctx, t, drv, pid, 80000, 20)

				missingUser := mustParse(t, "c5000000-0000-4000-8000-0000000000ff")
				locked := []domainpurchase.LockedProduct{domainpurchase.NewLockedProduct(pid, 80000, 20)}
				entity := newPurchase(t, missingUser, pid, 1, locked)

				require.ErrorIs(t, svc.CreatePurchase(ctx, entity), apperror.ErrInvalidArgument)
			})
		})
	})
}

// newPurchase は、単一明細の購入集約を生成するテストヘルパーです。
func newPurchase(t *testing.T, userID, productID uuid.UUID, quantity int, locked []domainpurchase.LockedProduct) *domainpurchase.Purchase {
	t.Helper()
	id, err := uuid.New()
	require.NoError(t, err)
	code, err := uuid.New()
	require.NoError(t, err)
	detailID, err := uuid.New()
	require.NoError(t, err)

	entity, err := domainpurchase.New(
		id, code.String(), userID,
		[]domainpurchase.DetailInput{{ID: detailID, ProductID: productID, Quantity: quantity}},
		locked,
	)
	require.NoError(t, err)
	return entity
}
