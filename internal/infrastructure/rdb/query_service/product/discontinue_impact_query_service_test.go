package product

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 既存 seed の FK 対象（users.prefecture_id / purchases.status_id が要求する行）。
const (
	seedPrefecture = "a03aaec4-3bd6-4bfb-8e47-2fbfa026d344"
	// 購入ステータスマスタ（seed 済み）。進行中と終端を 1 つずつ使う。
	seedShippedStatusID   = "5c9a1f3b-2d4e-4b6f-9c8a-3e0d1f2b4c5d"
	seedCompletedStatusID = "1904bf76-7d37-4288-bc15-359d2512ac91"
)

// insertDiscontinueUser は、廃番の影響見積もりが数える確定済みユーザーを挿入します。
// deleted が true のときは退会済みとして挿入し、母集団から外れることを確かめられるようにします。
func insertDiscontinueUser(ctx context.Context, t *testing.T, db driver.DBTX, id uuid.UUID, deleted bool) {
	t.Helper()
	deletedAt := "NULL"
	if deleted {
		deletedAt = "NOW()"
	}
	_, err := db.Exec(ctx,
		"INSERT INTO users "+
			"(id, first_name, last_name, email, phone, prefecture_id, city, street, postal_code, deleted_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,"+deletedAt+")",
		id, "Impact", "User", "impact-"+id.String()+"@example.com", "0000000000",
		seedPrefecture, "City", "Street", "000-0000",
	)
	require.NoError(t, err)
}

// insertDiscontinueCart は、対象商品の明細を 1 件持つカートを挿入します。owner が nil ならゲストのカートです。
func insertDiscontinueCart(
	ctx context.Context, t *testing.T, db driver.DBTX, cartID, productID uuid.UUID, owner *uuid.UUID,
) {
	t.Helper()
	// carts は所有者を user_id か session_token のどちらか一方だけ持つ（carts_owner_exclusive）。
	// ゲストのカートは後者で表すため、owner が nil のときは session_token を入れる。
	var sessionToken *string
	if owner == nil {
		token := "session-" + cartID.String()
		sessionToken = &token
	}

	_, err := db.Exec(ctx,
		"INSERT INTO carts (id, user_id, session_token, expires_at) VALUES ($1,$2,$3,NOW() + INTERVAL '1 day')",
		cartID, owner, sessionToken,
	)
	require.NoError(t, err)

	_, err = db.Exec(ctx,
		"INSERT INTO cart_items (id, cart_id, product_id, quantity) VALUES ($1,$2,$3,1)",
		uuidtestkit.NewTestFromSalt(t, "item_of_"+cartID.String()), cartID, productID,
	)
	require.NoError(t, err)
}

// insertImpactProduct は、廃番対象となる商品を挿入します。
func insertImpactProduct(ctx context.Context, t *testing.T, db driver.DBTX, id uuid.UUID) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO products "+
			"(id, name, description, price, quantity, stock_warning_threshold, status_id, category_id) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
		id, "廃番対象-"+id.String(), nil, 100, 10, nil, seedStatusInStock, seedCategory,
	)
	require.NoError(t, err)
}

// insertImpactPurchase は、対象商品の明細を持つ購入を 1 件挿入します。
// statusID で進行中と終端を撃ち分け、終端が母集団から外れることを確かめられるようにします。
func insertImpactPurchase(
	ctx context.Context, t *testing.T, db driver.DBTX, purchaseID, productID, userID uuid.UUID, statusID string,
) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO purchases "+
			"(id, code, user_id, status_id, subtotal_amount, tax_amount, shipping_fee, total_amount, ordered_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())",
		purchaseID, "code-"+purchaseID.String(), userID, statusID, 100, 0, 0, 100,
	)
	require.NoError(t, err)

	_, err = db.Exec(ctx,
		"INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price) VALUES ($1,$2,$3,1,100)",
		uuidtestkit.NewTestFromSalt(t, "detail_of_"+purchaseID.String()), purchaseID, productID,
	)
	require.NoError(t, err)
}

func TestNewDiscontinueImpactQueryService(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("依存を注入したクエリサービス実装を生成する", func(t *testing.T) {
			t.Parallel()

			testDB := testkit.NewTestDB(t)

			svc, ok := NewDiscontinueImpactQueryService(testDB, observability.NewNoopTracerFactory(t)).(*discontinueImpactService)
			require.True(t, ok)
			assert.Equal(t, testDB, svc.db)
			assert.NotNil(t, svc.tracer)
		})
	})
}

