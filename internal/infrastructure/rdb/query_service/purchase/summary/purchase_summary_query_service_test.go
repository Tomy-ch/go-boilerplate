package summary

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

// 既存 seed 由来の FK 対象。
const (
	seedUserA          = "550e8400-e29b-41d4-a716-446655440000" // 集計対象の購入所有者
	seedUserB          = "a95a2dd3-2b37-4def-8041-23d2138faccc" // 別ユーザー（他人）
	seedUnprocessedSID = "a66c996c-86b2-41d8-9bdd-9b685fb7c47d" // 購入ステータス（未処理・sort_key=1）
	seedPaidSID        = "4b8f0e2a-1c3d-4a5e-8b7f-2d9c0e1a3b4c" // 購入ステータス（支払い済み・sort_key=7）
	seedCanceledSID    = "e9d72547-adfe-48d9-9037-bd1f55d4158b" // 購入ステータス（キャンセル・sort_key=6）
)

func mustParse(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	require.NoError(t, err)
	return id
}

// canceledContext は、キャンセル済みのコンテキストを返します。
func canceledContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	return ctx
}

// insertPurchase は、指定ユーザー・ステータス・合計金額の購入を 1 件挿入します。
func insertPurchase(ctx context.Context, t *testing.T, db driver.DBTX, userID, statusID uuid.UUID, code string, totalAmount int64) {
	t.Helper()
	purchaseID, err := uuid.New()
	require.NoError(t, err)
	_, err = db.Exec(ctx,
		"INSERT INTO purchases (id, code, user_id, status_id, subtotal_amount, tax_amount, shipping_fee, total_amount) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
		purchaseID, code, userID, statusID, totalAmount, 0, 0, totalAmount,
	)
	require.NoError(t, err)
}

func Test_service_SummarizeByUserID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	svc := &service{tracer: lt, db: testDB}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("本人の購入をステータス単位に集計しマスタの表示順で返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				userA := mustParse(t, seedUserA)
				// 表示順（sort_key）が後のステータスから挿入し、挿入順ではなくマスタ順で並ぶことを確認する。
				insertPurchase(ctx, t, drv, userA, mustParse(t, seedPaidSID), "sm-paid-1", 300)
				insertPurchase(ctx, t, drv, userA, mustParse(t, seedUnprocessedSID), "sm-unproc-1", 150)
				insertPurchase(ctx, t, drv, userA, mustParse(t, seedUnprocessedSID), "sm-unproc-2", 250)

				got, err := svc.SummarizeByUserID(ctx, userA)
				require.NoError(t, err)
				require.Len(t, got, 2)

				assert.Equal(t, mustParse(t, seedUnprocessedSID), got[0].StatusID)
				assert.Equal(t, "未処理", got[0].StatusName)
				assert.Equal(t, int64(2), got[0].Count)
				assert.Equal(t, int64(400), got[0].TotalAmount)

				assert.Equal(t, mustParse(t, seedPaidSID), got[1].StatusID)
				assert.Equal(t, "支払い済み", got[1].StatusName)
				assert.Equal(t, int64(1), got[1].Count)
				assert.Equal(t, int64(300), got[1].TotalAmount)
			})
		})

		t.Run("他ユーザーの購入は集計に混入しない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				userA := mustParse(t, seedUserA)
				userB := mustParse(t, seedUserB)
				insertPurchase(ctx, t, drv, userA, mustParse(t, seedUnprocessedSID), "sm-own-1", 100)
				insertPurchase(ctx, t, drv, userB, mustParse(t, seedUnprocessedSID), "sm-other-1", 999)
				insertPurchase(ctx, t, drv, userB, mustParse(t, seedPaidSID), "sm-other-2", 999)

				got, err := svc.SummarizeByUserID(ctx, userA)
				require.NoError(t, err)
				require.Len(t, got, 1)
				assert.Equal(t, int64(1), got[0].Count)
				assert.Equal(t, int64(100), got[0].TotalAmount)
			})
		})

		t.Run("キャンセル済みの購入も集計対象に含める", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				userA := mustParse(t, seedUserA)
				insertPurchase(ctx, t, drv, userA, mustParse(t, seedCanceledSID), "sm-canceled-1", 400)

				got, err := svc.SummarizeByUserID(ctx, userA)
				require.NoError(t, err)
				require.Len(t, got, 1)
				assert.Equal(t, "キャンセル", got[0].StatusName)
				assert.Equal(t, int64(1), got[0].Count)
				assert.Equal(t, int64(400), got[0].TotalAmount)
			})
		})

		t.Run("購入が1件もないユーザーは空スライスを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				got, err := svc.SummarizeByUserID(ctx, mustParse(t, seedUserB))
				require.NoError(t, err)
				assert.Empty(t, got)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			got, err := svc.SummarizeByUserID(canceledContext(t), mustParse(t, seedUserA))
			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.Nil(t, got)
		})
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("依存を注入したクエリサービス実装を生成する", func(t *testing.T) {
			t.Parallel()

			testDB := testkit.NewTestDB(t)
			tf := observability.NewNoopTracerFactory(t)

			svc, ok := New(testDB, tf).(*service)
			require.True(t, ok)
			assert.Equal(t, testDB, svc.db)
			assert.NotNil(t, svc.tracer)
		})
	})
}
