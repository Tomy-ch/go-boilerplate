package dashboard

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/product"
	mock_product "go-boilerplate/internal/domain/product/mock"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	mock_authz "go-boilerplate/internal/usecase/boundary/authz/mock"
	clocktestkit "go-boilerplate/internal/usecase/boundary/clock/testkit"
	"go-boilerplate/internal/usecase/dashboard/query"
	mock_query "go-boilerplate/internal/usecase/dashboard/query/mock"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// 取得元ごとに一意なエラー。伝播先を取り違えたり一律のエラーへ潰したりする実装を検出するために分けます。
// testLoc は、集計期間の暦日境界を解釈するロケーションです。実行環境の time.Local に依存させないため、
// 本番設定と同じ Asia/Tokyo を明示的に固定します。
var testLoc = time.FixedZone("Asia/Tokyo", 9*60*60)

// fixedNow は、集計期間の境界算出の基準として用いる固定の現在時刻です。
// UTC で保持し、実装側が loc へ変換することを検証できるようにします。
var fixedNow = time.Date(2026, time.July, 15, 12, 0, 0, 0, testLoc).UTC()

var (
	errSalesProbe   = xerrors.New("sales probe")
	errStatusProbe  = xerrors.New("status probe")
	errProductProbe = xerrors.New("product probe")
)

// deps は、ユースケースが依存するモック一式です。
type deps struct {
	qs          *mock_query.MockDashboardQueryService
	productRepo *mock_product.MockRepository
	authorizer  *mock_authz.MockAuthorizer
}

func newUsecase(t *testing.T) (*usecase, deps) {
	t.Helper()
	ctrl := gomock.NewController(t)
	d := deps{
		qs:          mock_query.NewMockDashboardQueryService(ctrl),
		productRepo: mock_product.NewMockRepository(ctrl),
		authorizer:  mock_authz.NewMockAuthorizer(ctrl),
	}
	return &usecase{
		tracer:      observability.NewMockUsecaseLayerTracer(t),
		authorizer:  d.authorizer,
		qs:          d.qs,
		productRepo: d.productRepo,
		clk:         clocktestkit.NewMockClock(t, fixedNow),
		loc:         testLoc,
	}, d
}

// expectAllQueries は、3 つの集計取得すべてにゼロ値を返させます。
func expectAllQueries(d deps) {
	d.qs.EXPECT().SummarizeSales(gomock.Any(), gomock.Any()).Return(query.SalesResult{}, nil)
	d.qs.EXPECT().CountPurchasesByStatus(gomock.Any(), gomock.Any()).Return(nil, nil)
	d.productRepo.EXPECT().Count(gomock.Any()).Return(product.Counts{}, nil)
}

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse(time.DateOnly, s)
	require.NoError(t, err)
	return d
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡した依存とusecase層トレーサーを保持した実装を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)
			qs := mock_query.NewMockDashboardQueryService(ctrl)
			productRepo := mock_product.NewMockRepository(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			clk := clocktestkit.NewMockClock(t, fixedNow)

			expected := &usecase{
				tracer:      tf.Usecase(),
				authorizer:  authorizer,
				qs:          qs,
				productRepo: productRepo,
				clk:         clk,
				loc:         testLoc,
			}
			actual := New(qs, productRepo, authorizer, clk, testLoc, tf)

			assert.Equal(t, expected, actual)
		})
	})
}

