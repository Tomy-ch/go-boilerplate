package purchase

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"

	domainpurchase "go-boilerplate/internal/domain/purchase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// insertPurchaseDetailRow は、購入へ対象商品の明細を 1 件足します。
func insertPurchaseDetailRow(
	ctx context.Context, t *testing.T, db driver.DBTX, id, purchaseID, productID uuid.UUID,
) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price) VALUES ($1,$2,$3,1,100)",
		id, purchaseID, productID,
	)
	require.NoError(t, err)
}

func Test_repository_FindStatusesByProductID(t *testing.T) {
	t.Parallel()

	// 購入ステータスマスタ（seed 済み）。進行中と終端の両方を使う。
	const (
		seedShippedSID   = "5c9a1f3b-2d4e-4b6f-9c8a-3e0d1f2b4c5d"
		seedCompletedSID = "1904bf76-7d37-4288-bc15-359d2512ac91"
	)

	testDB := testkit.NewTestDB(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}

	orderedAt := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("対象商品を明細に持つ購入のステータスを進行中も終端も区別せず返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				userID := "fc000000-0000-4000-8000-000000000001"
				productID := mustParse(t, "fd000000-0000-4000-8000-000000000001")
				insertPurchaseOwner(ctx, t, drv, userID)
				insertProductRow(ctx, t, drv, productID, "廃番判定対象", 10)

				shipped := "fe000000-0000-4000-8000-000000000001"
				completed := "fe000000-0000-4000-8000-000000000002"
				insertPurchaseWithStatus(ctx, t, drv, shipped, userID, seedShippedSID, 100, orderedAt)
				insertPurchaseWithStatus(ctx, t, drv, completed, userID, seedCompletedSID, 200, orderedAt)
				insertPurchaseDetailRow(
					ctx, t, drv, mustParse(t, "ff000000-0000-4000-8000-000000000001"), mustParse(t, shipped), productID,
				)
				insertPurchaseDetailRow(
					ctx, t, drv, mustParse(t, "ff000000-0000-4000-8000-000000000002"), mustParse(t, completed), productID,
				)

				statuses, err := repo.FindStatusesByProductID(ctx, productID)

				require.NoError(t, err)
				assert.ElementsMatch(
					t,
					[]domainpurchase.Status{domainpurchase.StatusShipped, domainpurchase.StatusCompleted},
					statuses,
				)
			})
		})

		t.Run("同じステータスの購入が複数あっても1件へ畳む", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				userID := "fc000000-0000-4000-8000-000000000002"
				productID := mustParse(t, "fd000000-0000-4000-8000-000000000002")
				insertPurchaseOwner(ctx, t, drv, userID)
				insertProductRow(ctx, t, drv, productID, "重複ステータス対象", 10)

				first := "fe000000-0000-4000-8000-000000000003"
				second := "fe000000-0000-4000-8000-000000000004"
				insertPurchaseWithStatus(ctx, t, drv, first, userID, seedShippedSID, 100, orderedAt)
				insertPurchaseWithStatus(ctx, t, drv, second, userID, seedShippedSID, 200, orderedAt)
				insertPurchaseDetailRow(
					ctx, t, drv, mustParse(t, "ff000000-0000-4000-8000-000000000003"), mustParse(t, first), productID,
				)
				insertPurchaseDetailRow(
					ctx, t, drv, mustParse(t, "ff000000-0000-4000-8000-000000000004"), mustParse(t, second), productID,
				)

				statuses, err := repo.FindStatusesByProductID(ctx, productID)

				require.NoError(t, err)
				assert.Equal(t, []domainpurchase.Status{domainpurchase.StatusShipped}, statuses)
			})
		})

		t.Run("どの購入にも含まれない商品は空を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := mustParse(t, "fd000000-0000-4000-8000-000000000003")
				insertProductRow(ctx, t, drv, productID, "未購入商品", 10)

				statuses, err := repo.FindStatusesByProductID(ctx, productID)

				require.NoError(t, err)
				assert.Empty(t, statuses)
			})
		})

		t.Run("他の商品の明細しか持たない購入のステータスは返さない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				userID := "fc000000-0000-4000-8000-000000000003"
				target := mustParse(t, "fd000000-0000-4000-8000-000000000004")
				other := mustParse(t, "fd000000-0000-4000-8000-000000000005")
				insertPurchaseOwner(ctx, t, drv, userID)
				insertProductRow(ctx, t, drv, target, "対象商品", 10)
				insertProductRow(ctx, t, drv, other, "別商品", 10)

				purchaseID := "fe000000-0000-4000-8000-000000000005"
				insertPurchaseWithStatus(ctx, t, drv, purchaseID, userID, seedShippedSID, 100, orderedAt)
				insertPurchaseDetailRow(
					ctx, t, drv, mustParse(t, "ff000000-0000-4000-8000-000000000005"), mustParse(t, purchaseID), other,
				)

				statuses, err := repo.FindStatusesByProductID(ctx, target)

				require.NoError(t, err)
				assert.Empty(t, statuses)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			statuses, err := repo.FindStatusesByProductID(ctx, mustParse(t, "fd000000-0000-4000-8000-00000000000f"))

			assert.Nil(t, statuses)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})

		t.Run("ステータスマスタにドメインの知らないcodeがある場合、再構築のエラーを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				userID := "fc000000-0000-4000-8000-00000000000e"
				productID := mustParse(t, "fd000000-0000-4000-8000-00000000000e")
				insertPurchaseOwner(ctx, t, drv, userID)
				insertProductRow(ctx, t, drv, productID, "未知ステータス対象", 10)

				// 業務キーの解決はドメインが持つため、永続化側に未知の code があれば再構築で弾かれる。
				unknownStatusID := "fb000000-0000-4000-8000-0000000000ee"
				_, err := drv.Exec(ctx,
					"INSERT INTO purchase_statuses (id, code, name, sort_key) VALUES ($1,$2,$3,$4)",
					unknownStatusID, 99, "未知", 99,
				)
				require.NoError(t, err)

				purchaseID := "fe000000-0000-4000-8000-00000000000e"
				insertPurchaseWithStatus(ctx, t, drv, purchaseID, userID, unknownStatusID, 100, orderedAt)
				insertPurchaseDetailRow(
					ctx, t, drv, mustParse(t, "ff000000-0000-4000-8000-00000000000e"), mustParse(t, purchaseID), productID,
				)

				statuses, serr := repo.FindStatusesByProductID(ctx, productID)

				assert.Nil(t, statuses)
				require.ErrorIs(t, serr, domainpurchase.ErrInvalidStatusID)
			})
		})
	})
}
