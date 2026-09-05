package product

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/domain/coupon"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/product/command"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 既存 seed の FK 対象。
const (
	seedStatusInStock = "093170fb-83a2-4864-a2b3-53236eaf3597"
	seedCategory      = "5dd52d84-78eb-4a52-ba0b-2e11c95c2af2"
	seedPrefecture    = "a03aaec4-3bd6-4bfb-8e47-2fbfa026d344"
)

// insertRecipientUser は、クーポンの受給者となる確定済みユーザーを挿入します。
// deleted が true のときは退会済みとして挿入し、受給者から外れることを確かめられるようにします。
func insertRecipientUser(ctx context.Context, t *testing.T, db driver.DBTX, id uuid.UUID, deleted bool) {
	t.Helper()
	deletedAt := "NULL"
	if deleted {
		deletedAt = "NOW()"
	}
	_, err := db.Exec(ctx,
		"INSERT INTO users "+
			"(id, first_name, last_name, email, phone, prefecture_id, city, street, postal_code, deleted_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,"+deletedAt+")",
		id, "Recipient", "User", "recipient-"+id.String()+"@example.com", "0000000000",
		seedPrefecture, "City", "Street", "000-0000",
	)
	require.NoError(t, err)
}

// insertCartWithItem は、対象商品の明細を 1 件持つカートを挿入します。owner が nil ならゲストのカートです。
func insertCartWithItem(
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

// insertDiscontinuedProduct は、廃番対象となる商品を挿入します。
func insertDiscontinuedProduct(ctx context.Context, t *testing.T, db driver.DBTX, id uuid.UUID) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO products "+
			"(id, name, description, price, quantity, stock_warning_threshold, status_id, category_id) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
		id, "廃番対象-"+id.String(), nil, 100, 10, nil, seedStatusInStock, seedCategory,
	)
	require.NoError(t, err)
}

// newIssueParams は、既定の発行条件（定率 10% / カテゴリ限定 / 30 日有効）を組み立てます。
func newIssueParams(t *testing.T, productID uuid.UUID) command.IssueDiscontinuationCouponsParams {
	t.Helper()
	discount, err := coupon.NewRateDiscount(decimaltestkit.MustParse(t, "0.10"))
	require.NoError(t, err)

	categoryID, err := uuid.Parse(seedCategory)
	require.NoError(t, err)
	scope, err := coupon.NewCategoryScope(categoryID)
	require.NoError(t, err)

	issuedAt := time.Now()

	return command.IssueDiscontinuationCouponsParams{
		ProductID: productID,
		Scope:     scope,
		Discount:  discount,
		ExpiresAt: issuedAt.Add(30 * 24 * time.Hour),
		IssuedAt:  issuedAt,
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("依存を注入したコマンドサービス実装を生成する", func(t *testing.T) {
			t.Parallel()

			testDB := testkit.NewTestDB(t)

			svc, ok := New(testDB, observability.NewNoopTracerFactory(t)).(*commandService)
			require.True(t, ok)
			assert.Equal(t, testDB, svc.db)
			assert.NotNil(t, svc.tracer)
		})
	})
}

