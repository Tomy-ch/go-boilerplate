package purchase

import (
	"context"
	"math"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/lexicon/money"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

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
	// unknownStatusCode は、購入ステータスマスタにもドメインにも存在しない code です（seed は 1〜9）。
	unknownStatusCode = 99
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

// reconstructWithStatusCode は、指定ステータスコードで再構築した最小構成の購入エンティティを返します。
// timestamps は全て未発生とし、コードと日時の整合検証に触れずステータスコードだけを狙って与えます。
func reconstructWithStatusCode(t *testing.T, salt string, statusCode int) *domainpurchase.Purchase {
	t.Helper()

	unitPrice, err := money.NewPrice(decimaltestkit.MustParse(t, "800"))
	require.NoError(t, err)
	detail := domainpurchase.NewPurchaseDetail(
		uuidtestkit.NewTestFromSalt(t, salt+"_detail_id"),
		domainpurchase.PurchaseDetailAttributes{
			ProductID: uuidtestkit.NewTestFromSalt(t, salt+"_product_id"),
			Quantity:  1,
			UnitPrice: unitPrice,
		},
	)

	entity, err := domainpurchase.Reconstruct(
		uuidtestkit.NewTestFromSalt(t, salt+"_id"),
		domainpurchase.Attributes{
			Code:           "code-" + salt,
			UserID:         uuidtestkit.NewTestFromSalt(t, salt+"_user_id"),
			StatusID:       uuidtestkit.NewTestFromSalt(t, salt+"_status_id"),
			StatusCode:     statusCode,
			SubtotalAmount: 800,
			TotalAmount:    800,
			Details:        []domainpurchase.PurchaseDetail{detail},
			OrderedAt:      time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC),
		},
	)
	require.NoError(t, err)
	return entity
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
				assert.Nil(t, got.ShippedAt)
				assert.Nil(t, got.DeliveredAt)
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
				assert.Equal(t, domainpurchase.StatusUnprocessed.Code(), got.StatusCode())
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
				assert.Equal(t, domainpurchase.StatusPaid.Code(), got.StatusCode())
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
				_, err = locked.Pay(now)
				require.NoError(t, err)
				require.NoError(t, repo.UpdatePaid(ctx, locked))

				// 再読込で status_id が支払い済み（code=7）へ解決され、paid_at がセットされる。
				reread, err := repo.LockByID(ctx, purchaseID)
				require.NoError(t, err)
				assert.Equal(t, domainpurchase.StatusPaid.Code(), reread.StatusCode())
				require.NotNil(t, reread.PaidAt())
				assert.Equal(t, now.UTC(), reread.PaidAt().UTC())

				// 擬似決済は在庫を操作しない（挿入時の 20 のまま）。
				var stock int
				require.NoError(t, drv.QueryRow(ctx, "SELECT quantity FROM products WHERE id=$1", productID).Scan(&stock))
				assert.Equal(t, 20, stock)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("statusCodeがSMALLINT列に収まらない場合、クエリを発行せずオーバーフローエラーを返す", func(t *testing.T) {
			t.Parallel()

			entity := reconstructWithStatusCode(t, "update_paid_overflow", math.MaxInt16+1)
			require.ErrorIs(t, repo.UpdatePaid(context.Background(), entity), safecast.ErrOverflow)
		})
	})
}

