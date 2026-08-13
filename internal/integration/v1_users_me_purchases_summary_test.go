package integration

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	usersmepurchases "go-boilerplate/internal/controller/handler/v1/users/me/purchases"
	"go-boilerplate/internal/controller/handler/v1/users/me/purchases/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	summaryuc "go-boilerplate/internal/usecase/purchase/summary"
	mock_summaryuc "go-boilerplate/internal/usecase/purchase/summary/mock"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const purchaseSummaryPath = "/v1/users/me/purchases/summary"

func TestV1UsersMePurchasesSummary_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みで自分の購入集計がPurchaseAggregateResponseで返る", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_summaryuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchaseSummary(gomock.Any(), gomock.Any()).Return(summaryuc.SummaryView{
				TotalCount:  3,
				TotalAmount: 450,
				StatusBreakdown: []summaryuc.StatusCountView{
					{StatusID: uuidtestkit.NewTestFromSalt(t, "int_sm_unprocessed"), StatusName: "未処理", Count: 2, TotalAmount: 300},
					{StatusID: uuidtestkit.NewTestFromSalt(t, "int_sm_canceled"), StatusName: "キャンセル", Count: 1, TotalAmount: 150},
				},
			}, nil)

			usersmepurchases.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_sm_user"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, purchaseSummaryPath, nil, headers)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[gen.PurchaseAggregateResponse](t, actual)
		})

		t.Run("集計対象が認証主体のuserIDに限定されている", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			userID := uuidtestkit.NewTestFromSalt(t, "int_sm_owner")
			var capturedUserID uuid.UUID
			uc := mock_summaryuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchaseSummary(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, authn *auth.Authn) (summaryuc.SummaryView, error) {
					id, err := authn.UserID()
					require.NoError(t, err)
					capturedUserID = id
					return summaryuc.SummaryView{}, nil
				},
			)

			usersmepurchases.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, userID)
			actual := StartServer(t, e).DoJSON(http.MethodGet, purchaseSummaryPath, nil, headers)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			assert.Equal(t, userID, capturedUserID)
		})

		t.Run("購入が0件でもゼロ値の集計を200で返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_summaryuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchaseSummary(gomock.Any(), gomock.Any()).Return(summaryuc.SummaryView{}, nil)

			usersmepurchases.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_sm_empty"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, purchaseSummaryPath, nil, headers)
			AssertJSONResponseType[gen.PurchaseAggregateResponse](t, actual)
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

			uc := mock_summaryuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchaseSummary(gomock.Any(), gomock.Any()).Times(0)

			usersmepurchases.BindHandler(e, tf, uc)

			actual := StartServer(t, e).DoJSON(http.MethodGet, purchaseSummaryPath, nil, nil)
			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("ErrInternalで500を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_summaryuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchaseSummary(gomock.Any(), gomock.Any()).Return(summaryuc.SummaryView{}, apperror.ErrInternal)

			usersmepurchases.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_sm_err"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, purchaseSummaryPath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}
