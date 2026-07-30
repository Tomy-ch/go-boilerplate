package dashboard

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/dashboard/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	dashboarduc "go-boilerplate/internal/usecase/dashboard"
	mock_dashboarduc "go-boilerplate/internal/usecase/dashboard/mock"
	"go-boilerplate/pkg/uuid"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// authnContext は、内部ユーザー ID を解決済みの認証コンテキストを返すテストヘルパーです。
func authnContext(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()
	ctx := ctxhelper.WithAuthn(context.Background())
	authn, err := auth.New("subject", "issuer", nil, nil)
	require.NoError(t, err)

	resolved, err := authn.WithUserID(userID)
	require.NoError(t, err)
	require.True(t, ctxhelper.SetAuthn(ctx, *resolved))
	return ctx
}

// summaryViewFixture は、ダッシュボード集計ビューを生成するテストヘルパーです。
func summaryViewFixture(t *testing.T) dashboarduc.SummaryView {
	t.Helper()
	return dashboarduc.SummaryView{
		SalesAmount: 450000,
		SalesCount:  12,
		PurchaseStatusCounts: []dashboarduc.StatusCountView{
			{StatusID: uuid.NewTestFromSalt(t, "hd_unprocessed"), StatusName: "未処理", Count: 2},
			{StatusID: uuid.NewTestFromSalt(t, "hd_completed"), StatusName: "完了", Count: 10},
		},
		TotalProductCount:     120,
		PublishedProductCount: 98,
	}
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	uc := mock_dashboarduc.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, uc)

	routes := e.Router().Routes()
	require.Len(t, routes, 1)
	assert.Equal(t, http.MethodGet, routes[0].Method)
	assert.Equal(t, "/v1/dashboard/summary", routes[0].Path)
}

func Test_server_GetDashboardSummary(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証主体のAuthnをユースケースへ渡し集計を200で返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			userID := uuid.NewTestFromSalt(t, "hd_user")
			view := summaryViewFixture(t)
			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, authn *auth.Authn, _ dashboarduc.GetSummaryParams) (dashboarduc.SummaryView, error) {
					uid, uerr := authn.UserID()
					require.NoError(t, uerr)
					assert.Equal(t, userID, uid)
					return view, nil
				})

			resp, err := s.GetDashboardSummary(authnContext(t, userID), gen.GetDashboardSummaryRequestObject{})
			require.NoError(t, err)

			r, ok := resp.(gen.GetDashboardSummary200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, int64(450000), r.SalesAmount)
			assert.Equal(t, int64(12), r.SalesCount)
			assert.Equal(t, int64(120), r.TotalProductCount)
			assert.Equal(t, int64(98), r.PublishedProductCount)
			require.Len(t, r.PurchaseStatusCounts, 2)
			assert.Equal(t, "未処理", r.PurchaseStatusCounts[0].Status.Name)
			assert.Equal(t, int64(2), r.PurchaseStatusCounts[0].Count)
		})

		t.Run("period・from・toのクエリパラメータをユースケース入力へそのまま渡す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			period := gen.GetDashboardSummaryParamsPeriodRange
			from := openapi_types.Date{Time: time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)}
			to := openapi_types.Date{Time: time.Date(2026, time.July, 31, 0, 0, 0, 0, time.UTC)}

			var captured dashboarduc.GetSummaryParams
			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *auth.Authn, p dashboarduc.GetSummaryParams) (dashboarduc.SummaryView, error) {
					captured = p
					return dashboarduc.SummaryView{}, nil
				})

			_, err := s.GetDashboardSummary(
				authnContext(t, uuid.NewTestFromSalt(t, "hd_params")),
				gen.GetDashboardSummaryRequestObject{
					Params: gen.GetDashboardSummaryParams{Period: &period, From: &from, To: &to},
				},
			)
			require.NoError(t, err)

			assert.Equal(t, "range", captured.Period)
			require.NotNil(t, captured.From)
			require.NotNil(t, captured.To)
			assert.Equal(t, from.Time, *captured.From)
			assert.Equal(t, to.Time, *captured.To)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証情報が無い場合、ErrUnauthenticatedUserを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			_, err := s.GetDashboardSummary(context.Background(), gen.GetDashboardSummaryRequestObject{})
			require.ErrorIs(t, err, ErrUnauthenticatedUser)
		})

		t.Run("ユースケースがエラーを返した場合はそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(dashboarduc.SummaryView{}, apperror.ErrPermissionDenied)

			_, err := s.GetDashboardSummary(
				authnContext(t, uuid.NewTestFromSalt(t, "hd_user_err")),
				gen.GetDashboardSummaryRequestObject{},
			)
			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})
	})
}

func Test_periodParam(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未指定は空文字、指定時はその文字列を返す", func(t *testing.T) {
			t.Parallel()

			month := gen.GetDashboardSummaryParamsPeriodMonth

			assert.Empty(t, periodParam(nil))
			assert.Equal(t, "month", periodParam(&month))
		})
	})
}

func Test_dateParam(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未指定はnil、指定時は暦日の時刻を返す", func(t *testing.T) {
			t.Parallel()

			date := openapi_types.Date{Time: time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)}

			assert.Nil(t, dateParam(nil))
			got := dateParam(&date)
			require.NotNil(t, got)
			assert.Equal(t, date.Time, *got)
		})
	})
}

func Test_toDashboardSummaryResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ユースケースDTOの各集計値とステータス別内訳をレスポンスへ写像する", func(t *testing.T) {
			t.Parallel()

			view := summaryViewFixture(t)

			got := toDashboardSummaryResponse(view)

			assert.Equal(t, int64(450000), got.SalesAmount)
			assert.Equal(t, int64(12), got.SalesCount)
			assert.Equal(t, int64(120), got.TotalProductCount)
			assert.Equal(t, int64(98), got.PublishedProductCount)
			require.Len(t, got.PurchaseStatusCounts, 2)
			assert.Equal(t, view.PurchaseStatusCounts[0].StatusID.ToPrimitive(), got.PurchaseStatusCounts[0].Status.Id)
			assert.Equal(t, "未処理", got.PurchaseStatusCounts[0].Status.Name)
			assert.Equal(t, int64(2), got.PurchaseStatusCounts[0].Count)
			assert.Equal(t, "完了", got.PurchaseStatusCounts[1].Status.Name)
			assert.Equal(t, int64(10), got.PurchaseStatusCounts[1].Count)
		})

		t.Run("ステータス別内訳が空の場合nilでない空スライスを返す", func(t *testing.T) {
			t.Parallel()

			got := toDashboardSummaryResponse(dashboarduc.SummaryView{})

			assert.NotNil(t, got.PurchaseStatusCounts)
			assert.Empty(t, got.PurchaseStatusCounts)
		})
	})
}