func Test_repository_UpdateShipped(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{tracer: lt, db: testDB}
	paidAt := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	shippedAt := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("status_idを発送済みへ更新しshipped_atをセットし在庫は変更しない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				purchaseID, productID := insertPurchaseWithDetail(ctx, t, drv, "f4000000-0000-4000-8000-000000000001")

				paid, err := repo.LockByID(ctx, purchaseID)
				require.NoError(t, err)
				_, err = paid.Pay(paidAt)
				require.NoError(t, err)
				require.NoError(t, repo.UpdatePaid(ctx, paid))

				locked, err := repo.LockByID(ctx, purchaseID)
				require.NoError(t, err)
				_, err = locked.Ship(shippedAt)
				require.NoError(t, err)
				require.NoError(t, repo.UpdateShipped(ctx, locked))

				// 再読込で status_id が発送済み（code=8）へ解決され、shipped_at がセットされる。
				reread, err := repo.LockByID(ctx, purchaseID)
				require.NoError(t, err)
				assert.Equal(t, domainpurchase.StatusShipped.Code(), reread.StatusCode())
				require.NotNil(t, reread.ShippedAt())
				assert.Equal(t, shippedAt.UTC(), reread.ShippedAt().UTC())

				// 発送は在庫を操作しない（挿入時の 20 のまま）。
				var stock int
				require.NoError(t, drv.QueryRow(ctx, "SELECT quantity FROM products WHERE id=$1", productID).Scan(&stock))
				assert.Equal(t, 20, stock)
			})
		})

		t.Run("読み取りモデルにshipped_atが反映される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				purchaseID, _ := insertPurchaseWithDetail(ctx, t, drv, "f4000000-0000-4000-8000-000000000002")

				paid, err := repo.LockByID(ctx, purchaseID)
				require.NoError(t, err)
				_, err = paid.Pay(paidAt)
				require.NoError(t, err)
				require.NoError(t, repo.UpdatePaid(ctx, paid))

				locked, err := repo.LockByID(ctx, purchaseID)
				require.NoError(t, err)
				_, err = locked.Ship(shippedAt)
				require.NoError(t, err)
				require.NoError(t, repo.UpdateShipped(ctx, locked))

				detail, err := repo.FindDetailByID(ctx, purchaseID)
				require.NoError(t, err)
				assert.Equal(t, "発送済み", detail.StatusName)
				require.NotNil(t, detail.ShippedAt)
				assert.Equal(t, shippedAt.UTC(), detail.ShippedAt.UTC())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("マスタに無いステータスコードはドメインが再構築を拒否し永続化まで届かない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				purchaseID, _ := insertPurchaseWithDetail(ctx, t, drv, "f4000000-0000-4000-8000-000000000003")

				locked, err := repo.LockByID(ctx, purchaseID)
				require.NoError(t, err)

				// マスタに無い code は、そもそもドメインが再構築を拒否するため infra まで届かない。
				// status_id のサブクエリが NULL になる経路は Status VO の導入で到達不能になった。
				_, err = domainpurchase.Reconstruct(locked.ID(), domainpurchase.Attributes{
					Code:           locked.Code(),
					UserID:         locked.UserID(),
					StatusID:       locked.StatusID(),
					StatusCode:     unknownStatusCode,
					SubtotalAmount: locked.SubtotalAmount(),
					TaxAmount:      locked.TaxAmount(),
					ShippingFee:    locked.ShippingFee(),
					TotalAmount:    locked.TotalAmount(),
					Details:        locked.Details(),
					OrderedAt:      locked.OrderedAt(),
				})
				require.ErrorIs(t, err, domainpurchase.ErrInvalidStatusID)
			})
		})

		t.Run("statusCodeがSMALLINT列に収まらない場合、クエリを発行せずオーバーフローエラーを返す", func(t *testing.T) {
			t.Parallel()

			entity := reconstructWithStatusCode(t, "update_shipped_overflow", math.MaxInt16+1)
			require.ErrorIs(t, repo.UpdateShipped(context.Background(), entity), safecast.ErrOverflow)
		})
	})
}

