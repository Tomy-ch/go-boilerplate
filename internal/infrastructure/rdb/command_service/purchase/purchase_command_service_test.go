package purchase

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/kernel/money"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/decimal"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
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

// mustPrice は、テスト用に十進文字列（ドル）から非負の money.Price を構築します。
func mustPrice(t *testing.T, s string) money.Price {
	t.Helper()
	p, err := money.NewPrice(decimaltestkit.MustParse(t, s))
	require.NoError(t, err)
	return p
}

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
				assert.Equal(t, "80000", locked[0].Price().String())
				assert.Equal(t, 20, locked[0].Quantity())
			})
		})

		t.Run("サブセント単価をNUMERIC精度を保ったままロックする", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				pid := mustParse(t, "c1000000-0000-4000-8000-0000000000aa")
				// price=19.995（サブセント）を NUMERIC 列へ直接挿入し、価格スケールの往復を検証する。
				_, err := drv.Exec(ctx,
					"INSERT INTO products "+
						"(id, name, description, price, quantity, stock_warning_threshold, status_id, category_id, published_at) "+
						"VALUES ($1,$2,$3,$4::numeric,$5,$6,$7,$8,NOW())",
					pid, "subcent-"+pid.String(), nil, "19.995", 20, nil, seedStatusInStock, seedCategory,
				)
				require.NoError(t, err)

				locked, err := svc.LockProducts(ctx, []uuid.UUID{pid})
				require.NoError(t, err)
				require.Len(t, locked, 1)
				assert.Equal(t, "19.995", locked[0].Price().String())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("負値の価格は再構築不能としてErrInternalへ正規化する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				pid := mustParse(t, "c1000000-0000-4000-8000-0000000000bb")
				// NUMERIC 列は負値を格納できるが、money.Price は非負不変条件を持つ。
				// 格納行からの Price 再構築失敗が ErrInternal へ正規化されることを検証する。
				_, err := drv.Exec(ctx,
					"INSERT INTO products "+
						"(id, name, description, price, quantity, stock_warning_threshold, status_id, category_id, published_at) "+
						"VALUES ($1,$2,$3,$4::numeric,$5,$6,$7,$8,NOW())",
					pid, "neg-"+pid.String(), nil, "-1", 20, nil, seedStatusInStock, seedCategory,
				)
				require.NoError(t, err)

				_, err = svc.LockProducts(ctx, []uuid.UUID{pid})
				require.ErrorIs(t, err, apperror.ErrInternal)
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

				var unitPrice decimal.Decimal
				require.NoError(t, drv.QueryRow(ctx,
					"SELECT unit_price FROM purchase_details WHERE purchase_id=$1", entity.ID(),
				).Scan(&unitPrice))
				assert.Equal(t, "80000", unitPrice.String())
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
				stale := []domainpurchase.LockedProduct{domainpurchase.NewLockedProduct(pid, mustPrice(t, "80000"), 20)}

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
					domainpurchase.NewLockedProduct(pid1, mustPrice(t, "80000"), 20),
					domainpurchase.NewLockedProduct(pid2, mustPrice(t, "1500"), 20),
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
				locked := []domainpurchase.LockedProduct{domainpurchase.NewLockedProduct(pid, mustPrice(t, "80000"), 20)}
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

func Test_commandService_LockPurchase(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	svc := &commandService{tracer: lt, db: testDB}
	userID := mustParse(t, seedUserID)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入行をロックし現在状態と明細込みで再構築する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				pid := mustParse(t, "d1000000-0000-4000-8000-000000000001")
				insertTestProduct(ctx, t, drv, pid, 80000, 20)
				locked, err := svc.LockProducts(ctx, []uuid.UUID{pid})
				require.NoError(t, err)
				created := newPurchase(t, userID, pid, 2, locked)
				require.NoError(t, svc.CreatePurchase(ctx, created))

				actual, err := svc.LockPurchase(ctx, created.ID())
				require.NoError(t, err)
				assert.Equal(t, created.ID(), actual.ID())
				assert.Equal(t, userID, actual.UserID())
				assert.Equal(t, domainpurchase.StatusCodeUnprocessed, actual.StatusCode())
				assert.Nil(t, actual.PaidAt())
				assert.Nil(t, actual.CanceledAt())
				require.Len(t, actual.Details(), 1)
				assert.Equal(t, pid, actual.Details()[0].ProductID())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しない購入IDの場合はErrNotFoundを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				missing := mustParse(t, "d1000000-0000-4000-8000-0000000000ff")
				_, err := svc.LockPurchase(ctx, missing)
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})

		t.Run("明細の単価が負値で再構築不能な場合はErrInternalへ正規化する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				pid := mustParse(t, "d1000000-0000-4000-8000-0000000000ce")
				insertTestProduct(ctx, t, drv, pid, 80000, 20)

				purchaseID := mustParse(t, "d1000000-0000-4000-8000-0000000000cf")
				_, err := drv.Exec(ctx,
					"INSERT INTO purchases (id, code, user_id, status_id, subtotal_amount, tax_amount, shipping_fee, total_amount) "+
						"VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
					purchaseID, "lock-neg-code", userID, mustParse(t, seedUnprocessedSID), 160000, 16000, 500, 176500,
				)
				require.NoError(t, err)
				detailID := mustParse(t, "d1000000-0000-4000-8000-0000000000d0")
				// NUMERIC 列は負値を格納できるが money.Price は非負不変条件を持つため再構築が失敗する。
				_, err = drv.Exec(ctx,
					"INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price) VALUES ($1,$2,$3,$4,$5::numeric)",
					detailID, purchaseID, pid, 2, "-1",
				)
				require.NoError(t, err)

				_, err = svc.LockPurchase(ctx, purchaseID)
				require.ErrorIs(t, err, apperror.ErrInternal)
			})
		})
	})
}