func Test_usecase_GetDashboardSummary(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("3つの集計結果を1つのSummaryViewへ合成して返す", func(t *testing.T) {
			t.Parallel()

			statusID := uuidtestkit.NewTestFromSalt(t, "uc_dash_status")

			u, d := newUsecase(t)
			d.authorizer.EXPECT().
				Authorize(gomock.Any(), gomock.Any(), authz.ActionDashboardRead,
					authz.NewResource(resourceKindDashboard, nil)).Return(nil)
			d.qs.EXPECT().SummarizeSales(gomock.Any(), gomock.Any()).
				Return(query.SalesResult{Amount: 450000, Count: 12}, nil)
			d.qs.EXPECT().CountPurchasesByStatus(gomock.Any(), gomock.Any()).
				Return([]query.PurchaseStatusCountResult{
					{StatusID: statusID, StatusName: "完了", Count: 7},
				}, nil)
			d.productRepo.EXPECT().Count(gomock.Any()).
				Return(product.Counts{Total: 120, Published: 98}, nil)

			got, err := u.GetDashboardSummary(context.Background(), &auth.Authn{}, GetSummaryParams{Period: "today"})
			require.NoError(t, err)

			assert.Equal(t, int64(450000), got.SalesAmount)
			assert.Equal(t, int64(12), got.SalesCount)
			assert.Equal(t, int64(120), got.TotalProductCount)
			assert.Equal(t, int64(98), got.PublishedProductCount)
			require.Len(t, got.PurchaseStatusCounts, 1)
			assert.Equal(t, statusID, got.PurchaseStatusCounts[0].StatusID)
			assert.Equal(t, "完了", got.PurchaseStatusCounts[0].StatusName)
			assert.Equal(t, int64(7), got.PurchaseStatusCounts[0].Count)
		})

		t.Run("正規化した期間指定が売上とステータス別件数の双方へ同一の値で渡る", func(t *testing.T) {
			t.Parallel()

			from := mustDate(t, "2026-07-01")
			to := mustDate(t, "2026-07-31")

			u, d := newUsecase(t)
			d.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			var salesWindow, statusWindow query.Window
			d.qs.EXPECT().SummarizeSales(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, w query.Window) (query.SalesResult, error) {
					salesWindow = w
					return query.SalesResult{}, nil
				})
			d.qs.EXPECT().CountPurchasesByStatus(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, w query.Window) ([]query.PurchaseStatusCountResult, error) {
					statusWindow = w
					return nil, nil
				})
			d.productRepo.EXPECT().Count(gomock.Any()).Return(product.Counts{}, nil)

			_, err := u.GetDashboardSummary(context.Background(), &auth.Authn{}, GetSummaryParams{
				Period: "range", From: &from, To: &to,
			})
			require.NoError(t, err)

			// QS へは解決済みの半開区間が渡る。上限は終了日の翌日で、終了日当日の注文を取りこぼさない。
			assert.Equal(t, query.Window{
				After:  time.Date(2026, time.July, 1, 0, 0, 0, 0, testLoc),
				Before: time.Date(2026, time.August, 1, 0, 0, 0, 0, testLoc),
			}, salesWindow)
			assert.Equal(t, salesWindow, statusWindow)
		})

		t.Run("集計対象が無い場合ゼロ値とnilでない空スライスを返す", func(t *testing.T) {
			t.Parallel()

			u, d := newUsecase(t)
			d.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			expectAllQueries(d)

			got, err := u.GetDashboardSummary(context.Background(), &auth.Authn{}, GetSummaryParams{})
			require.NoError(t, err)

			assert.Zero(t, got.SalesAmount)
			assert.Zero(t, got.SalesCount)
			assert.Zero(t, got.TotalProductCount)
			assert.Zero(t, got.PublishedProductCount)
			assert.NotNil(t, got.PurchaseStatusCounts)
			assert.Empty(t, got.PurchaseStatusCounts)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("authnがnilの場合、認可を経ずErrUnauthenticatedを返す", func(t *testing.T) {
			t.Parallel()

			u, d := newUsecase(t)
			d.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			_, err := u.GetDashboardSummary(context.Background(), nil, GetSummaryParams{})
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
		})

		t.Run("管理者以外の場合、認可エラーを伝播し集計を実行しない", func(t *testing.T) {
			t.Parallel()

			u, d := newUsecase(t)
			d.authorizer.EXPECT().
				Authorize(gomock.Any(), gomock.Any(), authz.ActionDashboardRead, gomock.Any()).
				Return(apperror.ErrPermissionDenied)
			d.qs.EXPECT().SummarizeSales(gomock.Any(), gomock.Any()).Times(0)
			d.qs.EXPECT().CountPurchasesByStatus(gomock.Any(), gomock.Any()).Times(0)
			d.productRepo.EXPECT().Count(gomock.Any()).Times(0)

			_, err := u.GetDashboardSummary(context.Background(), &auth.Authn{}, GetSummaryParams{})
			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})

		t.Run("period=rangeでfrom/toが欠落する場合、集計を実行せずErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			u, d := newUsecase(t)
			d.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			d.qs.EXPECT().SummarizeSales(gomock.Any(), gomock.Any()).Times(0)

			_, err := u.GetDashboardSummary(context.Background(), &auth.Authn{}, GetSummaryParams{Period: "range"})
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("売上集計のエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			u, d := newUsecase(t)
			d.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			d.qs.EXPECT().SummarizeSales(gomock.Any(), gomock.Any()).
				Return(query.SalesResult{}, errSalesProbe)

			_, err := u.GetDashboardSummary(context.Background(), &auth.Authn{}, GetSummaryParams{})
			require.ErrorIs(t, err, errSalesProbe)
		})

		t.Run("ステータス別件数のエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			u, d := newUsecase(t)
			d.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			d.qs.EXPECT().SummarizeSales(gomock.Any(), gomock.Any()).Return(query.SalesResult{}, nil)
			d.qs.EXPECT().CountPurchasesByStatus(gomock.Any(), gomock.Any()).Return(nil, errStatusProbe)

			_, err := u.GetDashboardSummary(context.Background(), &auth.Authn{}, GetSummaryParams{})
			require.ErrorIs(t, err, errStatusProbe)
		})

		t.Run("商品数取得のエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			u, d := newUsecase(t)
			d.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			d.qs.EXPECT().SummarizeSales(gomock.Any(), gomock.Any()).Return(query.SalesResult{}, nil)
			d.qs.EXPECT().CountPurchasesByStatus(gomock.Any(), gomock.Any()).Return(nil, nil)
			d.productRepo.EXPECT().Count(gomock.Any()).Return(product.Counts{}, errProductProbe)

			_, err := u.GetDashboardSummary(context.Background(), &auth.Authn{}, GetSummaryParams{})
			require.ErrorIs(t, err, errProductProbe)
		})
	})
}

