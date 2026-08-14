package summary

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/purchase/period"
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
	seedLaptopPID      = "1d137c76-aa49-4676-8997-f8d8f7bb1af2" // 商品（電子機器・ASUS Zenbook 14 OLED）
	seedTabletPID      = "d42d659d-21f9-4b5c-b05d-3130de157a94" // 商品（電子機器・Lenovo Tab P12）
	seedBookPID        = "8e897115-312b-4f20-811e-96979682c7dc" // 商品（書籍・推し、燃ゆ）
)

// テスト基準日。seed が投入する購入履歴は削除するため、実時刻から離れた固定日でよい。
var (
	testBaseDay = time.Date(2026, time.January, 25, 12, 0, 0, 0, time.UTC)
	testLoc     = time.FixedZone("TEST+09", 9*60*60)
)

func mustParse(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	require.NoError(t, err)
	return id
}

// testWindow は、両端を含む暦日から絞り込み済みの対象期間を組み立てます。
func testWindow(t *testing.T, from, to time.Time) period.Window {
	t.Helper()
	w, err := period.Resolve(period.Spec{Kind: period.KindRange, From: &from, To: &to}, time.Time{}, testLoc)
	require.NoError(t, err)
	return w
}

// canceledContext は、キャンセル済みのコンテキストを返します。
func canceledContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	return ctx
}

// clearSeededPurchases は、呼び出したトランザクション内で購入と購入明細を空にします。
// seed が投入する購入履歴が残っていると、対象ユーザーの集計件数が期待値とずれます。
// 削除はロールバックされるので、コミット済みの seed は失われません。
func clearSeededPurchases(ctx context.Context, t *testing.T, db driver.DBTX) {
	t.Helper()
	_, err := db.Exec(ctx, "DELETE FROM purchase_details")
	require.NoError(t, err)
	_, err = db.Exec(ctx, "DELETE FROM purchases")
	require.NoError(t, err)
}

// insertPurchase は、指定ユーザー・ステータス・合計金額の購入を 1 件挿入します。
func insertPurchase(ctx context.Context, t *testing.T, db driver.DBTX, userID, statusID uuid.UUID, code string, totalAmount int64) {
	t.Helper()
	insertPurchaseAt(ctx, t, db, userID, statusID, code, totalAmount, testBaseDay, nil)
}

// insertPurchaseAt は、注文日時とキャンセル日時を指定して購入を 1 件挿入し、その ID を返します。
func insertPurchaseAt(
	ctx context.Context, t *testing.T, db driver.DBTX,
	userID, statusID uuid.UUID, code string, totalAmount int64, orderedAt time.Time, canceledAt *time.Time,
) uuid.UUID {
	t.Helper()
	purchaseID, err := uuid.New()
	require.NoError(t, err)
	_, err = db.Exec(ctx,
		"INSERT INTO purchases (id, code, user_id, status_id, subtotal_amount, tax_amount, shipping_fee, total_amount, ordered_at, canceled_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)",
		purchaseID, code, userID, statusID, totalAmount, 0, 0, totalAmount, orderedAt, canceledAt,
	)
	require.NoError(t, err)
	return purchaseID
}

