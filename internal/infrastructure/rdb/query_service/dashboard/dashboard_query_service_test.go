package dashboard

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	clocktestkit "go-boilerplate/internal/usecase/boundary/clock/testkit"
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

// testLoc は、集計期間の暦日境界を解釈するロケーションです。実行環境の time.Local に依存させないため、
// 本番設定と同じ Asia/Tokyo を明示的に固定します。
var testLoc = time.FixedZone("Asia/Tokyo", 9*60*60)

// fixedNow は、集計期間の境界算出の基準として用いる固定の現在時刻です。
// UTC で保持し、実装側が loc へ変換することを検証できるようにします。
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
		clk:    clocktestkit.NewMockClock(t, fixedNow),
		loc:    testLoc,
		tracer: observability.NewMockInfraLayerTracer(t),
	}
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

// todayPeriod は、fixedNow の当日を対象とする期間指定を返します。
func todayPeriod() query.Period {
	return query.Period{Kind: query.PeriodToday}
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡したドライバ・クロック・ロケーションとinfra層トレーサーを保持した実装を返す", func(t *testing.T) {
			t.Parallel()

			testDB := testkit.NewTestDB(t)
			tf := observability.NewNoopTracerFactory(t)
			clk := clocktestkit.NewMockClock(t, fixedNow)

			expected := &service{
				db:     testDB,
				clk:    clk,
				loc:    testLoc,
				tracer: tf.Infra(),
			}
			actual := New(testDB, clk, testLoc, tf)

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
				insertPurchase(ctx, t, drv, mustParse(t, "b1000000-0000-4000-8000-000000000001"),
					fixedNow, 1500, seedUnprocessedSID, nil)
				insertPurchase(ctx, t, drv, mustParse(t, "b1000000-0000-4000-8000-000000000002"),
					fixedNow, 2500, seedCompletedSID, nil)

				got, err := svc.SummarizeSales(ctx, todayPeriod())
				require.NoError(t, err)

				assert.Equal(t, int64(4000), got.Amount)
				assert.Equal(t, int64(2), got.Count)
			})
		})

		t.Run("キャンセル済みの購入は合計金額と件数の双方から除外する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				canceledAt := fixedNow
				insertPurchase(ctx, t, drv, mustParse(t, "b2000000-0000-4000-8000-000000000001"),
					fixedNow, 1000, seedUnprocessedSID, nil)
				insertPurchase(ctx, t, drv, mustParse(t, "b2000000-0000-4000-8000-000000000002"),
					fixedNow, 9999, seedCanceledSID, &canceledAt)

				got, err := svc.SummarizeSales(ctx, todayPeriod())
				require.NoError(t, err)

				assert.Equal(t, int64(1000), got.Amount)
				assert.Equal(t, int64(1), got.Count)
			})
		})

		t.Run("期間の下限は境界時刻を含み上限は含まない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				startOfToday := startOfDay(fixedNow.In(testLoc), testLoc)
				insertPurchase(ctx, t, drv, mustParse(t, "b3000000-0000-4000-8000-000000000001"),
					startOfToday, 100, seedUnprocessedSID, nil)
				insertPurchase(ctx, t, drv, mustParse(t, "b3000000-0000-4000-8000-000000000002"),
					startOfToday.AddDate(0, 0, 1), 200, seedUnprocessedSID, nil)
				insertPurchase(ctx, t, drv, mustParse(t, "b3000000-0000-4000-8000-000000000003"),
					startOfToday.Add(-time.Nanosecond), 400, seedUnprocessedSID, nil)

				got, err := svc.SummarizeSales(ctx, todayPeriod())
				require.NoError(t, err)

				assert.Equal(t, int64(100), got.Amount)
				assert.Equal(t, int64(1), got.Count)
			})
		})

		t.Run("range指定は開始日と終了日の暦日を両端とも集計対象に含める", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				from := time.Date(2026, time.June, 1, 0, 0, 0, 0, testLoc)
				to := time.Date(2026, time.June, 3, 0, 0, 0, 0, testLoc)
				insertPurchase(ctx, t, drv, mustParse(t, "b4000000-0000-4000-8000-000000000001"),
					from, 100, seedUnprocessedSID, nil)
				insertPurchase(ctx, t, drv, mustParse(t, "b4000000-0000-4000-8000-000000000002"),
					to.Add(23*time.Hour), 200, seedUnprocessedSID, nil)
				insertPurchase(ctx, t, drv, mustParse(t, "b4000000-0000-4000-8000-000000000003"),
					to.AddDate(0, 0, 1), 400, seedUnprocessedSID, nil)

				got, err := svc.SummarizeSales(ctx, query.Period{Kind: query.PeriodRange, From: from, To: to})
				require.NoError(t, err)

				assert.Equal(t, int64(300), got.Amount)
				assert.Equal(t, int64(2), got.Count)
			})
		})

		t.Run("期間内の購入が無い場合はゼロ値を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				got, err := svc.SummarizeSales(ctx, todayPeriod())
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

			got, err := svc.SummarizeSales(canceledContext(t), todayPeriod())
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
				// 表示順は 未処理(1) < 完了(5) のため、挿入順と逆になる。
				insertPurchase(ctx, t, drv, mustParse(t, "c1000000-0000-4000-8000-000000000001"),
					fixedNow, 100, seedCompletedSID, nil)
				insertPurchase(ctx, t, drv, mustParse(t, "c1000000-0000-4000-8000-000000000002"),
					fixedNow, 100, seedUnprocessedSID, nil)
				insertPurchase(ctx, t, drv, mustParse(t, "c1000000-0000-4000-8000-000000000003"),
					fixedNow, 100, seedUnprocessedSID, nil)

				got, err := svc.CountPurchasesByStatus(ctx, todayPeriod())
				require.NoError(t, err)

				require.Len(t, got, 2)
				assert.Equal(t, mustParse(t, seedUnprocessedSID), got[0].StatusID)
				assert.Equal(t, "未処理", got[0].StatusName)
				assert.Equal(t, int64(2), got[0].Count)
				assert.Equal(t, mustParse(t, seedCompletedSID), got[1].StatusID)
				assert.Equal(t, "完了", got[1].StatusName)
				assert.Equal(t, int64(1), got[1].Count)
			})
		})

		t.Run("キャンセル済みの購入も1ステータスとして内訳に含める", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				canceledAt := fixedNow
				insertPurchase(ctx, t, drv, mustParse(t, "c2000000-0000-4000-8000-000000000001"),
					fixedNow, 100, seedCanceledSID, &canceledAt)

				got, err := svc.CountPurchasesByStatus(ctx, todayPeriod())
				require.NoError(t, err)

				require.Len(t, got, 1)
				assert.Equal(t, mustParse(t, seedCanceledSID), got[0].StatusID)
				assert.Equal(t, "キャンセル", got[0].StatusName)
				assert.Equal(t, int64(1), got[0].Count)
			})
		})

		t.Run("期間外に注文された購入は集計から除外する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertPurchase(ctx, t, drv, mustParse(t, "c3000000-0000-4000-8000-000000000001"),
					fixedNow.AddDate(0, 0, -1), 100, seedUnprocessedSID, nil)

				got, err := svc.CountPurchasesByStatus(ctx, todayPeriod())
				require.NoError(t, err)

				assert.Empty(t, got)
			})
		})

		t.Run("期間内の購入が無い場合は空を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				got, err := svc.CountPurchasesByStatus(ctx, todayPeriod())
				require.NoError(t, err)

				assert.Empty(t, got)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			got, err := svc.CountPurchasesByStatus(canceledContext(t), todayPeriod())
			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.Nil(t, got)
		})
	})
}

