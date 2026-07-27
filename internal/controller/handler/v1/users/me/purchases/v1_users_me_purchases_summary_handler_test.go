package usersmepurchases

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/users/me/purchases/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	summaryuc "go-boilerplate/internal/usecase/purchase/summary"
	mock_summaryuc "go-boilerplate/internal/usecase/purchase/summary/mock"
	"go-boilerplate/pkg/uuid"

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
	require.True(t, ctxhelper.SetAuthn(ctx, *authn.WithUserID(userID)))
	return ctx
}

// summaryViewFixture は、購入集計ビューを生成するテストヘルパーです。
func summaryViewFixture(t *testing.T) summaryuc.SummaryView {
	t.Helper()
	return summaryuc.SummaryView{
		TotalCount:  3,
		TotalAmount: 450,
		StatusBreakdown: []summaryuc.StatusCountView{
			{StatusID: uuid.NewTestFromSalt(t, "hs_unprocessed"), StatusName: "未処理", Count: 2, TotalAmount: 300},
			{StatusID: uuid.NewTestFromSalt(t, "hs_canceled"), StatusName: "キャンセル", Count: 1, TotalAmount: 150},
		},
	}
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	uc := mock_summaryuc.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, uc)

	routes := e.Router().Routes()
	require.Len(t, routes, 1)
	assert.Equal(t, http.MethodGet, routes[0].Method)
	assert.Equal(t, "/v1/users/me/purchases/summary", routes[0].Path)
}

func Test_server_GetUsersMePurchasesSummary(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証主体のAuthnをユースケースへ渡し集計を200で返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_summaryuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			userID := uuid.NewTestFromSalt(t, "hs_user")
			view := summaryViewFixture(t)
			uc.EXPECT().GetPurchaseSummary(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, authn *auth.Authn) (summaryuc.SummaryView, error) {
					uid, uerr := authn.UserID()
					require.NoError(t, uerr)
					assert.Equal(t, userID, uid)
					return view, nil
				})

			resp, err := s.GetUsersMePurchasesSummary(authnContext(t, userID), gen.GetUsersMePurchasesSummaryRequestObject{})
			require.NoError(t, err)

			r, ok := resp.(gen.GetUsersMePurchasesSummary200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, int64(3), r.TotalCount)
			assert.Equal(t, int64(450), r.TotalAmount)
			require.Len(t, r.StatusBreakdown, 2)
			assert.Equal(t, "未処理", r.StatusBreakdown[0].Status.Name)
			assert.Equal(t, int64(2), r.StatusBreakdown[0].Count)
			assert.Equal(t, int64(300), r.StatusBreakdown[0].TotalAmount)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証情報が無い場合、ErrUnauthenticatedUserを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_summaryuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			_, err := s.GetUsersMePurchasesSummary(context.Background(), gen.GetUsersMePurchasesSummaryRequestObject{})
			require.ErrorIs(t, err, ErrUnauthenticatedUser)
		})

		t.Run("ユースケースがエラーを返した場合はそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_summaryuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().GetPurchaseSummary(gomock.Any(), gomock.Any()).
				Return(summaryuc.SummaryView{}, apperror.ErrInternal)

			_, err := s.GetUsersMePurchasesSummary(authnContext(t, uuid.NewTestFromSalt(t, "hs_user_err")),
				gen.GetUsersMePurchasesSummaryRequestObject{})
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func Test_toPurchaseAggregateResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("集計ビューをステータス別内訳込みのレスポンスへ写像する", func(t *testing.T) {
			t.Parallel()

			view := summaryViewFixture(t)
			r := toPurchaseAggregateResponse(view)

			assert.Equal(t, view.TotalCount, r.TotalCount)
			assert.Equal(t, view.TotalAmount, r.TotalAmount)
			require.Len(t, r.StatusBreakdown, len(view.StatusBreakdown))
			for i, b := range view.StatusBreakdown {
				assert.Equal(t, b.StatusID.ToPrimitive(), r.StatusBreakdown[i].Status.Id)
				assert.Equal(t, b.StatusName, r.StatusBreakdown[i].Status.Name)
				assert.Equal(t, b.Count, r.StatusBreakdown[i].Count)
				assert.Equal(t, b.TotalAmount, r.StatusBreakdown[i].TotalAmount)
			}
		})

		t.Run("内訳が空の場合はnilではない空配列のレスポンスを返す", func(t *testing.T) {
			t.Parallel()

			r := toPurchaseAggregateResponse(summaryuc.SummaryView{})

			assert.Equal(t, int64(0), r.TotalCount)
			assert.Equal(t, int64(0), r.TotalAmount)
			assert.NotNil(t, r.StatusBreakdown)
			assert.Empty(t, r.StatusBreakdown)
		})
	})
}