func Test_repository_UpdateDelivered(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{tracer: lt, db: testDB}
	paidAt := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	shippedAt := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)
	deliveredAt := time.Date(2026, time.July, 28, 9, 0, 0, 0, time.UTC)

	// ship は、挿入直後の購入を支払い済み → 発送済みまで進め、配達の起点状態を作ります。
	ship := func(ctx context.Context, t *testing.T, purchaseID uuid.UUID) {
		t.Helper()

		paid, err := repo.LockByID(ctx, purchaseID)
		require.NoError(t, err)
		_, err = paid.Pay(paidAt)
		require.NoError(t, err)
		require.NoError(t, repo.UpdatePaid(ctx, paid))

		toShip, err := repo.LockByID(ctx, purchaseID)
		require.NoError(t, err)
		_, err = toShip.Ship(shippedAt)
		require.NoError(t, err)
		require.NoError(t, repo.UpdateShipped(ctx, toShip))
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("status_idを配達済みへ更新しdelivered_atをセットし在庫は変更しない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				purchaseID, productID := insertPurchaseWithDetail(ctx, t, drv, "f5000000-0000-4000-8000-000000000001")
				ship(ctx, t, purchaseID)

				locked, err := repo.LockByID(ctx, purchaseID)
				require.NoError(t, err)
				_, err = locked.Deliver(deliveredAt)
				require.NoError(t, err)
				require.NoError(t, repo.UpdateDelivered(ctx, locked))

				// 再読込で status_id が配達済み（code=9）へ解決され、delivered_at がセットされる。
				reread, err := repo.LockByID(ctx, purchaseID)
				require.NoError(t, err)
				assert.Equal(t, domainpurchase.StatusDelivered.Code(), reread.StatusCode())
				require.NotNil(t, reread.DeliveredAt())
				assert.Equal(t, deliveredAt.UTC(), reread.DeliveredAt().UTC())

				// 配達完了は在庫を操作しない（挿入時の 20 のまま）。
				var stock int
				require.NoError(t, drv.QueryRow(ctx, "SELECT quantity FROM products WHERE id=$1", productID).Scan(&stock))
				assert.Equal(t, 20, stock)
			})
		})

		t.Run("読み取りモデルにdelivered_atが反映される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				purchaseID, _ := insertPurchaseWithDetail(ctx, t, drv, "f5000000-0000-4000-8000-000000000002")
				ship(ctx, t, purchaseID)

				locked, err := repo.LockByID(ctx, purchaseID)
				require.NoError(t, err)
				_, err = locked.Deliver(deliveredAt)
				require.NoError(t, err)
				require.NoError(t, repo.UpdateDelivered(ctx, locked))

				detail, err := repo.FindDetailByID(ctx, purchaseID)
				require.NoError(t, err)
				assert.Equal(t, "配達済み", detail.StatusName)
				require.NotNil(t, detail.DeliveredAt)
				assert.Equal(t, deliveredAt.UTC(), detail.DeliveredAt.UTC())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("マスタに無いステータスコードはドメインが再構築を拒否し永続化まで届かない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				purchaseID, _ := insertPurchaseWithDetail(ctx, t, drv, "f5000000-0000-4000-8000-000000000003")

				locked, err := repo.LockByID(ctx, purchaseID)
				require.NoError(t, err)

				// マスタに無い code は、そもそもドメインが再構築を拒否するため infra まで届かない。
				// status_id のサブクエリが NULL になる経路は Status VO の導入で到達不能になった。
				_, err = domainpurchase.Reconstruct(locked.ID(), domainpurchase.Attributes{
					Code:           locked.Code(),
					UserID:         locked.UserID(),
					StatusID:       locked.StatusID(),
					StatusCode:     unknownStatusCode,
					SubtotalAmount: locked.SubtotalAmount(),
					TaxAmount:      locked.TaxAmount(),
					ShippingFee:    locked.ShippingFee(),
					TotalAmount:    locked.TotalAmount(),
					Details:        locked.Details(),
					OrderedAt:      locked.OrderedAt(),
				})
				require.ErrorIs(t, err, domainpurchase.ErrInvalidStatusID)
			})
		})

		t.Run("statusCodeがSMALLINT列に収まらない場合、クエリを発行せずオーバーフローエラーを返す", func(t *testing.T) {
			t.Parallel()

			entity := reconstructWithStatusCode(t, "update_delivered_overflow", math.MaxInt16+1)
			require.ErrorIs(t, repo.UpdateDelivered(context.Background(), entity), safecast.ErrOverflow)
		})
	})
}

