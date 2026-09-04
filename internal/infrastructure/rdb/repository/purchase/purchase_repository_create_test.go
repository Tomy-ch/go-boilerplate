package purchase

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/lexicon/money"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// insertProductRow は、products へ 1 行挿入します。列構成を持つのはこの 1 箇所だけです。
func insertProductRow(ctx context.Context, t *testing.T, db driver.DBTX, id uuid.UUID, name string, quantity int) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO products (id, name, description, price, quantity, stock_warning_threshold, status_id, category_id, published_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())",
		id, name, nil, 80000, quantity, nil, seedStatusInStock, seedCategory,
	)
	require.NoError(t, err)
}

// insertPurchaseRow は、purchases へ 1 行挿入します。列構成を持つのはこの 1 箇所だけです。
func insertPurchaseRow(
	ctx context.Context, t *testing.T, db driver.DBTX,
	id uuid.UUID, code string, userID uuid.UUID, subtotal, tax, shipping, total int,
) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO purchases (id, code, user_id, status_id, subtotal_amount, tax_amount, shipping_fee, total_amount) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
		id, code, userID, mustParse(t, seedUnprocessedSID), subtotal, tax, shipping, total,
	)
	require.NoError(t, err)
}

// insertPurchasableProduct は、購入の FK 対象となる商品を挿入し、その ID と在庫を返します。
func insertPurchasableProduct(ctx context.Context, t *testing.T, db driver.DBTX, seed string) (uuid.UUID, int) {
	t.Helper()
	productID := mustParse(t, seed)
	const quantity = 20
	insertProductRow(ctx, t, db, productID, "purchase-create-test-"+seed, quantity)

	return productID, quantity
}

// insertPurchaseOnly は、明細を伴わずに購入本体だけを挿入します（明細登録の単体検証で使います）。
func insertPurchaseOnly(ctx context.Context, t *testing.T, db driver.DBTX, p *domainpurchase.Purchase) {
	t.Helper()
	insertPurchaseRow(
		ctx, t, db, p.ID(), p.Code(), p.UserID(),
		p.SubtotalAmount(), p.TaxAmount(), p.ShippingFee(), p.TotalAmount(),
	)
}

// newPurchaseToCreate は、Create へ渡す購入集約を明細 1 件つきで組み立てます。
func newPurchaseToCreate(t *testing.T, code string, productID uuid.UUID, stock int) *domainpurchase.Purchase {
	t.Helper()

	purchaseID, err := uuid.New()
	require.NoError(t, err)
	detailID, err := uuid.New()
	require.NoError(t, err)
	price, err := money.NewPrice(decimaltestkit.MustParse(t, "800"))
	require.NoError(t, err)

	entity, err := domainpurchase.New(
		purchaseID,
		code,
		mustParse(t, seedUserID),
		[]domainpurchase.DetailInput{{ID: detailID, ProductID: productID, Quantity: 2}},
		[]domainpurchase.LockedProduct{domainpurchase.NewLockedProduct(productID, price, stock)},
	)
	require.NoError(t, err)

	return entity
}

func Test_repository_Create(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入本体と明細を登録し、再読込で集約として復元できる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID, stock := insertPurchasableProduct(ctx, t, drv, "f4000000-0000-4000-8000-000000000001")
				entity := newPurchaseToCreate(t, "create-code-1", productID, stock)

				require.NoError(t, repo.Create(ctx, entity))

				reread, err := repo.FindByID(ctx, entity.ID())
				require.NoError(t, err)
				assert.Equal(t, entity.Code(), reread.Code())
				assert.Equal(t, entity.TotalAmount(), reread.TotalAmount())
				require.Len(t, reread.Details(), 1)
				assert.Equal(t, productID, reread.Details()[0].ProductID())
				assert.Equal(t, 2, reread.Details()[0].Quantity())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同じ購入コードを二重に登録すると衝突として返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID, stock := insertPurchasableProduct(ctx, t, drv, "f4000000-0000-4000-8000-000000000002")
				require.NoError(t, repo.Create(ctx, newPurchaseToCreate(t, "create-code-dup", productID, stock)))

				// 同じ code を持つ別の購入。二重登録はここで初めて失敗する。
				err := repo.Create(ctx, newPurchaseToCreate(t, "create-code-dup", productID, stock))

				require.ErrorIs(t, err, apperror.ErrConflict)
			})
		})

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			err := repo.Create(ctx, newPurchaseToCreate(
				t, "create-code-canceled", mustParse(t, "f4000000-0000-4000-8000-000000000003"), 20,
			))

			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

func Test_repository_insertDetails(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("集約が保持する明細を購入行へ紐づけて登録する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID, stock := insertPurchasableProduct(ctx, t, drv, "f5000000-0000-4000-8000-000000000001")
				entity := newPurchaseToCreate(t, "insert-details-code", productID, stock)
				// 購入本体だけを先に入れ、明細の登録は insertDetails 単体で行う。
				insertPurchaseOnly(ctx, t, drv, entity)

				require.NoError(t, repo.insertDetails(ctx, gen.New(drv), entity))

				rows, err := gen.New(drv).ListPurchaseDetailsByPurchaseID(ctx, entity.ID())
				require.NoError(t, err)
				require.Len(t, rows, 1)
				assert.Equal(t, entity.ID(), rows[0].PurchaseDetails.PurchaseID)
				assert.Equal(t, productID, rows[0].PurchaseDetails.ProductID)
				assert.Equal(t, int32(2), rows[0].PurchaseDetails.Quantity)
				assert.Equal(t, entity.Details()[0].ID(), rows[0].PurchaseDetails.ID)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入行が存在しない明細は外部キー違反として返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID, stock := insertPurchasableProduct(ctx, t, drv, "f5000000-0000-4000-8000-000000000002")
				orphan := newPurchaseToCreate(t, "insert-details-orphan", productID, stock)

				// 購入本体を登録せずに明細だけ入れる。
				err := repo.insertDetails(ctx, gen.New(drv), orphan)

				require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			})
		})

		t.Run("同じ明細IDを二重に登録すると衝突として返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID, stock := insertPurchasableProduct(ctx, t, drv, "f5000000-0000-4000-8000-000000000003")
				entity := newPurchaseToCreate(t, "insert-details-dup", productID, stock)
				require.NoError(t, repo.Create(ctx, entity))

				err := repo.insertDetails(ctx, gen.New(drv), entity)

				require.ErrorIs(t, err, apperror.ErrConflict)
			})
		})
	})
}
