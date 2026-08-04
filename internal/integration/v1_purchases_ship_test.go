package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	purchasesship "go-boilerplate/internal/controller/handler/v1/purchases/detail/ship"
	shipgen "go-boilerplate/internal/controller/handler/v1/purchases/detail/ship/gen"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	mock_purchaseuc "go-boilerplate/internal/usecase/purchase/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

//nolint:dupl // 発送/配達完了は対称な admin 専用状態遷移で、HTTP 写像テストの構造の重複は不可避
func TestV1PurchasesShip_Integration(t *testing.T) {
	t.Parallel()

	const purchasePath = "/v1/purchases/0190b0d4-7b1a-7c2e-9f3a-1b2c3d4e5f60/ship"

	// availableShipAdmin は、発送を要求する admin の認証ヘッダを返すローカルヘルパーです。
	// EnvTest の Authorizer は allowall 固定のため、非 admin との差は usecase の戻り値で表現します。
	availableShipAdmin := func(t *testing.T, e *echo.Echo) http.Header {
		t.Helper()
		return MakeAvailableUserID(t, e, uuid.NewTestFromSalt(t, "ship_int_admin"))
	}

	shipViewFixture := func(t *testing.T) purchaseuc.ShipPurchaseView {
		t.Helper()
		shippedAt := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)
		return purchaseuc.ShipPurchaseView{
			ID:             uuid.NewTestFromSalt(t, "ship_int_id"),
			Code:           "ship-int-code",
			UserID:         uuid.NewTestFromSalt(t, "ship_int_user"),
			StatusID:       uuid.NewTestFromSalt(t, "ship_int_status"),
			StatusName:     "発送済み",
			SubtotalAmount: 160000,
			TaxAmount:      16000,
			ShippingFee:    500,
			TotalAmount:    176500,
			Details: []purchaseuc.PurchaseDetailView{
				{ProductID: uuid.NewTestFromSalt(t, "ship_int_product"), Quantity: 2, UnitPrice: decimaltestkit.MustParse(t, "800")},
			},
			OrderedAt: time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
			ShippedAt: &shippedAt,
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("PATCHがadminで200とPurchaseShipResponseを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().ShipPurchase(gomock.Any(), gomock.Any(), gomock.Any()).Return(shipViewFixture(t), nil)

			purchasesship.BindHandler(e, tf, uc)

			headers := availableShipAdmin(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			AssertJSONResponseType[shipgen.PurchaseShipResponse](t, actual)
		})

		t.Run("認証主体と購入IDがUsecaseへバインドされる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var capturedAuthn *auth.Authn
			var capturedID uuid.UUID
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().ShipPurchase(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, authn *auth.Authn, purchaseID uuid.UUID) (purchaseuc.ShipPurchaseView, error) {
					capturedAuthn = authn
					capturedID = purchaseID
					return shipViewFixture(t), nil
				},
			)

			purchasesship.BindHandler(e, tf, uc)

			headers := availableShipAdmin(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			require.NotNil(t, capturedAuthn)
			resolved, err := capturedAuthn.UserID()
			require.NoError(t, err)
			assert.Equal(t, uuid.NewTestFromSalt(t, "ship_int_admin"), resolved)
			assert.Equal(t, "0190b0d4-7b1a-7c2e-9f3a-1b2c3d4e5f60", capturedID.String())
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
			uc.EXPECT().ShipPurchase(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			purchasesship.BindHandler(e, tf, uc)

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
			uc.EXPECT().ShipPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.ShipPurchaseView{}, apperror.ErrPermissionDenied)

			purchasesship.BindHandler(e, tf, uc)

			headers := availableShipAdmin(t, e)
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
			uc.EXPECT().ShipPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.ShipPurchaseView{}, apperror.ErrNotFound)

			purchasesship.BindHandler(e, tf, uc)

			headers := availableShipAdmin(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusNotFound)
		})

		t.Run("二重発送で409を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().ShipPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.ShipPurchaseView{}, domainpurchase.ErrAlreadyShipped)

			purchasesship.BindHandler(e, tf, uc)

			headers := availableShipAdmin(t, e)
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
			uc.EXPECT().ShipPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.ShipPurchaseView{}, domainpurchase.ErrShipNotAllowed)

			purchasesship.BindHandler(e, tf, uc)

			headers := availableShipAdmin(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPatch, purchasePath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusConflict)
		})
	})
}
