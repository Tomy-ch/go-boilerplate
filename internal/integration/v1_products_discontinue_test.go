package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	productsdetail "go-boilerplate/internal/controller/handler/v1/products/detail"
	productsdetailgen "go-boilerplate/internal/controller/handler/v1/products/detail/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/idempotency"
	productuc "go-boilerplate/internal/usecase/product"
	mock_product "go-boilerplate/internal/usecase/product/mock"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	productDiscontinuePath       = "/v1/products/b1d4e0f2-3c5a-4b6d-8e7f-1a2b3c4d5e6f/discontinue"
	productDiscontinueImpactPath = "/v1/products/b1d4e0f2-3c5a-4b6d-8e7f-1a2b3c4d5e6f/discontinue-impact"
)

func TestV1ProductsDiscontinue_Integration(t *testing.T) {
	t.Parallel()

	discontinuedAt := time.Date(2026, time.September, 5, 0, 0, 0, 0, time.UTC)

	availableAdmin := func(t *testing.T, e *echo.Echo) http.Header {
		t.Helper()
		return MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "integration_discontinue_admin"))
	}

	newBody := func() *productsdetailgen.PostProductsDiscontinueJSONRequestBody {
		return &productsdetailgen.PostProductsDiscontinueJSONRequestBody{
			CouponDiscountRate: "0.10",
			CouponValidityDays: 30,
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("POST /v1/products/{productId}/discontinue が admin で件数つきの 200 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var captured productuc.DiscontinueProductParams
			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().DiscontinueProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(
					_ context.Context, authn *auth.Authn, _ uuid.UUID, params productuc.DiscontinueProductParams,
				) (productuc.DiscontinueProductView, error) {
					require.NotNil(t, authn)
					captured = params

					return productuc.DiscontinueProductView{
						DiscontinuedAt:    discontinuedAt,
						AffectedCartCount: 12,
						AffectedUserCount: 9,
						IssuedCouponCount: 9,
					}, nil
				},
			)

			productsdetail.BindHandler(e, tf, mockUC, idempotency.Deps{})

			actual := StartServer(t, e).DoJSON(http.MethodPost, productDiscontinuePath, newBody(), availableAdmin(t, e))
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			var body productsdetailgen.ProductDiscontinueResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&body))

			assert.Equal(t, int64(12), body.AffectedCartCount)
			assert.Equal(t, int64(9), body.AffectedUserCount)
			assert.Equal(t, int64(9), body.IssuedCouponCount)
			assert.Equal(t, "0.10", captured.CouponDiscountRate.String())
			assert.Equal(t, 30*24*time.Hour, captured.CouponValidity)
		})

		t.Run("既に廃番の商品への再実行は件数がすべて0の200を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(gomock.NewController(t))
			mockUC.EXPECT().DiscontinueProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.DiscontinueProductView{DiscontinuedAt: discontinuedAt}, nil)

			productsdetail.BindHandler(e, tf, mockUC, idempotency.Deps{})

			actual := StartServer(t, e).DoJSON(http.MethodPost, productDiscontinuePath, newBody(), availableAdmin(t, e))
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			var body productsdetailgen.ProductDiscontinueResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&body))

			assert.Zero(t, body.AffectedCartCount)
			assert.Zero(t, body.AffectedUserCount)
			assert.Zero(t, body.IssuedCouponCount)
			assert.Equal(t, discontinuedAt, body.DiscontinuedAt)
		})

		t.Run("GET /v1/products/{productId}/discontinue-impact が admin で見積もりの3件数を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(gomock.NewController(t))
			mockUC.EXPECT().GetDiscontinueImpact(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.DiscontinueImpactView{
					AffectedCartCount:       12,
					AffectedUserCount:       9,
					InProgressPurchaseCount: 2,
				}, nil)

			productsdetail.BindHandler(e, tf, mockUC, idempotency.Deps{})

			actual := StartServer(t, e).DoJSON(http.MethodGet, productDiscontinueImpactPath, nil, availableAdmin(t, e))
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			var body productsdetailgen.ProductDiscontinueImpactResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&body))

			assert.Equal(t, int64(12), body.AffectedCartCount)
			assert.Equal(t, int64(9), body.AffectedUserCount)
			assert.Equal(t, int64(2), body.InProgressPurchaseCount)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トークン無しの廃番要求は401で拒まれユースケースへ届かない", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			mockUC := mock_product.NewMockUsecase(gomock.NewController(t))
			mockUC.EXPECT().DiscontinueProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			productsdetail.BindHandler(e, observability.NewNoopTracerFactory(t), mockUC, idempotency.Deps{})

			actual := StartServer(t, e).DoJSON(http.MethodPost, productDiscontinuePath, newBody(), nil)
			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("トークン無しの事前確認は401で拒まれユースケースへ届かない", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			mockUC := mock_product.NewMockUsecase(gomock.NewController(t))
			mockUC.EXPECT().GetDiscontinueImpact(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			productsdetail.BindHandler(e, observability.NewNoopTracerFactory(t), mockUC, idempotency.Deps{})

			actual := StartServer(t, e).DoJSON(http.MethodGet, productDiscontinueImpactPath, nil, nil)
			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("非 admin の権限エラーは403を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(gomock.NewController(t))
			mockUC.EXPECT().DiscontinueProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.DiscontinueProductView{}, apperror.ErrPermissionDenied)

			productsdetail.BindHandler(e, tf, mockUC, idempotency.Deps{})

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "integration_discontinue_member"))
			actual := StartServer(t, e).DoJSON(http.MethodPost, productDiscontinuePath, newBody(), headers)
			AssertErrorResponse(t, actual, http.StatusForbidden)
		})

		t.Run("進行中の購入が残っている場合は409を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(gomock.NewController(t))
			mockUC.EXPECT().DiscontinueProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.DiscontinueProductView{}, apperror.ErrConflict)

			productsdetail.BindHandler(e, tf, mockUC, idempotency.Deps{})

			actual := StartServer(t, e).DoJSON(http.MethodPost, productDiscontinuePath, newBody(), availableAdmin(t, e))
			AssertErrorResponse(t, actual, http.StatusConflict)
		})

		t.Run("値引き率が宣言の範囲外の本文は spec 検証で400になりユースケースへ届かない", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			mockUC := mock_product.NewMockUsecase(gomock.NewController(t))
			mockUC.EXPECT().DiscontinueProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			productsdetail.BindHandler(e, observability.NewNoopTracerFactory(t), mockUC, idempotency.Deps{})
			headers := availableAdmin(t, e)
			useOpenAPIValidation(t, e)

			body := map[string]any{"couponDiscountRate": "0.10", "couponValidityDays": 0}
			actual := StartServer(t, e).DoJSON(http.MethodPost, productDiscontinuePath, body, headers)

			AssertErrorResponse(t, actual, http.StatusBadRequest)
		})

		t.Run("宣言に無いフィールドを含む本文は400で弾かれる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			mockUC := mock_product.NewMockUsecase(gomock.NewController(t))
			mockUC.EXPECT().DiscontinueProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			productsdetail.BindHandler(e, observability.NewNoopTracerFactory(t), mockUC, idempotency.Deps{})
			headers := availableAdmin(t, e)
			useOpenAPIValidation(t, e)

			body := map[string]any{"couponDiscountRate": "0.10", "couponValidityDays": 30, "productId": "spoofed"}
			actual := StartServer(t, e).DoJSON(http.MethodPost, productDiscontinuePath, body, headers)

			AssertErrorResponse(t, actual, http.StatusBadRequest)
		})
	})
}
