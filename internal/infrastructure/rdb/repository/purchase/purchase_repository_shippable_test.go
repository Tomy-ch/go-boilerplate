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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// statusPaidID は、購入ステータスマスタ（migration 済み）の「支払い済み」です。発送可能の唯一の状態です。
	statusPaidID = "4b8f0e2a-1c3d-4a5e-8b7f-2d9c0e1a3b4c"
	// statusShippedID は、購入ステータスマスタ（migration 済み）の「発送済み」です。
	// 発送可能でない購入が絞り込みから漏れないことの検証に用います。
	statusShippedID = "5c9a1f3b-2d4e-4b6f-9c8a-3e0d1f2b4c5d"
)

// shippableProductIDs は、明細の FK 制約（product_id → products.id）を満たす seed 済み商品です。
// 集約は同一購入内での商品 ID 重複を拒むため、明細を複数持つ購入には別々の商品を割り当てます。
var shippableProductIDs = []string{
	"f211d877-36ef-41dc-aea0-72968a7d8f7e",
	"5266f905-1af4-477a-bc7c-e69729e22a2b",
}

// insertShippableUser は、購入の FK 制約（user_id → users.id）を満たすためのユーザーを挿入します。
func insertShippableUser(ctx context.Context, t *testing.T, db driver.DBTX, id string) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO users "+
			"(id, first_name, last_name, email, phone, prefecture_id, city, street, postal_code, created_at, updated_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW())",
		id,
		"Shippable",
		"User",
		"shippable-"+id+"@example.com",
		"000-000-0000",
		"a03aaec4-3bd6-4bfb-8e47-2fbfa026d344", // 既存 seed の都道府県ID
		"City",
		"Street",
		"000-0000",
	)
	require.NoError(t, err)
}

// insertShippablePurchase は、明細 1 件を伴う購入を挿入します。集約の再構築は明細が空だと失敗するため、
// また支払い済みは paid_at を必須とするため、両方をここで満たします。
func insertShippablePurchase(
	ctx context.Context, t *testing.T, db driver.DBTX,
	id, userID, statusID string, orderedAt time.Time, detailIDs ...string,
) {
	t.Helper()

	paidAt := orderedAt.Add(time.Minute)
	shippedAt := any(nil)
	if statusID == statusShippedID {
		shippedAt = orderedAt.Add(time.Hour)
	}
	_, err := db.Exec(ctx,
		"INSERT INTO purchases "+
			"(id, code, user_id, status_id, subtotal_amount, tax_amount, shipping_fee, total_amount, "+
			"ordered_at, paid_at, shipped_at, created_at, updated_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW(),NOW())",
		id, "code-"+id, userID, statusID, 1000, 100, 50, 1150, orderedAt, paidAt, shippedAt,
	)
	require.NoError(t, err)

	require.LessOrEqual(t, len(detailIDs), len(shippableProductIDs))
	for i, detailID := range detailIDs {
		_, derr := db.Exec(ctx,
			"INSERT INTO purchase_details "+
				"(id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) "+
				"VALUES ($1,$2,$3,$4,$5,NOW(),NOW())",
			detailID, id, shippableProductIDs[i], 2, "500",
		)
		require.NoError(t, derr)
	}
}