func Test_resolveWindow(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("monthは当月の月初から翌月の月初までを対象にする", func(t *testing.T) {
			t.Parallel()

			got, err := resolveWindow(GetSummaryParams{Period: "month"}, fixedNow, testLoc)
			require.NoError(t, err)

			assert.Equal(t, query.Window{
				After:  time.Date(2026, time.July, 1, 0, 0, 0, 0, testLoc),
				Before: time.Date(2026, time.August, 1, 0, 0, 0, 0, testLoc),
			}, got)
		})

		t.Run("todayは当日1日分を対象にする", func(t *testing.T) {
			t.Parallel()

			got, err := resolveWindow(GetSummaryParams{Period: "today"}, fixedNow, testLoc)
			require.NoError(t, err)

			assert.Equal(t, query.Window{
				After:  time.Date(2026, time.July, 15, 0, 0, 0, 0, testLoc),
				Before: time.Date(2026, time.July, 16, 0, 0, 0, 0, testLoc),
			}, got)
		})

		t.Run("空文字はtodayとして扱う", func(t *testing.T) {
			t.Parallel()

			got, err := resolveWindow(GetSummaryParams{Period: ""}, fixedNow, testLoc)
			require.NoError(t, err)
			assert.Equal(t, time.Date(2026, time.July, 15, 0, 0, 0, 0, testLoc), got.After)
		})

		t.Run("未知の区分はtodayとして扱う", func(t *testing.T) {
			t.Parallel()

			got, err := resolveWindow(GetSummaryParams{Period: "weekly"}, fixedNow, testLoc)
			require.NoError(t, err)
			assert.Equal(t, time.Date(2026, time.July, 15, 0, 0, 0, 0, testLoc), got.After)
		})

		t.Run("rangeは指定された暦日の両端を含み上限は終了日の翌日になる", func(t *testing.T) {
			t.Parallel()

			from := mustDate(t, "2026-07-01")
			to := mustDate(t, "2026-07-31")

			got, err := resolveWindow(GetSummaryParams{Period: "range", From: &from, To: &to}, fixedNow, testLoc)
			require.NoError(t, err)

			assert.Equal(t, query.Window{
				After:  time.Date(2026, time.July, 1, 0, 0, 0, 0, testLoc),
				Before: time.Date(2026, time.August, 1, 0, 0, 0, 0, testLoc),
			}, got)
		})

		t.Run("rangeはfromとtoが同一日の単日指定も受理する", func(t *testing.T) {
			t.Parallel()

			day := mustDate(t, "2026-07-15")

			got, err := resolveWindow(GetSummaryParams{Period: "range", From: &day, To: &day}, fixedNow, testLoc)
			require.NoError(t, err)

			assert.Equal(t, query.Window{
				After:  time.Date(2026, time.July, 15, 0, 0, 0, 0, testLoc),
				Before: time.Date(2026, time.July, 16, 0, 0, 0, 0, testLoc),
			}, got)
		})

		t.Run("rangeは時刻成分を落として暦日の境界へ揃える", func(t *testing.T) {
			t.Parallel()

			from := time.Date(2026, time.July, 15, 23, 59, 59, 0, time.UTC)
			to := time.Date(2026, time.July, 15, 1, 0, 0, 0, time.UTC)

			got, err := resolveWindow(GetSummaryParams{Period: "range", From: &from, To: &to}, fixedNow, testLoc)
			require.NoError(t, err)

			assert.Equal(t, time.Date(2026, time.July, 15, 0, 0, 0, 0, testLoc), got.After)
		})

		t.Run("todayの当日はUTCではなく指定ロケーションの暦日で決まる", func(t *testing.T) {
			t.Parallel()

			// UTC では 2026-07-15 22:00 だが testLoc では 2026-07-16。ロケーションを無視すると前日を集計する。
			crossing := time.Date(2026, time.July, 15, 22, 0, 0, 0, time.UTC)

			got, err := resolveWindow(GetSummaryParams{Period: "today"}, crossing, testLoc)
			require.NoError(t, err)
			assert.Equal(t, time.Date(2026, time.July, 16, 0, 0, 0, 0, testLoc), got.After)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("rangeでfromが未指定の場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			to := mustDate(t, "2026-07-31")

			_, err := resolveWindow(GetSummaryParams{Period: "range", To: &to}, fixedNow, testLoc)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("rangeでtoが未指定の場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			from := mustDate(t, "2026-07-01")

			_, err := resolveWindow(GetSummaryParams{Period: "range", From: &from}, fixedNow, testLoc)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("rangeでtoがfromより前の場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			from := mustDate(t, "2026-07-31")
			to := mustDate(t, "2026-07-01")

			_, err := resolveWindow(GetSummaryParams{Period: "range", From: &from, To: &to}, fixedNow, testLoc)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})
	})
}

func Test_startOfDay(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("時刻成分を落として暦日の開始時刻をlocのゾーンで返す", func(t *testing.T) {
			t.Parallel()

			got := startOfDay(time.Date(2026, time.July, 15, 23, 59, 59, 999, time.UTC), testLoc)

			assert.Equal(t, time.Date(2026, time.July, 15, 0, 0, 0, 0, testLoc), got)
		})

		t.Run("年月日は引数のロケーションのまま解釈しlocへ変換し直さない", func(t *testing.T) {
			t.Parallel()

			// 西経ゾーンへ変換してしまうと前日へずれる。利用者が指定した暦日を保つ契約。
			west := time.FixedZone("TEST-08", -8*60*60)

			got := startOfDay(time.Date(2026, time.July, 15, 2, 0, 0, 0, time.UTC), west)

			assert.Equal(t, time.Date(2026, time.July, 15, 0, 0, 0, 0, west), got)
		})
	})
}

