package dashboard

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/dashboard/query"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 既存 seed の FK 対象（purchases の user_id / status_id）。
const (
	seedUserID         = "550e8400-e29b-41d4-a716-446655440000"
	seedUnprocessedSID = "a66c996c-86b2-41d8-9bdd-9b685fb7c47d"
	seedCompletedSID   = "1904bf76-7d37-4288-bc15-359d2512ac91"
	seedCanceledSID    = "e9d72547-adfe-48d9-9037-bd1f55d4158b"
)

// testLoc は、テストが組み立てる集計対象期間の境界を表すロケーションです。期間の解決自体は usecase 層の
// 責務なので、ここでは解決済みの境界を作るためだけに用います。
var testLoc = time.FixedZone("Asia/Tokyo", 9*60*60)

// fixedNow は、テストが投入する購入の注文日時です。testLoc の暦日で 2026-07-15 に落ちる時刻を UTC で保持し、
// todayWindow が組み立てる区間に必ず含まれるようにします。
var fixedNow = time.Date(2026, time.July, 15, 12, 0, 0, 0, testLoc).UTC()

func canceledContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	return ctx
}

func mustParse(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	require.NoError(t, err)
	return id
}

func newService(t *testing.T, db driver.DatabaseDriver) *service {
	t.Helper()
	return &service{
		db:     db,
		tracer: observability.NewMockInfraLayerTracer(t),
	}
}

// clearSeededPurchases は、呼び出したトランザクション内で購入と購入明細を空にします。
// 売上集計は購入テーブル全体を期間で切って集計するため、seed が投入する購入履歴が期間に入ると
// 金額と件数の期待値が崩れます。削除はロールバックされるので、コミット済みの seed は失われません。
func clearSeededPurchases(ctx context.Context, t *testing.T, db driver.DBTX) {
	t.Helper()
	_, err := db.Exec(ctx, "DELETE FROM purchase_details")
	require.NoError(t, err)
	_, err = db.Exec(ctx, "DELETE FROM purchases")
	require.NoError(t, err)
}

// insertPurchase は、集計対象となる購入を注文日時・合計金額・ステータス・キャンセル日時（nil 可）指定で挿入します。
func insertPurchase(
	ctx context.Context, t *testing.T, db driver.DBTX,
	id uuid.UUID, orderedAt time.Time, totalAmount int64, statusID string, canceledAt *time.Time,
) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO purchases "+
			"(id, code, user_id, status_id, subtotal_amount, tax_amount, shipping_fee, total_amount, ordered_at, canceled_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)",
		id, "dash-"+id.String(), mustParse(t, seedUserID), mustParse(t, statusID),
		totalAmount, 0, 0, totalAmount, orderedAt, canceledAt,
	)
	require.NoError(t, err)
}

// todayWindow は、fixedNow が属する暦日 1 日分（testLoc 基準）を表す解決済みの集計対象期間を返します。
// 期間の解決自体は usecase 層の責務なので、ここでは解決済みの境界を直接組み立てます。
func todayWindow() query.Window {
	start := time.Date(2026, time.July, 15, 0, 0, 0, 0, testLoc)
	return query.Window{After: start, Before: start.AddDate(0, 0, 1)}
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡したドライバとinfra層トレーサーを保持した実装を返す", func(t *testing.T) {
			t.Parallel()

			testDB := testkit.NewTestDB(t)
			tf := observability.NewNoopTracerFactory(t)

			expected := &service{
				db:     testDB,
				tracer: tf.Infra(),
			}
			actual := New(testDB, tf)

			assert.Equal(t, expected, actual)
		})
	})
}

