package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	purchasesdetail "go-boilerplate/internal/controller/handler/v1/purchases/detail"
	detailgen "go-boilerplate/internal/controller/handler/v1/purchases/detail/gen"
	"go-boilerplate/internal/observability"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
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

func TestV1PurchasesDetail_Integration(t *testing.T) {
	t.Parallel()

	const purchasePath = "/v1/purchases/0190b0d4-7b1a-7c2e-9f3a-1b2c3d4e5f60"

	detailViewFixture := func(t *testing.T) purchaseuc.PurchaseGetDetailView {
		t.Helper()
		paidAt := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
		return purchaseuc.PurchaseGetDetailView{
			ID:             uuidtestkit.NewTestFromSalt(t, "detail_int_id"),
			Code:           "detail-int-code",
			UserID:         uuidtestkit.NewTestFromSalt(t, "detail_int_user"),
			StatusID:       uuidtestkit.NewTestFromSalt(t, "detail_int_status"),
			StatusName:     "支払い済み",
			SubtotalAmount: 160000,
			TaxAmount:      16000,
			ShippingFee:    500,
			TotalAmount:    176500,
			Details: []purchaseuc.PurchaseDetailItemView{
				{
					ProductID:   uuidtestkit.NewTestFromSalt(t, "detail_int_product"),
					ProductName: "ワイヤレスイヤホン",
					Quantity:    2,
					UnitPrice:   decimaltestkit.MustParse(t, "800"),
				},
			},
			OrderedAt:  time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
			PaidAt:     &paidAt,
			CanceledAt: nil,
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GETが認証済みで200と商品名/金額/status/paidAtを含む詳細を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchaseDetail(gomock.Any(), gomock.Any(), gomock.Any()).Return(detailViewFixture(t), nil)

			purchasesdetail.BindHandler(e, tf, uc)

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodGet, purchasePath, nil, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			var body detailgen.PurchaseGetDetailResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&body))
			assert.Equal(t, "detail-int-code", body.Code)
			assert.Equal(t, "支払い済み", body.Status.Name)
			assert.Equal(t, int64(160000), body.SubtotalAmount)
			assert.Equal(t, int64(176500), body.TotalAmount)
			require.Len(t, body.Details, 1)
			assert.Equal(t, "ワイヤレスイヤホン", body.Details[0].ProductName)
			assert.Equal(t, "800", body.Details[0].UnitPrice)
			assert.EqualValues(t, 2, body.Details[0].Quantity)
			require.NotNil(t, body.PaidAt)
			assert.Nil(t, body.CanceledAt)
		})

		t.Run("認証主体のuserIDと購入IDがUsecaseへバインドされる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var capturedID uuid.UUID
			var capturedUserID uuid.UUID
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchaseDetail(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, authn *authbd.Authn, id uuid.UUID) (purchaseuc.PurchaseGetDetailView, error) {
					capturedID = id
					userID, err := authn.UserID()
					require.NoError(t, err)
					capturedUserID = userID
					return detailViewFixture(t), nil
				},
			)

			purchasesdetail.BindHandler(e, tf, uc)

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodGet, purchasePath, nil, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			expectedUserID, err := uuid.Parse("b7f64798-7321-242b-e4ff-115f6a0b7810")
			require.NoError(t, err)
			assert.Equal(t, expectedUserID, capturedUserID)
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
			uc.EXPECT().GetPurchaseDetail(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			purchasesdetail.BindHandler(e, tf, uc)

			actual := StartServer(t, e).DoJSON(http.MethodGet, purchasePath, nil, nil)
			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("他人の購入・不存在で404を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchaseDetail(gomock.Any(), gomock.Any(), gomock.Any()).Return(purchaseuc.PurchaseGetDetailView{}, apperror.ErrNotFound)

			purchasesdetail.BindHandler(e, tf, uc)

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodGet, purchasePath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusNotFound)
		})
	})
}