func Test_startOfMonth(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("月の初日の開始時刻をlocのゾーンで返す", func(t *testing.T) {
			t.Parallel()

			got := startOfMonth(time.Date(2026, time.July, 15, 12, 0, 0, 0, testLoc), testLoc)

			assert.Equal(t, time.Date(2026, time.July, 1, 0, 0, 0, 0, testLoc), got)
		})

		t.Run("月初日そのものを渡しても同じ値を返す", func(t *testing.T) {
			t.Parallel()

			first := time.Date(2026, time.July, 1, 0, 0, 0, 0, testLoc)

			assert.Equal(t, first, startOfMonth(first, testLoc))
		})
	})
}

func Test_toStatusCountViews(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("集計結果の順序を保って出力DTOへ写像する", func(t *testing.T) {
			t.Parallel()

			first := uuidtestkit.NewTestFromSalt(t, "uc_dash_view_first")
			second := uuidtestkit.NewTestFromSalt(t, "uc_dash_view_second")

			got := toStatusCountViews([]query.PurchaseStatusCountResult{
				{StatusID: first, StatusName: "未処理", Count: 3},
				{StatusID: second, StatusName: "完了", Count: 5},
			})

			require.Len(t, got, 2)
			assert.Equal(t, StatusCountView{StatusID: first, StatusName: "未処理", Count: 3}, got[0])
			assert.Equal(t, StatusCountView{StatusID: second, StatusName: "完了", Count: 5}, got[1])
		})

		t.Run("集計結果が空の場合nilでない空スライスを返す", func(t *testing.T) {
			t.Parallel()

			got := toStatusCountViews(nil)

			assert.NotNil(t, got)
			assert.Empty(t, got)
		})
	})
}
