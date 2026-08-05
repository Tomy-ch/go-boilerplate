package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	dashboardhandler "go-boilerplate/internal/controller/handler/v1/dashboard"
	"go-boilerplate/internal/controller/handler/v1/dashboard/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	dashboarduc "go-boilerplate/internal/usecase/dashboard"
	mock_dashboarduc "go-boilerplate/internal/usecase/dashboard/mock"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const dashboardSummaryPath = "/v1/dashboard/summary"

func TestV1DashboardSummary_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みで横断集計がDashboardSummaryResponseで返る", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).Return(dashboarduc.SummaryView{
				SalesAmount: 450000,
				SalesCount:  12,
				PurchaseStatusCounts: []dashboarduc.StatusCountView{
					{StatusID: uuidtestkit.NewTestFromSalt(t, "int_dash_unprocessed"), StatusName: "未処理", Count: 2},
					{StatusID: uuidtestkit.NewTestFromSalt(t, "int_dash_completed"), StatusName: "完了", Count: 10},
				},
				TotalProductCount:     120,
				PublishedProductCount: 98,
			}, nil)

			dashboardhandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_dash_admin"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, dashboardSummaryPath, nil, headers)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[gen.DashboardSummaryResponse](t, actual)
		})

		t.Run("期間指定のクエリパラメータがユースケースへ届く", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var captured dashboarduc.GetSummaryParams
			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *auth.Authn, p dashboarduc.GetSummaryParams) (dashboarduc.SummaryView, error) {
					captured = p
					return dashboarduc.SummaryView{}, nil
				})

			dashboardhandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_dash_period"))
			path := dashboardSummaryPath + "?period=range&from=2026-07-01&to=2026-07-31"
			actual := StartServer(t, e).DoJSON(http.MethodGet, path, nil, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			assert.Equal(t, "range", captured.Period)
			require.NotNil(t, captured.From)
			require.NotNil(t, captured.To)
			assert.Equal(t, "2026-07-01", captured.From.Format(time.DateOnly))
			assert.Equal(t, "2026-07-31", captured.To.Format(time.DateOnly))
		})

		t.Run("集計対象が無くてもゼロ値の集計を200で返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(dashboarduc.SummaryView{}, nil)

			dashboardhandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_dash_empty"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, dashboardSummaryPath, nil, headers)
			AssertJSONResponseType[gen.DashboardSummaryResponse](t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未認証で401を返しUsecaseは呼ばれない", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			// 認証情報が無いためハンドラが早期に 401 で返し、Usecase は呼ばれない。
			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			dashboardhandler.BindHandler(e, tf, uc)

			// 認証ヘッダー（Authn）を張らずに呼び出す。
			actual := StartServer(t, e).DoJSON(http.MethodGet, dashboardSummaryPath, nil, nil)
			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("非adminの権限エラーを403へ変換する", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(dashboarduc.SummaryView{}, apperror.ErrPermissionDenied)

			dashboardhandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_dash_forbidden"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, dashboardSummaryPath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusForbidden)
		})

		t.Run("period=rangeでfrom/toが欠落する場合400を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(dashboarduc.SummaryView{}, apperror.ErrInvalidArgument)

			dashboardhandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_dash_badreq"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, dashboardSummaryPath+"?period=range", nil, headers)
			AssertErrorResponse(t, actual, http.StatusBadRequest)
		})

		t.Run("ErrInternalで500を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(dashboarduc.SummaryView{}, apperror.ErrInternal)

			dashboardhandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_dash_err"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, dashboardSummaryPath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}