func Test_repository_ExistsInProgressByUserID(t *testing.T) {
	t.Parallel()

	// 購入ステータスマスタ（seed 済み）。進行中は終端（完了 / キャンセル / 配達済み）の否定で判定する。
	const (
		seedCompletedSID = "1904bf76-7d37-4288-bc15-359d2512ac91"
		seedCanceledSID  = "e9d72547-adfe-48d9-9037-bd1f55d4158b"
		seedShippedSID   = "5c9a1f3b-2d4e-4b6f-9c8a-3e0d1f2b4c5d"
		seedDeliveredSID = "6d0b2a4c-3e5f-4c7a-ad9b-4f1e2a3c5d6e"
	)

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{tracer: lt, db: testDB}

	orderedAt := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未処理の購入を持つユーザーはtrueを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				userID := "fa000000-0000-4000-8000-000000000001"
				insertFeedUser(ctx, t, drv, userID)
				insertPurchase(ctx, t, drv, "fb000000-0000-4000-8000-000000000001", userID, seedUnprocessedSID, 100, orderedAt)

				exists, err := repo.ExistsInProgressByUserID(ctx, mustParse(t, userID))
				require.NoError(t, err)
				assert.True(t, exists)
			})
		})

		t.Run("発送済みの購入しか持たないユーザーもtrueを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				userID := "fa000000-0000-4000-8000-000000000002"
				insertFeedUser(ctx, t, drv, userID)
				insertPurchase(ctx, t, drv, "fb000000-0000-4000-8000-000000000002", userID, seedShippedSID, 100, orderedAt)

				exists, err := repo.ExistsInProgressByUserID(ctx, mustParse(t, userID))
				require.NoError(t, err)
				assert.True(t, exists)
			})
		})

		t.Run("終端ステータスの購入しか持たないユーザーはfalseを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				userID := "fa000000-0000-4000-8000-000000000003"
				insertFeedUser(ctx, t, drv, userID)
				insertPurchase(ctx, t, drv, "fb000000-0000-4000-8000-000000000003", userID, seedCompletedSID, 100, orderedAt)
				insertPurchase(ctx, t, drv, "fb000000-0000-4000-8000-000000000004", userID, seedCanceledSID, 200, orderedAt)
				insertPurchase(ctx, t, drv, "fb000000-0000-4000-8000-000000000005", userID, seedDeliveredSID, 300, orderedAt)

				exists, err := repo.ExistsInProgressByUserID(ctx, mustParse(t, userID))
				require.NoError(t, err)
				assert.False(t, exists)
			})
		})

		t.Run("購入を1件も持たないユーザーはfalseを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				userID := "fa000000-0000-4000-8000-000000000004"
				insertFeedUser(ctx, t, drv, userID)

				exists, err := repo.ExistsInProgressByUserID(ctx, mustParse(t, userID))
				require.NoError(t, err)
				assert.False(t, exists)
			})
		})

		t.Run("他ユーザーの進行中購入は判定に含めない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				subject := "fa000000-0000-4000-8000-000000000005"
				other := "fa000000-0000-4000-8000-000000000006"
				insertFeedUser(ctx, t, drv, subject)
				insertFeedUser(ctx, t, drv, other)
				insertPurchase(ctx, t, drv, "fb000000-0000-4000-8000-000000000006", subject, seedCompletedSID, 100, orderedAt)
				insertPurchase(ctx, t, drv, "fb000000-0000-4000-8000-000000000007", other, seedUnprocessedSID, 200, orderedAt)

				exists, err := repo.ExistsInProgressByUserID(ctx, mustParse(t, subject))
				require.NoError(t, err)
				assert.False(t, exists)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化される", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			exists, err := repo.ExistsInProgressByUserID(ctx, mustParse(t, seedUserID))
			assert.False(t, exists)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

func Test_toPurchaseDetails(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("明細行を購入明細の値オブジェクトへ写像する", func(t *testing.T) {
			t.Parallel()

			detailID := mustParse(t, "e2000000-0000-4000-8000-0000000000d1")
			productID := mustParse(t, "e2000000-0000-4000-8000-000000000001")
			rows := []*gen.ListPurchaseDetailsByPurchaseIDRow{
				{PurchaseDetails: gen.PurchaseDetails{
					ID:        detailID,
					ProductID: productID,
					Quantity:  2,
					UnitPrice: decimaltestkit.MustParse(t, "800"),
				}},
			}

			details, err := toPurchaseDetails(rows)
			require.NoError(t, err)
			require.Len(t, details, 1)
			// 明細 ID と商品 ID は同じ uuid.UUID なので、取り違えを検出できるよう両方を固定する。
			assert.Equal(t, detailID, details[0].ID())
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

func Test_toFeedItem(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入履歴フィードの行を読み取りモデルの各フィールドへ写像する", func(t *testing.T) {
			t.Parallel()

			id := mustParse(t, "e3000000-0000-4000-8000-000000000001")
			statusID := mustParse(t, "e3000000-0000-4000-8000-0000000000a1")
			orderedAt := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)

			item := toFeedItem(id, "feed-code-1", 176500, orderedAt, statusID, "未処理")

			assert.Equal(t, domainpurchase.FeedItem{
				ID:          id,
				Code:        "feed-code-1",
				TotalAmount: 176500,
				StatusID:    statusID,
				StatusName:  "未処理",
				OrderedAt:   orderedAt,
			}, item)
		})
	})
}

func Test_mustToInt16(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入ステータスコードを同値のint16へ変換する", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, int16(7), mustToInt16(7))
		})

		t.Run("int16の下限と上限の値をそのまま変換する", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, int16(math.MinInt16), mustToInt16(math.MinInt16))
			assert.Equal(t, int16(math.MaxInt16), mustToInt16(math.MaxInt16))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("int16の上限を超える場合、オーバーフローエラーでpanicする", func(t *testing.T) {
			t.Parallel()

			defer func() {
				err, ok := recover().(error)
				require.True(t, ok)
				require.ErrorIs(t, err, safecast.ErrOverflow)
			}()

			mustToInt16(math.MaxInt16 + 1)
		})

		t.Run("int16の下限を下回る場合、オーバーフローエラーでpanicする", func(t *testing.T) {
			t.Parallel()

			defer func() {
				err, ok := recover().(error)
				require.True(t, ok)
				require.ErrorIs(t, err, safecast.ErrOverflow)
			}()

			mustToInt16(math.MinInt16 - 1)
		})
	})
}
