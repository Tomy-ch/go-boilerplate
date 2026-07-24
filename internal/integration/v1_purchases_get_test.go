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
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/idempotency"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	mock_purchaseuc "go-boilerplate/internal/usecase/purchase/mock"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestV1PurchasesGet_Integration(t *testing.T) {
	t.Parallel()

	summaryFixture := func() purchaseuc.PurchaseSummaryView {
		return purchaseuc.PurchaseSummaryView{
			Code:        "int-code",
			TotalAmount: 176500,
			StatusID:    uuid.NewTestFromSalt(t, "int_status"),
			StatusName:  "完了",
			OrderedAt:   time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
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
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any()).Return(
				&purchaseuc.PurchaseListView{Items: []purchaseuc.PurchaseSummaryView{summaryFixture()}, NextCursor: &nextCursor}, nil,
			)

			v1purchases.BindHandler(e, tf, uc, idempotency.Deps{})

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/purchases", nil, headers)
			AssertJSONResponseType[gen.PurchaseListResponse](t, actual)
		})

		t.Run("afterカーソル指定で認証主体のuserIDとcursorがUsecaseへバインドされる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var capturedUserID uuid.UUID
			var capturedCursor *paging.Cursor
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, userID uuid.UUID, cursor *paging.Cursor) (*purchaseuc.PurchaseListView, error) {
					capturedUserID = userID
					capturedCursor = cursor
					return &purchaseuc.PurchaseListView{Items: []purchaseuc.PurchaseSummaryView{}, NextCursor: nil}, nil
				},
			)

			v1purchases.BindHandler(e, tf, uc, idempotency.Deps{})

			headers := availablePurchaseUser(t, e)
			after := paging.EncodeCursor("2026-07-23T00:00:00Z", uuid.NewTestFromSalt(t, "int_after").String())
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/purchases?first=5&after="+after, nil, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			// availablePurchaseUser が注入する固定ユーザーIDが認証主体として解決されている。
			expectedUserID, err := uuid.Parse("b7f64798-7321-242b-e4ff-115f6a0b7810")
			require.NoError(t, err)
			assert.Equal(t, expectedUserID, capturedUserID)
			require.NotNil(t, capturedCursor)
			assert.True(t, capturedCursor.HasCursor())
		})

		t.Run("購入ゼロで200かつitemsが空配列でnextCursorがnull", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any()).Return(
				&purchaseuc.PurchaseListView{Items: []purchaseuc.PurchaseSummaryView{}, NextCursor: nil}, nil,
			)

			v1purchases.BindHandler(e, tf, uc, idempotency.Deps{})

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
			// NewCursor が失敗するため Usecase は呼ばれない。
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			v1purchases.BindHandler(e, tf, uc, idempotency.Deps{})

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
			// 認証情報が無いためハンドラが早期に 401 で返し、Usecase は呼ばれない。
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			v1purchases.BindHandler(e, tf, uc, idempotency.Deps{})

			// 認証ヘッダー（Authn）を張らずに呼び出す。
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
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, apperror.ErrInternal)

			v1purchases.BindHandler(e, tf, uc, idempotency.Deps{})

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/purchases", nil, headers)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}