func Test_service_SummarizeSales(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	txm := testkit.NewTestTransactionRunner(t)
	svc := newService(t, testDB)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期間内の購入の合計金額と件数を返し未払いも含める", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				clearSeededPurchases(ctx, t, drv)
				insertPurchase(ctx, t, drv, mustParse(t, "b1000000-0000-4000-8000-000000000001"),
					fixedNow, 1500, seedUnprocessedSID, nil)
				insertPurchase(ctx, t, drv, mustParse(t, "b1000000-0000-4000-8000-000000000002"),
					fixedNow, 2500, seedCompletedSID, nil)

				got, err := svc.SummarizeSales(ctx, todayWindow())
				require.NoError(t, err)

				assert.Equal(t, int64(4000), got.Amount)
				assert.Equal(t, int64(2), got.Count)
			})
		})

		t.Run("キャンセル済みの購入は合計金額と件数の双方から除外する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				clearSeededPurchases(ctx, t, drv)
				canceledAt := fixedNow
				insertPurchase(ctx, t, drv, mustParse(t, "b2000000-0000-4000-8000-000000000001"),
					fixedNow, 1000, seedUnprocessedSID, nil)
				insertPurchase(ctx, t, drv, mustParse(t, "b2000000-0000-4000-8000-000000000002"),
					fixedNow, 9999, seedCanceledSID, &canceledAt)

				got, err := svc.SummarizeSales(ctx, todayWindow())
				require.NoError(t, err)

				assert.Equal(t, int64(1000), got.Amount)
				assert.Equal(t, int64(1), got.Count)
			})
		})

		t.Run("期間の下限は境界時刻を含み上限は含まない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				clearSeededPurchases(ctx, t, drv)
				startOfToday := time.Date(2026, time.July, 15, 0, 0, 0, 0, testLoc)
				insertPurchase(ctx, t, drv, mustParse(t, "b3000000-0000-4000-8000-000000000001"),
					startOfToday, 100, seedUnprocessedSID, nil)
				insertPurchase(ctx, t, drv, mustParse(t, "b3000000-0000-4000-8000-000000000002"),
					startOfToday.AddDate(0, 0, 1), 200, seedUnprocessedSID, nil)
				insertPurchase(ctx, t, drv, mustParse(t, "b3000000-0000-4000-8000-000000000003"),
					startOfToday.Add(-time.Nanosecond), 400, seedUnprocessedSID, nil)

				got, err := svc.SummarizeSales(ctx, todayWindow())
				require.NoError(t, err)

				assert.Equal(t, int64(100), got.Amount)
				assert.Equal(t, int64(1), got.Count)
			})
		})

		t.Run("range指定は開始日と終了日の暦日を両端とも集計対象に含める", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				clearSeededPurchases(ctx, t, drv)
				from := time.Date(2026, time.June, 1, 0, 0, 0, 0, testLoc)
				to := time.Date(2026, time.June, 3, 0, 0, 0, 0, testLoc)
				insertPurchase(ctx, t, drv, mustParse(t, "b4000000-0000-4000-8000-000000000001"),
					from, 100, seedUnprocessedSID, nil)
				insertPurchase(ctx, t, drv, mustParse(t, "b4000000-0000-4000-8000-000000000002"),
					to.Add(23*time.Hour), 200, seedUnprocessedSID, nil)
				insertPurchase(ctx, t, drv, mustParse(t, "b4000000-0000-4000-8000-000000000003"),
					to.AddDate(0, 0, 1), 400, seedUnprocessedSID, nil)

				got, err := svc.SummarizeSales(ctx, query.Window{After: from, Before: to.AddDate(0, 0, 1)})
				require.NoError(t, err)

				assert.Equal(t, int64(300), got.Amount)
				assert.Equal(t, int64(2), got.Count)
			})
		})

		t.Run("期間内の購入が無い場合はゼロ値を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				clearSeededPurchases(ctx, t, driver.New(ctx, testDB))

				got, err := svc.SummarizeSales(ctx, todayWindow())
				require.NoError(t, err)

				assert.Zero(t, got.Amount)
				assert.Zero(t, got.Count)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			got, err := svc.SummarizeSales(canceledContext(t), todayWindow())
			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.Zero(t, got.Amount)
		})
	})
}

func Test_service_CountPurchasesByStatus(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	txm := testkit.NewTestTransactionRunner(t)
	svc := newService(t, testDB)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ステータス単位に件数を集計し表示順の昇順で名称と共に返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				clearSeededPurchases(ctx, t, drv)
				// 表示順は 未処理(1) < 完了(5) のため、挿入順と逆になる。
				insertPurchase(ctx, t, drv, mustParse(t, "c1000000-0000-4000-8000-000000000001"),
					fixedNow, 100, seedCompletedSID, nil)
				insertPurchase(ctx, t, drv, mustParse(t, "c1000000-0000-4000-8000-000000000002"),
					fixedNow, 100, seedUnprocessedSID, nil)
				insertPurchase(ctx, t, drv, mustParse(t, "c1000000-0000-4000-8000-000000000003"),
					fixedNow, 100, seedUnprocessedSID, nil)

				got, err := svc.CountPurchasesByStatus(ctx, todayWindow())
				require.NoError(t, err)

				require.Len(t, got, 2)
				assert.Equal(t, mustParse(t, seedUnprocessedSID), got[0].StatusID)
				assert.Equal(t, "未処理", got[0].StatusName)
				assert.Equal(t, domainpurchase.StatusUnprocessed.Code(), got[0].StatusCode)
				assert.Equal(t, int64(2), got[0].Count)
				assert.Equal(t, mustParse(t, seedCompletedSID), got[1].StatusID)
				assert.Equal(t, "完了", got[1].StatusName)
				assert.Equal(t, domainpurchase.StatusCompleted.Code(), got[1].StatusCode)
				assert.Equal(t, int64(1), got[1].Count)
			})
		})

		t.Run("キャンセル済みの購入も1ステータスとして内訳に含める", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				clearSeededPurchases(ctx, t, drv)
				canceledAt := fixedNow
				insertPurchase(ctx, t, drv, mustParse(t, "c2000000-0000-4000-8000-000000000001"),
					fixedNow, 100, seedCanceledSID, &canceledAt)

				got, err := svc.CountPurchasesByStatus(ctx, todayWindow())
				require.NoError(t, err)

				require.Len(t, got, 1)
				assert.Equal(t, mustParse(t, seedCanceledSID), got[0].StatusID)
				assert.Equal(t, "キャンセル", got[0].StatusName)
				assert.Equal(t, domainpurchase.StatusCanceled.Code(), got[0].StatusCode)
				assert.Equal(t, int64(1), got[0].Count)
			})
		})

		t.Run("期間外に注文された購入は集計から除外する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				clearSeededPurchases(ctx, t, drv)
				insertPurchase(ctx, t, drv, mustParse(t, "c3000000-0000-4000-8000-000000000001"),
					fixedNow.AddDate(0, 0, -1), 100, seedUnprocessedSID, nil)

				got, err := svc.CountPurchasesByStatus(ctx, todayWindow())
				require.NoError(t, err)

				assert.Empty(t, got)
			})
		})

		t.Run("期間内の購入が無い場合は空を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				clearSeededPurchases(ctx, t, driver.New(ctx, testDB))

				got, err := svc.CountPurchasesByStatus(ctx, todayWindow())
				require.NoError(t, err)

				assert.Empty(t, got)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			got, err := svc.CountPurchasesByStatus(canceledContext(t), todayWindow())
			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.Nil(t, got)
		})
	})
}