func Test_commandService_CancelPurchase(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	svc := &commandService{tracer: lt, db: testDB}
	userID := mustParse(t, seedUserID)
	now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("在庫を復元しstatus_idをキャンセルへ更新しcanceled_atをセットする", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				pid := mustParse(t, "d2000000-0000-4000-8000-000000000001")
				insertTestProduct(ctx, t, drv, pid, 80000, 20)
				locked, err := svc.LockProducts(ctx, []uuid.UUID{pid})
				require.NoError(t, err)
				created := newPurchase(t, userID, pid, 2, locked)
				require.NoError(t, svc.CreatePurchase(ctx, created)) // 在庫 20 → 18

				var afterCreate int
				require.NoError(t, drv.QueryRow(ctx, "SELECT quantity FROM products WHERE id=$1", pid).Scan(&afterCreate))
				require.Equal(t, 18, afterCreate)

				lockedPurchase, err := svc.LockPurchase(ctx, created.ID())
				require.NoError(t, err)
				require.NoError(t, lockedPurchase.Cancel(now))
				require.NoError(t, svc.CancelPurchase(ctx, lockedPurchase))

				// 在庫が明細分（2）復元される（減算の対称）。
				var restored int
				require.NoError(t, drv.QueryRow(ctx, "SELECT quantity FROM products WHERE id=$1", pid).Scan(&restored))
				assert.Equal(t, 20, restored)

				// status_id は code=6（キャンセル）へ解決され、canceled_at がセットされる。
				var statusCode int
				var canceledAt *time.Time
				require.NoError(t, drv.QueryRow(ctx,
					"SELECT ps.code, p.canceled_at FROM purchases AS p "+
						"INNER JOIN purchase_statuses AS ps ON p.status_id = ps.id WHERE p.id=$1", created.ID(),
				).Scan(&statusCode, &canceledAt))
				assert.Equal(t, domainpurchase.StatusCodeCanceled, statusCode)
				assert.NotNil(t, canceledAt)
			})
		})

		t.Run("複数明細の在庫をすべて復元する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				pid1 := mustParse(t, "d3000000-0000-4000-8000-000000000001")
				pid2 := mustParse(t, "d3000000-0000-4000-8000-000000000002")
				insertTestProduct(ctx, t, drv, pid1, 80000, 20)
				insertTestProduct(ctx, t, drv, pid2, 1500, 10)
				locked, err := svc.LockProducts(ctx, []uuid.UUID{pid1, pid2})
				require.NoError(t, err)

				id, err := uuid.New()
				require.NoError(t, err)
				code, err := uuid.New()
				require.NoError(t, err)
				d1, err := uuid.New()
				require.NoError(t, err)
				d2, err := uuid.New()
				require.NoError(t, err)
				created, err := domainpurchase.New(id, code.String(), userID, []domainpurchase.DetailInput{
					{ID: d1, ProductID: pid1, Quantity: 3},
					{ID: d2, ProductID: pid2, Quantity: 4},
				}, locked)
				require.NoError(t, err)
				require.NoError(t, svc.CreatePurchase(ctx, created)) // 20→17, 10→6

				lockedPurchase, err := svc.LockPurchase(ctx, created.ID())
				require.NoError(t, err)
				require.NoError(t, lockedPurchase.Cancel(now))
				require.NoError(t, svc.CancelPurchase(ctx, lockedPurchase))

				var q1, q2 int
				require.NoError(t, drv.QueryRow(ctx, "SELECT quantity FROM products WHERE id=$1", pid1).Scan(&q1))
				require.NoError(t, drv.QueryRow(ctx, "SELECT quantity FROM products WHERE id=$1", pid2).Scan(&q2))
				assert.Equal(t, 20, q1)
				assert.Equal(t, 10, q2)
			})
		})
	})
}

func TestNew(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func Test_toInt16(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func Test_toInt32(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