func Test_repository_FindShippable(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{tracer: lt, db: testDB}

	// seed データより必ず後ろ（新しい）に来るよう十分未来を基準にする。
	base := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)

	buyer := "eeeeeeee-1111-4000-8000-000000000001"
	older := "eeeeeeee-2222-4000-8000-000000000001"
	newer := "eeeeeeee-2222-4000-8000-000000000002"
	shipped := "eeeeeeee-2222-4000-8000-000000000003"

	mustParse := func(s string) uuid.UUID {
		id, err := uuid.Parse(s)
		require.NoError(t, err)
		return id
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("支払い済みの購入だけを注文日時の古い順に明細込みで返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertShippableUser(ctx, t, drv, buyer)
				insertShippablePurchase(ctx, t, drv, newer, buyer, statusPaidID, base.Add(time.Hour),
					"eeeeeeee-3333-4000-8000-000000000021")
				insertShippablePurchase(ctx, t, drv, older, buyer, statusPaidID, base,
					"eeeeeeee-3333-4000-8000-000000000011", "eeeeeeee-3333-4000-8000-000000000012")
				insertShippablePurchase(ctx, t, drv, shipped, buyer, statusShippedID, base.Add(-time.Hour),
					"eeeeeeee-3333-4000-8000-000000000031")

				got, err := repo.FindShippable(ctx, 10)
				require.NoError(t, err)

				// 発送済みは絞り込みから外れ、支払い済みの 2 件が古い順に並ぶ。
				require.Len(t, got, 2)
				assert.Equal(t, mustParse(older), got[0].ID())
				assert.Equal(t, mustParse(newer), got[1].ID())
				// 明細は購入ごとに正しく振り分けられる（一括取得の取り違えを弾く）。
				assert.Len(t, got[0].Details(), 2)
				assert.Len(t, got[1].Details(), 1)
				// 返した行は発送可能の述語を満たす。
				assert.True(t, got[0].IsShippable())
				assert.True(t, got[1].IsShippable())
			})
		})

		t.Run("limitで件数を絞る場合、古い側から打ち切る", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertShippableUser(ctx, t, drv, buyer)
				insertShippablePurchase(ctx, t, drv, older, buyer, statusPaidID, base,
					"eeeeeeee-3333-4000-8000-000000000011")
				insertShippablePurchase(ctx, t, drv, newer, buyer, statusPaidID, base.Add(time.Hour),
					"eeeeeeee-3333-4000-8000-000000000021")

				got, err := repo.FindShippable(ctx, 1)
				require.NoError(t, err)

				require.Len(t, got, 1)
				assert.Equal(t, mustParse(older), got[0].ID())
			})
		})

		t.Run("注文日時が同時刻の場合、購入IDの昇順で並びlimitもその順で打ち切る", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertShippableUser(ctx, t, drv, buyer)
				// 同時刻の 2 件。SQL のタイブレークが崩れると limit 境界で拾う 1 件が入れ替わる。
				insertShippablePurchase(ctx, t, drv, older, buyer, statusPaidID, base,
					"eeeeeeee-3333-4000-8000-000000000011")
				insertShippablePurchase(ctx, t, drv, newer, buyer, statusPaidID, base,
					"eeeeeeee-3333-4000-8000-000000000021")

				got, err := repo.FindShippable(ctx, 1)
				require.NoError(t, err)

				require.Len(t, got, 1)
				assert.Equal(t, mustParse(older), got[0].ID())
			})
		})

		t.Run("発送可能な購入が無い場合、空を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertShippableUser(ctx, t, drv, buyer)
				insertShippablePurchase(ctx, t, drv, shipped, buyer, statusShippedID, base,
					"eeeeeeee-3333-4000-8000-000000000031")

				got, err := repo.FindShippable(ctx, 10)
				require.NoError(t, err)

				assert.Empty(t, got)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化される", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			got, err := repo.FindShippable(ctx, 10)
			assert.Nil(t, got)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})

		t.Run("limitが負数の場合、ErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				// 負数 LIMIT は PostgreSQL の 2201W（map 未定義）となり ErrInternal へ写像される。
				got, err := repo.FindShippable(ctx, -1)
				require.Nil(t, got)
				require.ErrorIs(t, err, apperror.ErrInternal)
			})
		})

		t.Run("DB行がドメイン不変条件に違反する場合はErrInternalへ正規化する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertShippableUser(ctx, t, drv, buyer)
				// 支払い済み status は paidAt を必須とする。paid_at を欠く行は Reconstruct の
				// 不変条件に違反する破損行で、再構築の失敗が ErrInternal へ写像される。
				_, err := drv.Exec(ctx,
					"INSERT INTO purchases "+
						"(id, code, user_id, status_id, subtotal_amount, tax_amount, shipping_fee, total_amount, "+
						"ordered_at, created_at, updated_at) "+
						"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW())",
					older, "code-"+older, buyer, statusPaidID, 1000, 100, 50, 1150, base,
				)
				require.NoError(t, err)
				_, err = drv.Exec(ctx,
					"INSERT INTO purchase_details "+
						"(id, purchase_id, product_id, quantity, unit_price, created_at, updated_at) "+
						"VALUES ($1,$2,$3,$4,$5,NOW(),NOW())",
					"eeeeeeee-3333-4000-8000-000000000091", older, shippableProductIDs[0], 2, "500",
				)
				require.NoError(t, err)

				got, err := repo.FindShippable(ctx, 10)
				require.Nil(t, got)
				require.ErrorIs(t, err, apperror.ErrInternal)
			})
		})
	})
}
