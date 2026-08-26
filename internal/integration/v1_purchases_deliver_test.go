package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	purchasesdeliver "go-boilerplate/internal/controller/handler/v1/purchases/detail/deliver"
	delivergen "go-boilerplate/internal/controller/handler/v1/purchases/detail/deliver/gen"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	mock_purchaseuc "go-boilerplate/internal/usecase/purchase/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

//nolint:dupl // 発送/配達完了は対称な admin 専用状態遷移で、HTTP 写像テストの構造の重複は不可避
func TestV1PurchasesDeliver_Integration(t *testing.T) {
	t.Parallel()

	const purchasePath = "/v1/purchases/PC-2026-0042/deliver"

	deliveredAt := time.Date(2026, time.July, 28, 9, 0, 0, 0, time.UTC)

	// availableDeliverAdmin は、配達完了を要求する admin の認証ヘッダを返すローカルヘルパーです。
	availableDeliverAdmin := func(t *testing.T, e *echo.Echo) http.Header {
		t.Helper()
		return MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "deliver_int_admin"))
	}

	deliverViewFixture := func(t *testing.T) purchaseuc.DeliverPurchaseView {
		t.Helper()
		return purchaseuc.DeliverPurchaseView{
			Code:           "deliver-int-code",
			UserID:         uuidtestkit.NewTestFromSalt(t, "deliver_int_user"),
			StatusID:       uuidtestkit.NewTestFromSalt(t, "deliver_int_status"),
			StatusName:     "配達済み",
			SubtotalAmount: 160000,
			TaxAmount:      16000,
			ShippingFee:    500,
			TotalAmount:    176500,
			Details: []purchaseuc.PurchaseDetailView{
				{ProductID: uuidtestkit.NewTestFromSalt(t, "deliver_int_product"), Quantity: 2, UnitPrice: decimaltestkit.MustParse(t, "800")},
			},
			OrderedAt:   time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
			DeliveredAt: &deliveredAt,
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("PATCHがadminで200とPurchaseDeliverResponseを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().DeliverPurchase(gomock.Any(), gomock.Any(), gomock.Any()).Return(deliverViewFixture(t), nil)

			purchasesdeliver.BindHandler(e, tf, uc)

			headers := availableDeliverAdmin(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			AssertJSONResponseType[delivergen.PurchaseDeliverResponse](t, actual)
		})

		t.Run("配達済みstatusと配達日時がレスポンスへ載る", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().DeliverPurchase(gomock.Any(), gomock.Any(), gomock.Any()).Return(deliverViewFixture(t), nil)

			purchasesdeliver.BindHandler(e, tf, uc)

			headers := availableDeliverAdmin(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			var body delivergen.PurchaseDeliverResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&body))
			assert.Equal(t, "配達済み", body.Status.Name)
			assert.Equal(t, deliveredAt, body.DeliveredAt.UTC())
		})

		t.Run("認証主体と購入IDがUsecaseへバインドされる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var capturedAuthn *auth.Authn
			var capturedCode string
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().DeliverPurchase(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, authn *auth.Authn, purchaseCode string) (purchaseuc.DeliverPurchaseView, error) {
					capturedAuthn = authn
					capturedCode = purchaseCode
					return deliverViewFixture(t), nil
				},
			)

			purchasesdeliver.BindHandler(e, tf, uc)

			headers := availableDeliverAdmin(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			require.NotNil(t, capturedAuthn)
			resolved, err := capturedAuthn.UserID()
			require.NoError(t, err)
			assert.Equal(t, uuidtestkit.NewTestFromSalt(t, "deliver_int_admin"), resolved)
			assert.Equal(t, "PC-2026-0042", capturedCode)
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
			uc.EXPECT().DeliverPurchase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			purchasesdeliver.BindHandler(e, tf, uc)

			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, nil)
			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("非adminの権限エラー(usecase)を403へ変換する", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().DeliverPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.DeliverPurchaseView{}, apperror.ErrPermissionDenied)

			purchasesdeliver.BindHandler(e, tf, uc)

			headers := availableDeliverAdmin(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusForbidden)
		})

		t.Run("存在しない購入で404を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().DeliverPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.DeliverPurchaseView{}, apperror.ErrNotFound)

			purchasesdeliver.BindHandler(e, tf, uc)

			headers := availableDeliverAdmin(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusNotFound)
		})

		t.Run("二重配達で409を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().DeliverPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.DeliverPurchaseView{}, domainpurchase.ErrAlreadyDelivered)

			purchasesdeliver.BindHandler(e, tf, uc)

			headers := availableDeliverAdmin(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusConflict)
		})

		t.Run("不正遷移（未発送・キャンセル済み等）で409を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().DeliverPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.DeliverPurchaseView{}, domainpurchase.ErrDeliverNotAllowed)

			purchasesdeliver.BindHandler(e, tf, uc)

			headers := availableDeliverAdmin(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusConflict)
		})
	})
}
