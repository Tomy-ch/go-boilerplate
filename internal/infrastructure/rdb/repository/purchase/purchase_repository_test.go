package purchase

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 既存 seed の FK 対象。
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

// insertPurchaseWithDetail は、FK を満たす商品・購入・明細を挿入し、購入 ID / 商品 ID を返します。
func insertPurchaseWithDetail(ctx context.Context, t *testing.T, db driver.DBTX, seed string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	productID := mustParse(t, seed)
	_, err := db.Exec(ctx,
		"INSERT INTO products (id, name, description, price, quantity, stock_warning_threshold, status_id, category_id, published_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())",
		productID, "purchase-repo-test-"+seed, nil, 80000, 20, nil, seedStatusInStock, seedCategory,
	)
	require.NoError(t, err)

	purchaseID, err := uuid.New()
	require.NoError(t, err)
	_, err = db.Exec(ctx,
		"INSERT INTO purchases (id, code, user_id, status_id, subtotal_amount, tax_amount, shipping_fee, total_amount) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
		purchaseID, "repo-code-"+seed, mustParse(t, seedUserID), mustParse(t, seedUnprocessedSID), 160000, 16000, 500, 176500,
	)
	require.NoError(t, err)

	detailID, err := uuid.New()
	require.NoError(t, err)
	_, err = db.Exec(ctx,
		"INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price) VALUES ($1,$2,$3,$4,$5)",
		detailID, purchaseID, productID, 2, 80000,
	)
	require.NoError(t, err)
	return purchaseID, productID
}

func Test_repository_FindByID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{tracer: lt, db: testDB}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入と明細を取得し集約へ再構築する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				purchaseID, productID := insertPurchaseWithDetail(ctx, t, drv, "d1000000-0000-4000-8000-000000000001")

				got, err := repo.FindByID(ctx, purchaseID)
				require.NoError(t, err)
				assert.Equal(t, purchaseID, got.ID())
				assert.Equal(t, mustParse(t, seedUnprocessedSID), got.StatusID())
				assert.Equal(t, 176500, got.TotalAmount())
				require.Len(t, got.Details(), 1)
				assert.Equal(t, productID, got.Details()[0].ProductID())
				assert.Equal(t, "80000", got.Details()[0].UnitPrice().String())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないIDの場合はNotFoundを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				missing, err := uuid.New()
				require.NoError(t, err)
				_, ferr := repo.FindByID(ctx, missing)
				require.ErrorIs(t, ferr, apperror.ErrNotFound)
			})
		})

		t.Run("DB行がドメイン不変条件に違反する場合はErrInternalへ正規化する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := mustParse(t, "d2000000-0000-4000-8000-000000000001")
				_, err := drv.Exec(ctx,
					"INSERT INTO products (id, name, description, price, quantity, stock_warning_threshold, status_id, category_id, published_at) "+
						"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())",
					productID, "purchase-repo-broken", nil, 80000, 20, nil, seedStatusInStock, seedCategory,
				)
				require.NoError(t, err)

				purchaseID, err := uuid.New()
				require.NoError(t, err)
				// code="" は Reconstruct の不変条件（ErrInvalidCode）に違反する破損行。
				_, err = drv.Exec(ctx,
					"INSERT INTO purchases (id, code, user_id, status_id, subtotal_amount, tax_amount, shipping_fee, total_amount) "+
						"VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
					purchaseID, "", mustParse(t, seedUserID), mustParse(t, seedUnprocessedSID), 160000, 16000, 500, 176500,
				)
				require.NoError(t, err)
				detailID, err := uuid.New()
				require.NoError(t, err)
				_, err = drv.Exec(ctx,
					"INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price) VALUES ($1,$2,$3,$4,$5)",
					detailID, purchaseID, productID, 2, 80000,
				)
				require.NoError(t, err)

				_, ferr := repo.FindByID(ctx, purchaseID)
				require.ErrorIs(t, ferr, apperror.ErrInternal)
			})
		})

		t.Run("明細の単価が負値の場合はErrInternalへ正規化する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := mustParse(t, "d3000000-0000-4000-8000-000000000001")
				_, err := drv.Exec(
					ctx,
					"INSERT INTO products (id, name, description, price, quantity, stock_warning_threshold, status_id, category_id, published_at) "+
						"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())",
					productID,
					"purchase-repo-negprice",
					nil,
					80000,
					20,
					nil,
					seedStatusInStock,
					seedCategory,
				)
				require.NoError(t, err)

				purchaseID, err := uuid.New()
				require.NoError(t, err)
				_, err = drv.Exec(ctx,
					"INSERT INTO purchases (id, code, user_id, status_id, subtotal_amount, tax_amount, shipping_fee, total_amount) "+
						"VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
					purchaseID, "repo-negprice", mustParse(t, seedUserID), mustParse(t, seedUnprocessedSID), 160000, 16000, 500, 176500,
				)
				require.NoError(t, err)
				detailID, err := uuid.New()
				require.NoError(t, err)
				// unit_price=-1 は money.Price の非負不変条件（ErrNegativePrice）に違反する破損行。
				_, err = drv.Exec(ctx,
					"INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price) VALUES ($1,$2,$3,$4,$5::numeric)",
					detailID, purchaseID, productID, 2, "-1",
				)
				require.NoError(t, err)

				_, ferr := repo.FindByID(ctx, purchaseID)
				require.ErrorIs(t, ferr, apperror.ErrInternal)
			})
		})
	})
}
