package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	usersmepurchases "go-boilerplate/internal/controller/handler/v1/users/me/purchases"
	"go-boilerplate/internal/controller/handler/v1/users/me/purchases/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/purchase/period"
	summaryuc "go-boilerplate/internal/usecase/purchase/summary"
	mock_summaryuc "go-boilerplate/internal/usecase/purchase/summary/mock"
	"go-boilerplate/pkg/decimal"
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
			uc.EXPECT().GetPurchaseSummary(gomock.Any(), gomock.Any(), gomock.Any()).Return(summaryuc.SummaryView{
				TotalCount:  3,
				TotalAmount: 450,
				StatusBreakdown: []summaryuc.StatusCountView{
					{StatusID: uuidtestkit.NewTestFromSalt(t, "int_sm_unprocessed"), StatusName: "未処理", Count: 2, TotalAmount: 300},
					{StatusID: uuidtestkit.NewTestFromSalt(t, "int_sm_paid"), StatusName: "支払い済み", Count: 1, TotalAmount: 150},
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
			uc.EXPECT().GetPurchaseSummary(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, authn *auth.Authn, _ summaryuc.GetSummaryParams) (summaryuc.SummaryView, error) {
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

		t.Run("期間とグループ化のクエリがユースケースの入力へ届く", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var captured summaryuc.GetSummaryParams
			uc := mock_summaryuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchaseSummary(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *auth.Authn, params summaryuc.GetSummaryParams) (summaryuc.SummaryView, error) {
					captured = params
					return summaryuc.SummaryView{}, nil
				},
			)

			usersmepurchases.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_sm_query"))
			actual := StartServer(t, e).DoJSON(
				http.MethodGet, purchaseSummaryPath+"?period=recent&days=10&groupBy=category,product", nil, headers)
			assert.Equal(t, http.StatusOK, actual.StatusCode)

			assert.Equal(t, period.KindRecent, captured.Period.Kind)
			require.NotNil(t, captured.Period.Days)
			assert.Equal(t, 10, *captured.Period.Days)
			// カンマ区切りの配列がその順序のまま bind されることを HTTP 経路で固定する。
			assert.Equal(t, []summaryuc.GroupKind{summaryuc.GroupByCategory, summaryuc.GroupByProduct}, captured.GroupBy)
		})

		t.Run("グループ化した集計が入れ子のオブジェクトで直列化される", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			productID := uuidtestkit.NewTestFromSalt(t, "int_sm_product").String()
			itemsTotal, err := decimal.Parse("980.00")
			require.NoError(t, err)

			uc := mock_summaryuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchaseSummary(gomock.Any(), gomock.Any(), gomock.Any()).Return(summaryuc.SummaryView{
				ItemsTotal: itemsTotal,
				Groups: map[string]summaryuc.GroupNodeView{
					"電子機器": {
						Name:       "電子機器",
						ItemsTotal: itemsTotal,
						Groups:     map[string]summaryuc.GroupNodeView{productID: {Name: "ノートPC", ItemsTotal: itemsTotal}},
					},
				},
			}, nil)

			usersmepurchases.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_sm_groups"))
			actual := StartServer(t, e).DoJSON(
				http.MethodGet, purchaseSummaryPath+"?groupBy=category,product", nil, headers)
			assert.Equal(t, http.StatusOK, actual.StatusCode)

			resBody, err := io.ReadAll(actual.Body)
			require.NoError(t, err)
			var body gen.PurchaseAggregateResponse
			require.NoError(t, json.Unmarshal(resBody, &body))

			require.NotNil(t, body.Groups)
			groups := *body.Groups
			require.Contains(t, groups, "電子機器")
			require.NotNil(t, groups["電子機器"].Groups)
			require.Contains(t, *groups["電子機器"].Groups, productID)
			assert.Equal(t, "ノートPC", (*groups["電子機器"].Groups)[productID].Name)
		})

		t.Run("購入が0件でもゼロ値の集計を200で返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_summaryuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchaseSummary(gomock.Any(), gomock.Any(), gomock.Any()).Return(summaryuc.SummaryView{}, nil)

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
			uc.EXPECT().GetPurchaseSummary(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

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
			uc.EXPECT().GetPurchaseSummary(gomock.Any(), gomock.Any(), gomock.Any()).Return(summaryuc.SummaryView{}, apperror.ErrInternal)

			usersmepurchases.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_sm_err"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, purchaseSummaryPath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}
