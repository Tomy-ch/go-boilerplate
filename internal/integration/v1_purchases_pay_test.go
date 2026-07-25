package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	purchasespay "go-boilerplate/internal/controller/handler/v1/purchases/detail/pay"
	paygen "go-boilerplate/internal/controller/handler/v1/purchases/detail/pay/gen"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/observability"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	mock_purchaseuc "go-boilerplate/internal/usecase/purchase/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

//nolint:dupl // 支払い/キャンセルの HTTP 統合テストは状態遷移が対称で構造の重複は不可避
func TestV1PurchasesPay_Integration(t *testing.T) {
	t.Parallel()

	const purchasePath = "/v1/purchases/0190b0d4-7b1a-7c2e-9f3a-1b2c3d4e5f60/pay"

	payViewFixture := func(t *testing.T) purchaseuc.PayPurchaseView {
		t.Helper()
		paidAt := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
		return purchaseuc.PayPurchaseView{
			ID:             uuid.NewTestFromSalt(t, "pay_int_id"),
			Code:           "pay-int-code",
			UserID:         uuid.NewTestFromSalt(t, "pay_int_user"),
			StatusID:       uuid.NewTestFromSalt(t, "pay_int_status"),
			StatusName:     "支払い済み",
			SubtotalAmount: 160000,
			TaxAmount:      16000,
			ShippingFee:    500,
			TotalAmount:    176500,
			Details: []purchaseuc.PurchaseDetailView{
				{ProductID: uuid.NewTestFromSalt(t, "pay_int_product"), Quantity: 2, UnitPrice: decimaltestkit.MustParse(t, "800")},
			},
			OrderedAt: time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
			PaidAt:    &paidAt,
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("PATCHが認証済みで200とPurchasePayResponseを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().PayPurchase(gomock.Any(), gomock.Any()).Return(payViewFixture(t), nil)

			purchasespay.BindHandler(e, tf, uc)

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			AssertJSONResponseType[paygen.PurchasePayResponse](t, actual)
		})

		t.Run("認証主体のuserIDと購入IDがUsecaseへバインドされる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var captured purchaseuc.PayPurchaseParams
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().PayPurchase(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params purchaseuc.PayPurchaseParams) (purchaseuc.PayPurchaseView, error) {
					captured = params
					return payViewFixture(t), nil
				},
			)

			purchasespay.BindHandler(e, tf, uc)

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			expectedUserID, err := uuid.Parse("b7f64798-7321-242b-e4ff-115f6a0b7810")
			require.NoError(t, err)
			assert.Equal(t, expectedUserID, captured.UserID)
			assert.Equal(t, "0190b0d4-7b1a-7c2e-9f3a-1b2c3d4e5f60", captured.PurchaseID.String())
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
			uc.EXPECT().PayPurchase(gomock.Any(), gomock.Any()).Times(0)

			purchasespay.BindHandler(e, tf, uc)

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
			uc.EXPECT().PayPurchase(gomock.Any(), gomock.Any()).Return(purchaseuc.PayPurchaseView{}, apperror.ErrNotFound)

			purchasespay.BindHandler(e, tf, uc)

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusNotFound)
		})

		t.Run("二重支払いで409を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().PayPurchase(gomock.Any(), gomock.Any()).Return(purchaseuc.PayPurchaseView{}, domainpurchase.ErrAlreadyPaid)

			purchasespay.BindHandler(e, tf, uc)

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusConflict)
		})

		t.Run("不正遷移で409を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().PayPurchase(gomock.Any(), gomock.Any()).Return(purchaseuc.PayPurchaseView{}, domainpurchase.ErrPayNotAllowed)

			purchasespay.BindHandler(e, tf, uc)

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusConflict)
		})
	})
}