// insertPurchaseDetail は、指定購入に商品 1 件分の明細を挿入します。
func insertPurchaseDetail(
	ctx context.Context, t *testing.T, db driver.DBTX, purchaseID, productID uuid.UUID, quantity int, unitPrice string,
) {
	t.Helper()
	detailID, err := uuid.New()
	require.NoError(t, err)
	_, err = db.Exec(ctx,
		"INSERT INTO purchase_details (id, purchase_id, product_id, quantity, unit_price) VALUES ($1,$2,$3,$4,$5)",
		detailID, purchaseID, productID, quantity, unitPrice,
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
				clearSeededPurchases(ctx, t, drv)
				userA := mustParse(t, seedUserA)
				// 表示順（sort_key）が後のステータスから挿入し、挿入順ではなくマスタ順で並ぶことを確認する。
				insertPurchase(ctx, t, drv, userA, mustParse(t, seedPaidSID), "sm-paid-1", 300)
				insertPurchase(ctx, t, drv, userA, mustParse(t, seedUnprocessedSID), "sm-unproc-1", 150)
				insertPurchase(ctx, t, drv, userA, mustParse(t, seedUnprocessedSID), "sm-unproc-2", 250)

				got, err := svc.SummarizeByUserID(ctx, userA, period.Window{})
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
				clearSeededPurchases(ctx, t, drv)
				userA := mustParse(t, seedUserA)
				userB := mustParse(t, seedUserB)
				insertPurchase(ctx, t, drv, userA, mustParse(t, seedUnprocessedSID), "sm-own-1", 100)
				insertPurchase(ctx, t, drv, userB, mustParse(t, seedUnprocessedSID), "sm-other-1", 999)
				insertPurchase(ctx, t, drv, userB, mustParse(t, seedPaidSID), "sm-other-2", 999)

				got, err := svc.SummarizeByUserID(ctx, userA, period.Window{})
				require.NoError(t, err)
				require.Len(t, got, 1)
				assert.Equal(t, int64(1), got[0].Count)
				assert.Equal(t, int64(100), got[0].TotalAmount)
			})
		})

		t.Run("キャンセル済みの購入は集計対象から除外する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				clearSeededPurchases(ctx, t, drv)
				userA := mustParse(t, seedUserA)
				canceledAt := testBaseDay
				insertPurchaseAt(ctx, t, drv, userA, mustParse(t, seedCanceledSID), "sm-canceled-1", 400, testBaseDay, &canceledAt)
				insertPurchase(ctx, t, drv, userA, mustParse(t, seedUnprocessedSID), "sm-live-1", 100)

				got, err := svc.SummarizeByUserID(ctx, userA, period.Window{})
				require.NoError(t, err)
				// キャンセルは 1 要素としても現れないため、内訳の総和は総件数と常に一致する。
				require.Len(t, got, 1)
				assert.Equal(t, "未処理", got[0].StatusName)
				assert.Equal(t, int64(1), got[0].Count)
				assert.Equal(t, int64(100), got[0].TotalAmount)
			})
		})

		t.Run("対象期間の外に注文された購入は集計しない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				clearSeededPurchases(ctx, t, drv)
				userA := mustParse(t, seedUserA)
				statusID := mustParse(t, seedUnprocessedSID)
				insertPurchaseAt(ctx, t, drv, userA, statusID, "sm-before", 100, testBaseDay.AddDate(0, 0, -10), nil)
				insertPurchaseAt(ctx, t, drv, userA, statusID, "sm-inside", 200, testBaseDay, nil)
				insertPurchaseAt(ctx, t, drv, userA, statusID, "sm-after", 400, testBaseDay.AddDate(0, 0, 10), nil)

				window := testWindow(t, testBaseDay.AddDate(0, 0, -1), testBaseDay.AddDate(0, 0, 1))
				got, err := svc.SummarizeByUserID(ctx, userA, window)
				require.NoError(t, err)
				require.Len(t, got, 1)
				assert.Equal(t, int64(1), got[0].Count)
				assert.Equal(t, int64(200), got[0].TotalAmount)
			})
		})

		t.Run("対象期間は終了日の当日中に注文された購入も含む", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				clearSeededPurchases(ctx, t, drv)
				userA := mustParse(t, seedUserA)
				// loc の暦日で終了日の 23:00 に相当する時刻。上限が終了日 00:00 だと取りこぼす境界。
				lateOnEndDay := time.Date(2026, time.January, 25, 23, 0, 0, 0, testLoc)
				insertPurchaseAt(ctx, t, drv, userA, mustParse(t, seedUnprocessedSID), "sm-late", 700, lateOnEndDay, nil)

				window := testWindow(t, testBaseDay, testBaseDay)
				got, err := svc.SummarizeByUserID(ctx, userA, window)
				require.NoError(t, err)
				require.Len(t, got, 1)
				assert.Equal(t, int64(700), got[0].TotalAmount)
			})
		})

		t.Run("購入が1件もないユーザーは空スライスを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				clearSeededPurchases(ctx, t, driver.New(ctx, testDB))

				got, err := svc.SummarizeByUserID(ctx, mustParse(t, seedUserB), period.Window{})
				require.NoError(t, err)
				assert.Empty(t, got)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			got, err := svc.SummarizeByUserID(canceledContext(t), mustParse(t, seedUserA), period.Window{})
			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.Nil(t, got)
		})
	})
}