func Test_commandService_IssueDiscontinuationCoupons(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	txm := testkit.NewTestTransactionRunner(t)
	svc := &commandService{db: testDB, tracer: observability.NewMockInfraLayerTracer(t)}

	countCoupons := func(ctx context.Context, t *testing.T, db driver.DBTX, categoryID uuid.UUID) int {
		t.Helper()
		var count int
		row := db.QueryRow(ctx, "SELECT COUNT(*) FROM coupons WHERE scope_target_id = $1", categoryID)
		require.NoError(t, row.Scan(&count))

		return count
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("確定済みユーザーへ1人1枚を発行し件数を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := uuidtestkit.NewTestFromSalt(t, "cs_issue_product")
				userA := uuidtestkit.NewTestFromSalt(t, "cs_issue_user_a")
				userB := uuidtestkit.NewTestFromSalt(t, "cs_issue_user_b")
				insertDiscontinuedProduct(ctx, t, drv, productID)
				insertRecipientUser(ctx, t, drv, userA, false)
				insertRecipientUser(ctx, t, drv, userB, false)
				insertCartWithItem(ctx, t, drv, uuidtestkit.NewTestFromSalt(t, "cs_issue_cart_a"), productID, &userA)
				insertCartWithItem(ctx, t, drv, uuidtestkit.NewTestFromSalt(t, "cs_issue_cart_b"), productID, &userB)

				params := newIssueParams(t, productID)
				got, err := svc.IssueDiscontinuationCoupons(ctx, params)

				require.NoError(t, err)
				assert.Equal(t, int64(2), got.AffectedCartCount)
				assert.Equal(t, int64(2), got.AffectedUserCount)
				assert.Equal(t, int64(2), got.IssuedCouponCount)
				assert.Equal(t, 2, countCoupons(ctx, t, drv, *params.Scope.TargetID()))
			})
		})

		t.Run("発行したクーポンは定率かつカテゴリ限定で受給者ごとに別のidを持つ", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := uuidtestkit.NewTestFromSalt(t, "cs_shape_product")
				userID := uuidtestkit.NewTestFromSalt(t, "cs_shape_user")
				insertDiscontinuedProduct(ctx, t, drv, productID)
				insertRecipientUser(ctx, t, drv, userID, false)
				insertCartWithItem(ctx, t, drv, uuidtestkit.NewTestFromSalt(t, "cs_shape_cart"), productID, &userID)

				params := newIssueParams(t, productID)
				_, err := svc.IssueDiscontinuationCoupons(ctx, params)
				require.NoError(t, err)

				var (
					discountKind int16
					scopeKind    int16
					target       uuid.UUID
					owner        uuid.UUID
				)
				row := drv.QueryRow(ctx,
					"SELECT discount_kind, scope_kind, scope_target_id, user_id FROM coupons WHERE scope_target_id = $1",
					*params.Scope.TargetID(),
				)
				require.NoError(t, row.Scan(&discountKind, &scopeKind, &target, &owner))

				assert.Equal(t, coupon.DiscountKindRate.Code(), int(discountKind))
				assert.Equal(t, coupon.ScopeKindCategory.Code(), int(scopeKind))
				assert.Equal(t, *params.Scope.TargetID(), target)
				assert.Equal(t, userID, owner)
			})
		})

		t.Run("ゲストのカートは影響件数に数えるが受給者にならず発行もしない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := uuidtestkit.NewTestFromSalt(t, "cs_guest_product")
				insertDiscontinuedProduct(ctx, t, drv, productID)
				insertCartWithItem(ctx, t, drv, uuidtestkit.NewTestFromSalt(t, "cs_guest_cart"), productID, nil)

				params := newIssueParams(t, productID)
				got, err := svc.IssueDiscontinuationCoupons(ctx, params)

				require.NoError(t, err)
				assert.Equal(t, int64(1), got.AffectedCartCount)
				assert.Zero(t, got.AffectedUserCount)
				assert.Zero(t, got.IssuedCouponCount)
				assert.Equal(t, 0, countCoupons(ctx, t, drv, *params.Scope.TargetID()))
			})
		})

		t.Run("退会済みユーザーは受給者にならない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := uuidtestkit.NewTestFromSalt(t, "cs_deleted_product")
				userID := uuidtestkit.NewTestFromSalt(t, "cs_deleted_user")
				insertDiscontinuedProduct(ctx, t, drv, productID)
				insertRecipientUser(ctx, t, drv, userID, true)
				insertCartWithItem(ctx, t, drv, uuidtestkit.NewTestFromSalt(t, "cs_deleted_cart"), productID, &userID)

				params := newIssueParams(t, productID)
				got, err := svc.IssueDiscontinuationCoupons(ctx, params)

				require.NoError(t, err)
				assert.Equal(t, int64(1), got.AffectedCartCount)
				assert.Zero(t, got.IssuedCouponCount)
			})
		})

		t.Run("同じユーザーが複数のカートに入れていても1枚しか発行しない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := uuidtestkit.NewTestFromSalt(t, "cs_dup_product")
				userID := uuidtestkit.NewTestFromSalt(t, "cs_dup_user")
				insertDiscontinuedProduct(ctx, t, drv, productID)
				insertRecipientUser(ctx, t, drv, userID, false)
				insertCartWithItem(ctx, t, drv, uuidtestkit.NewTestFromSalt(t, "cs_dup_cart_a"), productID, &userID)
				insertCartWithItem(ctx, t, drv, uuidtestkit.NewTestFromSalt(t, "cs_dup_cart_b"), productID, &userID)

				params := newIssueParams(t, productID)
				got, err := svc.IssueDiscontinuationCoupons(ctx, params)

				require.NoError(t, err)
				assert.Equal(t, int64(2), got.AffectedCartCount)
				assert.Equal(t, int64(1), got.AffectedUserCount)
				assert.Equal(t, int64(1), got.IssuedCouponCount)
			})
		})

		t.Run("同じ受給者でも1枚ずつ別のidを採番する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := uuidtestkit.NewTestFromSalt(t, "cs_ids_product")
				userA := uuidtestkit.NewTestFromSalt(t, "cs_ids_user_a")
				userB := uuidtestkit.NewTestFromSalt(t, "cs_ids_user_b")
				insertDiscontinuedProduct(ctx, t, drv, productID)
				insertRecipientUser(ctx, t, drv, userA, false)
				insertRecipientUser(ctx, t, drv, userB, false)
				insertCartWithItem(ctx, t, drv, uuidtestkit.NewTestFromSalt(t, "cs_ids_cart_a"), productID, &userA)
				insertCartWithItem(ctx, t, drv, uuidtestkit.NewTestFromSalt(t, "cs_ids_cart_b"), productID, &userB)

				params := newIssueParams(t, productID)
				_, err := svc.IssueDiscontinuationCoupons(ctx, params)
				require.NoError(t, err)

				var distinct int
				row := drv.QueryRow(ctx,
					"SELECT COUNT(DISTINCT id) FROM coupons WHERE scope_target_id = $1", *params.Scope.TargetID())
				require.NoError(t, row.Scan(&distinct))
				assert.Equal(t, 2, distinct)
			})
		})

		t.Run("どのカートにも入っていない商品は発行を行わない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := uuidtestkit.NewTestFromSalt(t, "cs_untouched_product")
				insertDiscontinuedProduct(ctx, t, drv, productID)

				params := newIssueParams(t, productID)
				got, err := svc.IssueDiscontinuationCoupons(ctx, params)

				require.NoError(t, err)
				assert.Zero(t, got.AffectedCartCount)
				assert.Zero(t, got.AffectedUserCount)
				assert.Zero(t, got.IssuedCouponCount)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効期限が発行日時以前の場合、集約の検証で弾き1行も挿入しない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := uuidtestkit.NewTestFromSalt(t, "cs_invalid_product")
				userID := uuidtestkit.NewTestFromSalt(t, "cs_invalid_user")
				insertDiscontinuedProduct(ctx, t, drv, productID)
				insertRecipientUser(ctx, t, drv, userID, false)
				insertCartWithItem(ctx, t, drv, uuidtestkit.NewTestFromSalt(t, "cs_invalid_cart"), productID, &userID)

				params := newIssueParams(t, productID)
				params.ExpiresAt = params.IssuedAt

				_, err := svc.IssueDiscontinuationCoupons(ctx, params)

				require.ErrorIs(t, err, coupon.ErrInvalidExpiresAt)
				assert.Equal(t, 0, countCoupons(ctx, t, drv, *params.Scope.TargetID()))
			})
		})
	})
}
