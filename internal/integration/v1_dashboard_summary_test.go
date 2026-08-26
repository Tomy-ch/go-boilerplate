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
	"go-boilerplate/internal/usecase/tools/timewindow"
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

			var captured timewindow.Window
			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *auth.Authn, w timewindow.Window) (dashboarduc.SummaryView, error) {
					captured = w
					return dashboarduc.SummaryView{}, nil
				})

			dashboardhandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_dash_period"))
			path := dashboardSummaryPath + "?orderedAfter=2026-07-01T00:00:00%2B09:00&orderedBefore=2026-08-01T00:00:00%2B09:00"
			actual := StartServer(t, e).DoJSON(http.MethodGet, path, nil, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			require.NotNil(t, captured.After())
			require.NotNil(t, captured.Before())
			assert.True(t, time.Date(2026, time.July, 1, 0, 0, 0, 0, time.FixedZone("", 9*60*60)).Equal(*captured.After()))
			assert.True(t, time.Date(2026, time.August, 1, 0, 0, 0, 0, time.FixedZone("", 9*60*60)).Equal(*captured.Before()))
		})

		t.Run("廃止したperiod/from/toを送っても無視され全期間として扱われる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var captured timewindow.Window
			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *auth.Authn, w timewindow.Window) (dashboarduc.SummaryView, error) {
					captured = w
					return dashboarduc.SummaryView{}, nil
				})

			dashboardhandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_dash_legacy"))
			path := dashboardSummaryPath + "?period=month&from=2026-07-01&to=2026-07-31"
			actual := StartServer(t, e).DoJSON(http.MethodGet, path, nil, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			assert.Nil(t, captured.After())
			assert.Nil(t, captured.Before())
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
			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			dashboardhandler.BindHandler(e, tf, uc)

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

		t.Run("orderedBeforeがorderedAfter以前の場合400を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			dashboardhandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_dash_badreq"))
			path := dashboardSummaryPath + "?orderedAfter=2026-08-01T00:00:00Z&orderedBefore=2026-07-01T00:00:00Z"
			actual := StartServer(t, e).DoJSON(http.MethodGet, path, nil, headers)
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