func Test_service_SumItemsByUserID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	svc := &service{tracer: lt, db: testDB}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("明細の単価と数量の積を丸めずに合計する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				clearSeededPurchases(ctx, t, drv)
				userA := mustParse(t, seedUserA)
				purchaseID := insertPurchaseAt(ctx, t, drv, userA, mustParse(t, seedUnprocessedSID), "si-1", 0, testBaseDay, nil)
				// 決済スケール（セント）へ丸めると失われる小数を含む単価を混ぜる。
				insertPurchaseDetail(ctx, t, drv, purchaseID, mustParse(t, seedLaptopPID), 1, "800.005")
				insertPurchaseDetail(ctx, t, drv, purchaseID, mustParse(t, seedBookPID), 3, "10.00")

				got, err := svc.SumItemsByUserID(ctx, userA, period.Window{})
				require.NoError(t, err)
				// セントへ切り捨てると 830.00 になる値。丸めを挟まないことを 0.005 の残存で固定する。
				assert.Equal(t, "830.005", got.String())
			})
		})

		t.Run("他ユーザーとキャンセル済みと期間外の明細は合計に含めない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				clearSeededPurchases(ctx, t, drv)
				userA := mustParse(t, seedUserA)
				userB := mustParse(t, seedUserB)
				statusID := mustParse(t, seedUnprocessedSID)
				canceledAt := testBaseDay

				target := insertPurchaseAt(ctx, t, drv, userA, statusID, "si-target", 0, testBaseDay, nil)
				insertPurchaseDetail(ctx, t, drv, target, mustParse(t, seedBookPID), 1, "10.00")

				other := insertPurchaseAt(ctx, t, drv, userB, statusID, "si-other", 0, testBaseDay, nil)
				insertPurchaseDetail(ctx, t, drv, other, mustParse(t, seedBookPID), 5, "10.00")

				canceled := insertPurchaseAt(ctx, t, drv, userA, mustParse(t, seedCanceledSID), "si-canceled", 0, testBaseDay, &canceledAt)
				insertPurchaseDetail(ctx, t, drv, canceled, mustParse(t, seedBookPID), 7, "10.00")

				outside := insertPurchaseAt(ctx, t, drv, userA, statusID, "si-outside", 0, testBaseDay.AddDate(0, 0, 10), nil)
				insertPurchaseDetail(ctx, t, drv, outside, mustParse(t, seedBookPID), 9, "10.00")

				window := testWindow(t, testBaseDay.AddDate(0, 0, -1), testBaseDay.AddDate(0, 0, 1))
				got, err := svc.SumItemsByUserID(ctx, userA, window)
				require.NoError(t, err)
				assert.Equal(t, "10", got.String())
			})
		})

		t.Run("対象の明細が無いときゼロ値を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				clearSeededPurchases(ctx, t, driver.New(ctx, testDB))

				got, err := svc.SumItemsByUserID(ctx, mustParse(t, seedUserB), period.Window{})
				require.NoError(t, err)
				assert.True(t, got.IsZero())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			got, err := svc.SumItemsByUserID(canceledContext(t), mustParse(t, seedUserA), period.Window{})
			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.True(t, got.IsZero())
		})
	})
}

