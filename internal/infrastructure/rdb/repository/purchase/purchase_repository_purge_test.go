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

// insertPurchaseForUser は、指定ユーザーの購入を 1 件挿入するヘルパーです（明細は伴いません）。
func insertPurchaseForUser(ctx context.Context, t *testing.T, db driver.DBTX, userID uuid.UUID, seed string) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO purchases (id, code, user_id, status_id, subtotal_amount, tax_amount, shipping_fee, total_amount) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
		mustParse(t, seed), "purge-code-"+seed, userID, mustParse(t, seedUnprocessedSID), 1000, 100, 0, 1100,
	)
	require.NoError(t, err)
}

func Test_repository_FindUserIDsWithPurchases(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{tracer: lt, db: testDB}

	// 購入を持たないユーザー（seed の Bob Brown は購入 seed を持たない）。
	const seedUserWithoutPurchaseID = "090f5b51-37ac-4413-b326-1709ae4661f4"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入を持つユーザーIDだけを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				withPurchase := mustParse(t, seedUserID)
				withoutPurchase := mustParse(t, seedUserWithoutPurchaseID)
				insertPurchaseForUser(ctx, t, drv, withPurchase, "dd000000-0000-4000-8000-000000000001")

				got, err := repo.FindUserIDsWithPurchases(ctx, []uuid.UUID{withPurchase, withoutPurchase})
				require.NoError(t, err)

				assert.Equal(t, []uuid.UUID{withPurchase}, got)
			})
		})

		t.Run("同一ユーザーが複数の購入を持っても1件に重複排除する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				withPurchase := mustParse(t, seedUserID)
				insertPurchaseForUser(ctx, t, drv, withPurchase, "dd000000-0000-4000-8000-000000000002")
				insertPurchaseForUser(ctx, t, drv, withPurchase, "dd000000-0000-4000-8000-000000000003")

				got, err := repo.FindUserIDsWithPurchases(ctx, []uuid.UUID{withPurchase})
				require.NoError(t, err)

				assert.Equal(t, []uuid.UUID{withPurchase}, got)
			})
		})

		t.Run("購入を持つユーザーが1人もいなければ空を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.FindUserIDsWithPurchases(ctx, []uuid.UUID{mustParse(t, seedUserWithoutPurchaseID)})
				require.NoError(t, err)

				assert.Empty(t, got)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			got, err := repo.FindUserIDsWithPurchases(ctx, []uuid.UUID{mustParse(t, seedUserWithoutPurchaseID)})
			assert.Nil(t, got)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}
