package purchase

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
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

// 既存 seed の FK 対象（支払い済みは code=7）。
const (
	seedStatusInStock  = "093170fb-83a2-4864-a2b3-53236eaf3597"
	seedCategory       = "5dd52d84-78eb-4a52-ba0b-2e11c95c2af2"
	seedUserID         = "550e8400-e29b-41d4-a716-446655440000"
	seedUnprocessedSID = "a66c996c-86b2-41d8-9bdd-9b685fb7c47d"
	seedPaidSID        = "4b8f0e2a-1c3d-4a5e-8b7f-2d9c0e1a3b4c"
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
				assert.Nil(t, got.PaidAt())
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

func Test_repository_FindDetailByID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{tracer: lt, db: testDB}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入詳細をステータス名解決済みの読み取りモデルで取得する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				purchaseID, productID := insertPurchaseWithDetail(ctx, t, drv, "e1000000-0000-4000-8000-000000000001")

				got, err := repo.FindDetailByID(ctx, purchaseID)
				require.NoError(t, err)
				assert.Equal(t, purchaseID, got.ID)
				assert.Equal(t, mustParse(t, seedUnprocessedSID), got.StatusID)
				assert.Equal(t, "未処理", got.StatusName)
				assert.Equal(t, 176500, got.TotalAmount)
				assert.Nil(t, got.PaidAt)
				assert.Nil(t, got.CanceledAt)
				require.Len(t, got.Details, 1)
				assert.Equal(t, productID, got.Details[0].ProductID())
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
				_, ferr := repo.FindDetailByID(ctx, missing)
				require.ErrorIs(t, ferr, apperror.ErrNotFound)
			})
		})
	})
}

// insertPaidPurchase は、支払い済みステータス（code=7）+ paid_at セット済みの購入を挿入し、購入 ID を返します。
func insertPaidPurchase(ctx context.Context, t *testing.T, db driver.DBTX, seed string, paidAt time.Time) uuid.UUID {
	t.Helper()
	productID := mustParse(t, seed)
	_, err := db.Exec(ctx,
		"INSERT INTO products (id, name, description, price, quantity, stock_warning_threshold, status_id, category_id, published_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())",
		productID, "purchase-repo-paid-"+seed, nil, 80000, 20, nil, seedStatusInStock, seedCategory,
	)
	require.NoError(t, err)

	purchaseID, err := uuid.New()
	require.NoError(t, err)
	_, err = db.Exec(ctx,
		"INSERT INTO purchases (id, code, user_id, status_id, subtotal_amount, tax_amount, shipping_fee, total_amount, paid_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)",
		purchaseID, "repo-paid-"+seed, mustParse(t, seedUserID), mustParse(t, seedPaidSID), 160000, 16000, 500, 176500, paidAt,
	)
	require.NoError(t, err)

	detailID, err := uuid.New()
	require.NoError(t, err)
	_, err = db.Exec(ctx,
		"INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price) VALUES ($1,$2,$3,$4,$5)",
		detailID, purchaseID, productID, 2, 80000,
	)
	require.NoError(t, err)
	return purchaseID
}

func Test_repository_LockByID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{tracer: lt, db: testDB}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未処理購入を行ロックし再構築する（PaidAtはnil）", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				purchaseID, productID := insertPurchaseWithDetail(ctx, t, drv, "f1000000-0000-4000-8000-000000000001")

				got, err := repo.LockByID(ctx, purchaseID)
				require.NoError(t, err)
				assert.Equal(t, purchaseID, got.ID())
				assert.Equal(t, mustParse(t, seedUserID), got.UserID())
				assert.Equal(t, domainpurchase.StatusCodeUnprocessed, got.StatusCode())
				assert.Nil(t, got.PaidAt())
				require.Len(t, got.Details(), 1)
				assert.Equal(t, productID, got.Details()[0].ProductID())
			})
		})

		t.Run("支払い済み購入を行ロックするとPaidAtが読み取りモデルへ正しくマッピングされる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				paidAt := time.Date(2026, time.July, 24, 9, 0, 0, 0, time.UTC)
				purchaseID := insertPaidPurchase(ctx, t, drv, "f2000000-0000-4000-8000-000000000001", paidAt)

				got, err := repo.LockByID(ctx, purchaseID)
				require.NoError(t, err)
				assert.Equal(t, domainpurchase.StatusCodePaid, got.StatusCode())
				require.NotNil(t, got.PaidAt())
				assert.Equal(t, paidAt.UTC(), got.PaidAt().UTC())
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
				_, lerr := repo.LockByID(ctx, missing)
				require.ErrorIs(t, lerr, apperror.ErrNotFound)
			})
		})
	})
}

func Test_repository_UpdatePaid(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{tracer: lt, db: testDB}
	now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("status_idを支払い済みへ更新しpaid_atをセットし在庫は変更しない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				purchaseID, productID := insertPurchaseWithDetail(ctx, t, drv, "f3000000-0000-4000-8000-000000000001")

				locked, err := repo.LockByID(ctx, purchaseID)
				require.NoError(t, err)
				require.NoError(t, locked.Pay(now))
				require.NoError(t, repo.UpdatePaid(ctx, locked))

				// 再読込で status_id が支払い済み（code=7）へ解決され、paid_at がセットされる。
				reread, err := repo.LockByID(ctx, purchaseID)
				require.NoError(t, err)
				assert.Equal(t, domainpurchase.StatusCodePaid, reread.StatusCode())
				require.NotNil(t, reread.PaidAt())
				assert.Equal(t, now.UTC(), reread.PaidAt().UTC())

				// 擬似決済は在庫を操作しない（挿入時の 20 のまま）。
				var stock int
				require.NoError(t, drv.QueryRow(ctx, "SELECT quantity FROM products WHERE id=$1", productID).Scan(&stock))
				assert.Equal(t, 20, stock)
			})
		})
	})
}

func Test_toPurchaseDetails(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("明細行を購入明細の値オブジェクトへ写像する", func(t *testing.T) {
			t.Parallel()

			productID := mustParse(t, "e2000000-0000-4000-8000-000000000001")
			rows := []*gen.ListPurchaseDetailsByPurchaseIDRow{
				{PurchaseDetails: gen.PurchaseDetails{
					ID:        mustParse(t, "e2000000-0000-4000-8000-0000000000d1"),
					ProductID: productID,
					Quantity:  2,
					UnitPrice: decimaltestkit.MustParse(t, "800"),
				}},
			}

			details, err := toPurchaseDetails(rows)
			require.NoError(t, err)
			require.Len(t, details, 1)
			assert.Equal(t, productID, details[0].ProductID())
			assert.Equal(t, 2, details[0].Quantity())
			assert.Equal(t, "800", details[0].UnitPrice().String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("単価が負で再構築不能な場合はErrInternalへ正規化する", func(t *testing.T) {
			t.Parallel()

			rows := []*gen.ListPurchaseDetailsByPurchaseIDRow{
				{PurchaseDetails: gen.PurchaseDetails{
					ID:        mustParse(t, "e2000000-0000-4000-8000-0000000000d2"),
					ProductID: mustParse(t, "e2000000-0000-4000-8000-000000000002"),
					Quantity:  1,
					UnitPrice: decimaltestkit.MustParse(t, "-1"),
				}},
			}

			_, err := toPurchaseDetails(rows)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}
