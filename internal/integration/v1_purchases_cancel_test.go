package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	purchasescancel "go-boilerplate/internal/controller/handler/v1/purchases/detail/cancel"
	cancelgen "go-boilerplate/internal/controller/handler/v1/purchases/detail/cancel/gen"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/observability"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	mock_purchaseuc "go-boilerplate/internal/usecase/purchase/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

//nolint:dupl // 支払い/キャンセルの HTTP 統合テストは状態遷移が対称で構造の重複は不可避
func TestV1PurchasesCancel_Integration(t *testing.T) {
	t.Parallel()

	const purchasePath = "/v1/purchases/PC-2026-0042/cancel"

	cancelViewFixture := func(t *testing.T) purchaseuc.CancelPurchaseView {
		t.Helper()
		canceledAt := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
		return purchaseuc.CancelPurchaseView{
			Code:           "cancel-int-code",
			UserID:         uuidtestkit.NewTestFromSalt(t, "cancel_int_user"),
			StatusID:       uuidtestkit.NewTestFromSalt(t, "cancel_int_status"),
			StatusName:     "キャンセル",
			SubtotalAmount: 160000,
			TaxAmount:      16000,
			ShippingFee:    500,
			TotalAmount:    176500,
			Details: []purchaseuc.PurchaseDetailView{
				{ProductID: uuidtestkit.NewTestFromSalt(t, "cancel_int_product"), Quantity: 2, UnitPrice: decimaltestkit.MustParse(t, "800")},
			},
			OrderedAt:  time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
			CanceledAt: &canceledAt,
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("PATCHが認証済みで200とPurchaseCancelResponseを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().CancelPurchase(gomock.Any(), gomock.Any()).Return(cancelViewFixture(t), nil)

			purchasescancel.BindHandler(e, tf, uc)

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			AssertJSONResponseType[cancelgen.PurchaseCancelResponse](t, actual)
		})

		t.Run("認証主体のuserIDと購入IDがUsecaseへバインドされる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var captured purchaseuc.CancelPurchaseParams
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().CancelPurchase(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params purchaseuc.CancelPurchaseParams) (purchaseuc.CancelPurchaseView, error) {
					captured = params
					return cancelViewFixture(t), nil
				},
			)

			purchasescancel.BindHandler(e, tf, uc)

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			expectedUserID, err := uuid.Parse("b7f64798-7321-242b-e4ff-115f6a0b7810")
			require.NoError(t, err)
			assert.Equal(t, expectedUserID, captured.UserID)
			assert.Equal(t, "PC-2026-0042", captured.PurchaseCode)
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

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().CancelPurchase(gomock.Any(), gomock.Any()).Times(0)

			purchasescancel.BindHandler(e, tf, uc)

			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, nil)
			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("他人の購入・不存在で404を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().CancelPurchase(gomock.Any(), gomock.Any()).Return(purchaseuc.CancelPurchaseView{}, apperror.ErrNotFound)

			purchasescancel.BindHandler(e, tf, uc)

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusNotFound)
		})

		t.Run("不正遷移で409を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().CancelPurchase(gomock.Any(), gomock.Any()).Return(purchaseuc.CancelPurchaseView{}, domainpurchase.ErrCancelNotAllowed)

			purchasescancel.BindHandler(e, tf, uc)

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusConflict)
		})
	})
}