func Test_service_SummarizeItemsByProductByUserID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	svc := &service{tracer: lt, db: testDB}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("商品単位に集計しカテゴリ名を添えてマスタの表示順で返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				clearSeededPurchases(ctx, t, drv)
				userA := mustParse(t, seedUserA)
				statusID := mustParse(t, seedUnprocessedSID)

				// 別購入に分かれた同一商品が 1 行へ合算されることも同時に固定する。
				first := insertPurchaseAt(ctx, t, drv, userA, statusID, "sp-1", 0, testBaseDay, nil)
				insertPurchaseDetail(ctx, t, drv, first, mustParse(t, seedBookPID), 1, "10.00")
				insertPurchaseDetail(ctx, t, drv, first, mustParse(t, seedLaptopPID), 1, "800.00")

				second := insertPurchaseAt(ctx, t, drv, userA, statusID, "sp-2", 0, testBaseDay, nil)
				insertPurchaseDetail(ctx, t, drv, second, mustParse(t, seedLaptopPID), 2, "800.00")

				got, err := svc.SummarizeItemsByProductByUserID(ctx, userA, period.Window{})
				require.NoError(t, err)
				require.Len(t, got, 2)

				// 電子機器（sort_key=1）が書籍（sort_key=2）より先に並ぶ。
				assert.Equal(t, "電子機器", got[0].CategoryName)
				assert.Equal(t, mustParse(t, seedLaptopPID), got[0].ProductID)
				assert.Equal(t, "ASUS Zenbook 14 OLED", got[0].ProductName)
				assert.Equal(t, "2400", got[0].ItemsTotal.String())

				assert.Equal(t, "書籍", got[1].CategoryName)
				assert.Equal(t, "10", got[1].ItemsTotal.String())
			})
		})

		t.Run("同一カテゴリの別商品は別行として返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				clearSeededPurchases(ctx, t, drv)
				userA := mustParse(t, seedUserA)
				purchaseID := insertPurchaseAt(ctx, t, drv, userA, mustParse(t, seedUnprocessedSID), "sp-cat", 0, testBaseDay, nil)
				insertPurchaseDetail(ctx, t, drv, purchaseID, mustParse(t, seedLaptopPID), 1, "800.00")
				insertPurchaseDetail(ctx, t, drv, purchaseID, mustParse(t, seedTabletPID), 1, "400.00")

				got, err := svc.SummarizeItemsByProductByUserID(ctx, userA, period.Window{})
				require.NoError(t, err)
				require.Len(t, got, 2)
				assert.Equal(t, "電子機器", got[0].CategoryName)
				assert.Equal(t, "電子機器", got[1].CategoryName)
				assert.NotEqual(t, got[0].ProductID, got[1].ProductID)
			})
		})

		t.Run("他ユーザーとキャンセル済みと期間外の明細は集計しない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				clearSeededPurchases(ctx, t, drv)
				userA := mustParse(t, seedUserA)
				userB := mustParse(t, seedUserB)
				statusID := mustParse(t, seedUnprocessedSID)
				canceledAt := testBaseDay

				target := insertPurchaseAt(ctx, t, drv, userA, statusID, "sp-target", 0, testBaseDay, nil)
				insertPurchaseDetail(ctx, t, drv, target, mustParse(t, seedBookPID), 1, "10.00")

				other := insertPurchaseAt(ctx, t, drv, userB, statusID, "sp-other", 0, testBaseDay, nil)
				insertPurchaseDetail(ctx, t, drv, other, mustParse(t, seedLaptopPID), 1, "800.00")

				canceled := insertPurchaseAt(ctx, t, drv, userA, mustParse(t, seedCanceledSID), "sp-canceled", 0, testBaseDay, &canceledAt)
				insertPurchaseDetail(ctx, t, drv, canceled, mustParse(t, seedTabletPID), 1, "400.00")

				outside := insertPurchaseAt(ctx, t, drv, userA, statusID, "sp-outside", 0, testBaseDay.AddDate(0, 0, 10), nil)
				insertPurchaseDetail(ctx, t, drv, outside, mustParse(t, seedTabletPID), 1, "400.00")

				window := testWindow(t, testBaseDay.AddDate(0, 0, -1), testBaseDay.AddDate(0, 0, 1))
				got, err := svc.SummarizeItemsByProductByUserID(ctx, userA, window)
				require.NoError(t, err)
				require.Len(t, got, 1)
				assert.Equal(t, mustParse(t, seedBookPID), got[0].ProductID)
			})
		})

		t.Run("非公開の商品も購入実績として集計する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				clearSeededPurchases(ctx, t, drv)
				userA := mustParse(t, seedUserA)
				purchaseID := insertPurchaseAt(ctx, t, drv, userA, mustParse(t, seedUnprocessedSID), "sp-unpub", 0, testBaseDay, nil)
				// Lenovo Tab P12 は published_at が NULL。売上ランキングと違い購入実績は現在の公開状態に依存しない。
				insertPurchaseDetail(ctx, t, drv, purchaseID, mustParse(t, seedTabletPID), 1, "400.00")

				got, err := svc.SummarizeItemsByProductByUserID(ctx, userA, period.Window{})
				require.NoError(t, err)
				require.Len(t, got, 1)
				assert.Equal(t, mustParse(t, seedTabletPID), got[0].ProductID)
			})
		})

		t.Run("対象の明細が無いとき空スライスを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				clearSeededPurchases(ctx, t, driver.New(ctx, testDB))

				got, err := svc.SummarizeItemsByProductByUserID(ctx, mustParse(t, seedUserB), period.Window{})
				require.NoError(t, err)
				assert.Empty(t, got)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			got, err := svc.SummarizeItemsByProductByUserID(canceledContext(t), mustParse(t, seedUserA), period.Window{})
			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.Nil(t, got)
		})
	})
}

func Test_bounds(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("絞り込む期間は半開区間の両端をポインタで渡す", func(t *testing.T) {
			t.Parallel()

			window := testWindow(t, testBaseDay, testBaseDay)
			filter, after, before := bounds(window)

			assert.True(t, filter)
			require.NotNil(t, after)
			require.NotNil(t, before)
			expectedAfter, expectedBefore := window.Bounds()
			assert.True(t, expectedAfter.Equal(*after))
			assert.True(t, expectedBefore.Equal(*before))
		})

		t.Run("絞り込まない期間は境界をNULLのまま渡しフラグだけで述語を無効にする", func(t *testing.T) {
			t.Parallel()

			// 境界に値を入れてフラグだけ false にすると、SQL 側の OR 短絡が壊れたときに気づけない。
			filter, after, before := bounds(period.Window{})

			assert.False(t, filter)
			assert.Nil(t, after)
			assert.Nil(t, before)
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