func Test_discontinueImpactService_EstimateDiscontinueImpact(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	txm := testkit.NewTestTransactionRunner(t)
	svc := &discontinueImpactService{db: testDB, tracer: observability.NewMockInfraLayerTracer(t)}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("どのカートにも入っていない商品は3件数とも0を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := uuidtestkit.NewTestFromSalt(t, "impact_untouched_product")
				insertImpactProduct(ctx, t, drv, productID)

				got, err := svc.EstimateDiscontinueImpact(ctx, productID)

				require.NoError(t, err)
				assert.Zero(t, got.AffectedCartCount)
				assert.Zero(t, got.AffectedUserCount)
				assert.Zero(t, got.InProgressPurchaseCount)
			})
		})

		t.Run("ゲストのカートはカート数に数えるが受給者数には数えない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := uuidtestkit.NewTestFromSalt(t, "impact_guest_product")
				insertImpactProduct(ctx, t, drv, productID)
				insertDiscontinueCart(ctx, t, drv, uuidtestkit.NewTestFromSalt(t, "impact_guest_cart"), productID, nil)

				got, err := svc.EstimateDiscontinueImpact(ctx, productID)

				require.NoError(t, err)
				assert.Equal(t, int64(1), got.AffectedCartCount)
				assert.Zero(t, got.AffectedUserCount)
			})
		})

		t.Run("退会済みユーザーのカートはカート数に数えるが受給者数には数えない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := uuidtestkit.NewTestFromSalt(t, "impact_deleted_product")
				userID := uuidtestkit.NewTestFromSalt(t, "impact_deleted_user")
				insertImpactProduct(ctx, t, drv, productID)
				insertDiscontinueUser(ctx, t, drv, userID, true)
				insertDiscontinueCart(
					ctx, t, drv, uuidtestkit.NewTestFromSalt(t, "impact_deleted_cart"), productID, &userID,
				)

				got, err := svc.EstimateDiscontinueImpact(ctx, productID)

				require.NoError(t, err)
				assert.Equal(t, int64(1), got.AffectedCartCount)
				assert.Zero(t, got.AffectedUserCount)
			})
		})

		t.Run("確定済みユーザーのカートは両方に数える", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := uuidtestkit.NewTestFromSalt(t, "impact_active_product")
				userID := uuidtestkit.NewTestFromSalt(t, "impact_active_user")
				insertImpactProduct(ctx, t, drv, productID)
				insertDiscontinueUser(ctx, t, drv, userID, false)
				insertDiscontinueCart(
					ctx, t, drv, uuidtestkit.NewTestFromSalt(t, "impact_active_cart"), productID, &userID,
				)

				got, err := svc.EstimateDiscontinueImpact(ctx, productID)

				require.NoError(t, err)
				assert.Equal(t, int64(1), got.AffectedCartCount)
				assert.Equal(t, int64(1), got.AffectedUserCount)
			})
		})

		t.Run("進行中の購入がある場合、進行中件数に数える", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := uuidtestkit.NewTestFromSalt(t, "impact_inprogress_product")
				userID := uuidtestkit.NewTestFromSalt(t, "impact_inprogress_user")
				insertImpactProduct(ctx, t, drv, productID)
				insertDiscontinueUser(ctx, t, drv, userID, false)
				insertImpactPurchase(
					ctx, t, drv,
					uuidtestkit.NewTestFromSalt(t, "impact_inprogress_purchase"), productID, userID, seedShippedStatusID,
				)

				got, err := svc.EstimateDiscontinueImpact(ctx, productID)

				require.NoError(t, err)
				assert.Equal(t, int64(1), got.InProgressPurchaseCount)
			})
		})

		t.Run("終端ステータスの購入は進行中件数に数えない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := uuidtestkit.NewTestFromSalt(t, "impact_terminal_product")
				userID := uuidtestkit.NewTestFromSalt(t, "impact_terminal_user")
				insertImpactProduct(ctx, t, drv, productID)
				insertDiscontinueUser(ctx, t, drv, userID, false)
				insertImpactPurchase(
					ctx, t, drv,
					uuidtestkit.NewTestFromSalt(t, "impact_terminal_purchase"), productID, userID, seedCompletedStatusID,
				)

				got, err := svc.EstimateDiscontinueImpact(ctx, productID)

				require.NoError(t, err)
				assert.Zero(t, got.InProgressPurchaseCount)
			})
		})

		t.Run("他の商品のカートは母集団に入らない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				target := uuidtestkit.NewTestFromSalt(t, "impact_target_product")
				other := uuidtestkit.NewTestFromSalt(t, "impact_other_product")
				insertImpactProduct(ctx, t, drv, target)
				insertImpactProduct(ctx, t, drv, other)
				insertDiscontinueCart(ctx, t, drv, uuidtestkit.NewTestFromSalt(t, "impact_other_cart"), other, nil)

				got, err := svc.EstimateDiscontinueImpact(ctx, target)

				require.NoError(t, err)
				assert.Zero(t, got.AffectedCartCount)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			_, err := svc.EstimateDiscontinueImpact(ctx, uuidtestkit.NewTestFromSalt(t, "impact_canceled"))

			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}
