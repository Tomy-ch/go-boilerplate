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
	"go-boilerplate/internal/usecase/tools/timewindow"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

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
			{
				StatusID:   uuidtestkit.NewTestFromSalt(t, "hd_unprocessed"),
				StatusCode: 1,
				StatusName: "未処理",
				Count:      2,
			},
			{
				StatusID:   uuidtestkit.NewTestFromSalt(t, "hd_completed"),
				StatusCode: 5,
				StatusName: "完了",
				Count:      10,
			},
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

			userID := uuidtestkit.NewTestFromSalt(t, "hd_user")
			view := summaryViewFixture(t)
			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, authn *auth.Authn, _ timewindow.Window) (dashboarduc.SummaryView, error) {
					uid, uerr := authn.UserID()
					require.NoError(t, uerr)
					assert.Equal(t, userID, uid)
					return view, nil
				},
			)

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

		t.Run("orderedAfter・orderedBeforeのクエリパラメータを対象期間へ変換して渡す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			after := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
			before := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)

			var captured timewindow.Window
			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *auth.Authn, w timewindow.Window) (dashboarduc.SummaryView, error) {
					captured = w
					return dashboarduc.SummaryView{}, nil
				},
			)

			_, err := s.GetDashboardSummary(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "hd_params")),
				gen.GetDashboardSummaryRequestObject{
					Params: gen.GetDashboardSummaryParams{OrderedAfter: &after, OrderedBefore: &before},
				},
			)
			require.NoError(t, err)

			require.NotNil(t, captured.After())
			require.NotNil(t, captured.Before())
			assert.Equal(t, after, *captured.After())
			assert.Equal(t, before, *captured.Before())
		})

		t.Run("期間のクエリパラメータ未指定は境界を持たない対象期間として渡す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			var captured timewindow.Window
			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *auth.Authn, w timewindow.Window) (dashboarduc.SummaryView, error) {
					captured = w
					return dashboarduc.SummaryView{}, nil
				},
			)

			_, err := s.GetDashboardSummary(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "hd_noparams")),
				gen.GetDashboardSummaryRequestObject{},
			)
			require.NoError(t, err)

			assert.Nil(t, captured.After())
			assert.Nil(t, captured.Before())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("orderedBeforeがorderedAfter以前の場合、ユースケースを呼ばずErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}
			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			after := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
			before := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)

			_, err := s.GetDashboardSummary(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "hd_badwindow")),
				gen.GetDashboardSummaryRequestObject{
					Params: gen.GetDashboardSummaryParams{OrderedAfter: &after, OrderedBefore: &before},
				},
			)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("認証情報が無い場合、ErrUnauthenticatedUserを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			_, err := s.GetDashboardSummary(context.Background(), gen.GetDashboardSummaryRequestObject{})
			require.ErrorIs(t, err, ctxhelper.ErrUnauthenticatedUser)
		})

		t.Run("ユースケースがエラーを返した場合はそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_dashboarduc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().GetDashboardSummary(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(dashboarduc.SummaryView{}, apperror.ErrPermissionDenied)

			_, err := s.GetDashboardSummary(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "hd_user_err")),
				gen.GetDashboardSummaryRequestObject{},
			)
			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
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
			assert.EqualValues(t, view.PurchaseStatusCounts[1].StatusCode, got.PurchaseStatusCounts[1].Status.Code)
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