func Test_resolveWindow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 15, 12, 34, 56, 0, testLoc)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("todayは当日00:00から翌日00:00までの半開区間を返す", func(t *testing.T) {
			t.Parallel()

			after, before := resolveWindow(query.Period{Kind: query.PeriodToday}, now, testLoc)

			assert.Equal(t, time.Date(2026, time.July, 15, 0, 0, 0, 0, testLoc), after)
			assert.Equal(t, time.Date(2026, time.July, 16, 0, 0, 0, 0, testLoc), before)
		})

		t.Run("monthは月初00:00から翌月1日00:00までの半開区間を返す", func(t *testing.T) {
			t.Parallel()

			after, before := resolveWindow(query.Period{Kind: query.PeriodMonth}, now, testLoc)

			assert.Equal(t, time.Date(2026, time.July, 1, 0, 0, 0, 0, testLoc), after)
			assert.Equal(t, time.Date(2026, time.August, 1, 0, 0, 0, 0, testLoc), before)
		})

		t.Run("12月のmonthは翌年1月1日を上限とする", func(t *testing.T) {
			t.Parallel()

			december := time.Date(2026, time.December, 31, 23, 0, 0, 0, testLoc)

			after, before := resolveWindow(query.Period{Kind: query.PeriodMonth}, december, testLoc)

			assert.Equal(t, time.Date(2026, time.December, 1, 0, 0, 0, 0, testLoc), after)
			assert.Equal(t, time.Date(2027, time.January, 1, 0, 0, 0, 0, testLoc), before)
		})

		t.Run("rangeは開始日00:00から終了日の翌日00:00までの半開区間を返す", func(t *testing.T) {
			t.Parallel()

			from := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
			to := time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)

			after, before := resolveWindow(query.Period{Kind: query.PeriodRange, From: from, To: to}, now, testLoc)

			assert.Equal(t, time.Date(2026, time.June, 1, 0, 0, 0, 0, testLoc), after)
			assert.Equal(t, time.Date(2026, time.July, 1, 0, 0, 0, 0, testLoc), before)
		})

		t.Run("月末を終了日とするrangeは翌月1日を上限とする", func(t *testing.T) {
			t.Parallel()

			from := time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)
			to := time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)

			after, before := resolveWindow(query.Period{Kind: query.PeriodRange, From: from, To: to}, now, testLoc)

			assert.Equal(t, time.Date(2026, time.January, 31, 0, 0, 0, 0, testLoc), after)
			assert.Equal(t, time.Date(2026, time.February, 1, 0, 0, 0, 0, testLoc), before)
		})

		t.Run("現在時刻が別ロケーションで渡されても暦日は指定ロケーションで切る", func(t *testing.T) {
			t.Parallel()

			utcNow := time.Date(2026, time.July, 15, 16, 0, 0, 0, time.UTC)

			after, before := resolveWindow(query.Period{Kind: query.PeriodToday}, utcNow, testLoc)

			assert.Equal(t, time.Date(2026, time.July, 16, 0, 0, 0, 0, testLoc), after)
			assert.Equal(t, time.Date(2026, time.July, 17, 0, 0, 0, 0, testLoc), before)
		})

		t.Run("range指定の開始日と終了日は現在時刻のロケーション変換を受けない", func(t *testing.T) {
			t.Parallel()

			from := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
			to := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
			utcNow := time.Date(2026, time.July, 15, 16, 0, 0, 0, time.UTC)

			after, before := resolveWindow(
				query.Period{Kind: query.PeriodRange, From: from, To: to}, utcNow, testLoc)

			assert.Equal(t, time.Date(2026, time.June, 1, 0, 0, 0, 0, testLoc), after)
			assert.Equal(t, time.Date(2026, time.June, 2, 0, 0, 0, 0, testLoc), before)
		})

		t.Run("未知の区分はtodayと同じ半開区間を返す", func(t *testing.T) {
			t.Parallel()

			after, before := resolveWindow(query.Period{Kind: "weekly"}, now, testLoc)

			assert.Equal(t, time.Date(2026, time.July, 15, 0, 0, 0, 0, testLoc), after)
			assert.Equal(t, time.Date(2026, time.July, 16, 0, 0, 0, 0, testLoc), before)
		})
	})
}

func Test_startOfDay(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("時刻部分を切り落として指定ロケーションの暦日の開始時刻を返す", func(t *testing.T) {
			t.Parallel()

			got := startOfDay(time.Date(2026, time.July, 15, 23, 59, 59, 999, time.UTC), testLoc)

			assert.Equal(t, time.Date(2026, time.July, 15, 0, 0, 0, 0, testLoc), got)
		})
	})
}
