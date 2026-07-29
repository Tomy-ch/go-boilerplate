package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	v1purchases "go-boilerplate/internal/controller/handler/v1/purchases"
	"go-boilerplate/internal/controller/handler/v1/purchases/gen"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/observability"
	clocktest "go-boilerplate/internal/usecase/boundary/clock/testkit"
	mock_idempotency "go-boilerplate/internal/usecase/boundary/idempotency/mock"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	"go-boilerplate/internal/usecase/idempotency"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	mock_purchaseuc "go-boilerplate/internal/usecase/purchase/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func purchaseViewFixture(t *testing.T) purchaseuc.PurchaseView {
	t.Helper()
	return purchaseuc.PurchaseView{
		ID:             uuid.NewTestFromSalt(t, "int_id"),
		Code:           "int-code",
		UserID:         uuid.NewTestFromSalt(t, "int_user"),
		StatusID:       uuid.NewTestFromSalt(t, "int_status"),
		SubtotalAmount: 160000,
		TaxAmount:      16000,
		ShippingFee:    500,
		TotalAmount:    176500,
		Details: []purchaseuc.PurchaseDetailView{
			{ProductID: uuid.NewTestFromSalt(t, "int_prod"), Quantity: 2, UnitPrice: decimaltestkit.MustParse(t, "800")},
		},
		OrderedAt: time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
	}
}

func purchaseRequestBody(t *testing.T) *gen.PostPurchasesJSONRequestBody {
	t.Helper()
	return &gen.PostPurchasesJSONRequestBody{
		Details: []gen.PurchaseDetailInput{
			{ProductId: uuid.NewTestFromSalt(t, "int_prod").ToPrimitive(), Quantity: 2},
		},
	}
}

func availablePurchaseUser(t *testing.T, e *echo.Echo) http.Header {
	t.Helper()
	id, err := uuid.Parse("b7f64798-7321-242b-e4ff-115f6a0b7810")
	require.NoError(t, err)
	return MakeAvailableUserID(t, e, id)
}

func TestV1Purchases_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("POST /v1/purchasesが購入を作成し201を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(purchaseViewFixture(t), nil)

			v1purchases.BindHandler(e, tf, uc, idempotency.Deps{})

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPost, "/v1/purchases", purchaseRequestBody(t), headers)
			require.Equal(t, http.StatusCreated, actual.StatusCode)

			var body gen.PurchaseResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&body))
			assert.Equal(t, "int-code", body.Code)
			require.Len(t, body.Details, 1)
		})

		t.Run("Idempotency-Key付きでclaim→completeし201を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(purchaseViewFixture(t), nil)

			store := mock_idempotency.NewMockStore(ctrl)
			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(true, nil)
			store.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(nil)
			txm := mock_tx.NewMockManager(ctrl)
			txm.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
			clk := clocktest.NewMockClock(t, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))

			v1purchases.BindHandler(e, tf, uc, idempotency.Deps{Txm: txm, Store: store, Clock: clk})

			headers := availablePurchaseUser(t, e)
			headers.Set("Idempotency-Key", "integration-key-1")
			actual := StartServer(t, e).DoJSON(http.MethodPost, "/v1/purchases", purchaseRequestBody(t), headers)
			require.Equal(t, http.StatusCreated, actual.StatusCode)

			var body gen.PurchaseResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&body))
			assert.Equal(t, "int-code", body.Code)
		})

		t.Run("displayCurrency指定時はreferenceAmountを含める", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			view := purchaseViewFixture(t)
			view.ReferenceAmount = &purchaseuc.ReferenceAmountView{
				Currency: "JPY",
				Amount:   26475,
				Rate:     decimaltestkit.MustParse(t, "150.5"),
				RateDate: "2026-07-21",
			}
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(view, nil)

			v1purchases.BindHandler(e, tf, uc, idempotency.Deps{})

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPost, "/v1/purchases?displayCurrency=JPY", purchaseRequestBody(t), headers)
			require.Equal(t, http.StatusCreated, actual.StatusCode)

			var body gen.PurchaseResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&body))
			require.NotNil(t, body.ReferenceAmount)
			assert.Equal(t, int64(26475), body.ReferenceAmount.Amount)
		})

		t.Run("exchange障害時はreferenceAmountがnullでも購入は成立する", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			// degrade: usecase が ReferenceAmount=nil のビューを返す
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(purchaseViewFixture(t), nil)

			v1purchases.BindHandler(e, tf, uc, idempotency.Deps{})

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPost, "/v1/purchases?displayCurrency=JPY", purchaseRequestBody(t), headers)
			require.Equal(t, http.StatusCreated, actual.StatusCode)

			var body gen.PurchaseResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&body))
			assert.Nil(t, body.ReferenceAmount)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("売り越し（在庫不足）は409を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(purchaseuc.PurchaseView{}, domainpurchase.ErrInsufficientStock)

			v1purchases.BindHandler(e, tf, uc, idempotency.Deps{})

			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPost, "/v1/purchases", purchaseRequestBody(t), headers)
			AssertErrorResponse(t, actual, http.StatusConflict)
		})

		t.Run("明細が空配列の場合は422を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(purchaseuc.PurchaseView{}, domainpurchase.ErrEmptyDetails)

			v1purchases.BindHandler(e, tf, uc, idempotency.Deps{})

			headers := availablePurchaseUser(t, e)
			actual := StartServer(
				t,
				e,
			).DoJSON(http.MethodPost, "/v1/purchases", &gen.PostPurchasesJSONRequestBody{Details: []gen.PurchaseDetailInput{}}, headers)
			AssertErrorResponse(t, actual, http.StatusUnprocessableEntity)
		})

		t.Run("同一productIDが重複する場合は422を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(purchaseuc.PurchaseView{}, domainpurchase.ErrDuplicateProductID)

			v1purchases.BindHandler(e, tf, uc, idempotency.Deps{})

			dupID := uuid.NewTestFromSalt(t, "dup_prod").ToPrimitive()
			body := &gen.PostPurchasesJSONRequestBody{Details: []gen.PurchaseDetailInput{
				{ProductId: dupID, Quantity: 1},
				{ProductId: dupID, Quantity: 2},
			}}
			headers := availablePurchaseUser(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPost, "/v1/purchases", body, headers)
			AssertErrorResponse(t, actual, http.StatusUnprocessableEntity)
		})
	})
}
