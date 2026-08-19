package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	v1purchases "go-boilerplate/internal/controller/handler/v1/purchases"
	"go-boilerplate/internal/controller/handler/v1/purchases/gen"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/idempotency"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	mock_purchaseuc "go-boilerplate/internal/usecase/purchase/mock"
	"go-boilerplate/internal/usecase/purchase/period"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestV1PurchasesGet_Integration(t *testing.T) {
	t.Parallel()

	summaryFixture := func() purchaseuc.PurchaseSummaryView {
		return purchaseuc.PurchaseSummaryView{
			Code:          "int-code",
			TotalAmount:   176500,
			StatusID:      uuidtestkit.NewTestFromSalt(t, "int_status"),
			StatusCode:    domainpurchase.StatusCompleted.Code(),
			StatusName:    "完了",
			FirstItemName: "ワイヤレスイヤホン",
			ItemCount:     3,
			OrderedAt:     time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/purchasesが認証済みでPurchaseListResponseを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			nextCursor := "next-opaque-cursor"
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
				&purchaseuc.PurchaseListView{Items: []purchaseuc.PurchaseSummaryView{summaryFixture()}, NextCursor: &nextCursor}, nil,
			)

			v1purchases.BindHandler(e, tf, uc, nil, idempotency.Deps{})

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/purchases", nil, headers)
			AssertJSONResponseType[gen.PurchaseListResponse](t, actual)
		})

		t.Run("一覧の要素が明細の要約（先頭商品名・点数）を載せて返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
				&purchaseuc.PurchaseListView{Items: []purchaseuc.PurchaseSummaryView{summaryFixture()}}, nil,
			)

			v1purchases.BindHandler(e, tf, uc, nil, idempotency.Deps{})

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/purchases", nil, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			var body gen.PurchaseListResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&body))
			require.Len(t, body.Items, 1)
			assert.Equal(t, "ワイヤレスイヤホン", body.Items[0].FirstItemName)
			assert.EqualValues(t, 3, body.Items[0].ItemCount)
		})

		t.Run("一覧の要素がステータスの業務キーcodeを載せて返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
				&purchaseuc.PurchaseListView{Items: []purchaseuc.PurchaseSummaryView{summaryFixture()}}, nil,
			)

			v1purchases.BindHandler(e, tf, uc, nil, idempotency.Deps{})

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/purchases", nil, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			var body gen.PurchaseListResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&body))
			require.Len(t, body.Items, 1)
			assert.Equal(t, "完了", body.Items[0].Status.Name)
			assert.EqualValues(t, domainpurchase.StatusCompleted.Code(), body.Items[0].Status.Code)
		})

		t.Run("afterカーソル指定で認証主体のuserIDとcursorがUsecaseへバインドされる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var capturedUserID uuid.UUID
			var capturedCursor *paging.Cursor
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, userID uuid.UUID, cursor *paging.Cursor, _ period.Spec) (*purchaseuc.PurchaseListView, error) {
					capturedUserID = userID
					capturedCursor = cursor
					return &purchaseuc.PurchaseListView{Items: []purchaseuc.PurchaseSummaryView{}, NextCursor: nil}, nil
				},
			)

			v1purchases.BindHandler(e, tf, uc, nil, idempotency.Deps{})

			headers := availablePurchaseUser(t, e)
			after := paging.EncodeCursor("2026-07-23T00:00:00Z", uuidtestkit.NewTestFromSalt(t, "int_after").String())
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/purchases?first=5&after="+after, nil, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			// availablePurchaseUser が注入する固定ユーザーIDが認証主体として解決されている。
			expectedUserID, err := uuid.Parse("b7f64798-7321-242b-e4ff-115f6a0b7810")
			require.NoError(t, err)
			assert.Equal(t, expectedUserID, capturedUserID)
			require.NotNil(t, capturedCursor)
			assert.True(t, capturedCursor.HasCursor())
		})

		t.Run("期間指定のクエリがUsecaseの期間指定へバインドされる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var captured period.Spec
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, _ *paging.Cursor, spec period.Spec) (*purchaseuc.PurchaseListView, error) {
					captured = spec
					return &purchaseuc.PurchaseListView{Items: []purchaseuc.PurchaseSummaryView{}, NextCursor: nil}, nil
				},
			)

			v1purchases.BindHandler(e, tf, uc, nil, idempotency.Deps{})

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(
				http.MethodGet, "/v1/purchases?period=range&from=2026-01-21&to=2026-01-31", nil, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			assert.Equal(t, period.KindRange, captured.Kind)
			require.NotNil(t, captured.From)
			require.NotNil(t, captured.To)
			assert.Equal(t, "2026-01-21", captured.From.Format(time.DateOnly))
			assert.Equal(t, "2026-01-31", captured.To.Format(time.DateOnly))
		})

		t.Run("購入ゼロで200かつitemsが空配列でnextCursorがnull", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
				&purchaseuc.PurchaseListView{Items: []purchaseuc.PurchaseSummaryView{}, NextCursor: nil}, nil,
			)

			v1purchases.BindHandler(e, tf, uc, nil, idempotency.Deps{})

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/purchases", nil, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			var body gen.PurchaseListResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&body))
			assert.Empty(t, body.Items)
			assert.Nil(t, body.NextCursor)
			assert.False(t, body.HasNext)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不正なafterで400を返しUsecaseは呼ばれない", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			v1purchases.BindHandler(e, tf, uc, nil, idempotency.Deps{})

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/purchases?after=%21%21%21", nil, headers)
			AssertErrorResponse(t, actual, http.StatusBadRequest)
		})

		t.Run("未認証で401を返しUsecaseは呼ばれない", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			v1purchases.BindHandler(e, tf, uc, nil, idempotency.Deps{})

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/purchases", nil, nil)
			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("ErrInternalで500を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, apperror.ErrInternal)

			v1purchases.BindHandler(e, tf, uc, nil, idempotency.Deps{})

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/purchases", nil, headers)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}
