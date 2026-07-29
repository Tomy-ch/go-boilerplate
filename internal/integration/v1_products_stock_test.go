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
	domainproduct "go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	productuc "go-boilerplate/internal/usecase/product"
	mock_product "go-boilerplate/internal/usecase/product/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	productStockExistingPath = "/v1/products/b1d4e0f2-3c5a-4b6d-8e7f-1a2b3c4d5e6f/stock"
	productStockMissingPath  = "/v1/products/00000000-0000-0000-0000-000000000000/stock"
)

func TestV1ProductsStock_Integration(t *testing.T) {
	t.Parallel()

	// stockedView は、在庫更新後の商品を表すユースケース出力です。quantity / version は更新後の値です。
	stockedView := func(t *testing.T, quantity, version int) productuc.ProductView {
		t.Helper()
		return productuc.ProductView{
			ID:                    uuid.NewTestFromSalt(t, "integration_product_stock"),
			Name:                  "商品",
			Description:           ptr.To("説明"),
			Price:                 decimaltestkit.MustParse(t, "19.99"),
			Quantity:              quantity,
			StockWarningThreshold: ptr.To(10),
			StatusID:              uuid.NewTestFromSalt(t, "integration_stock_status"),
			StatusName:            "在庫あり",
			CategoryID:            uuid.NewTestFromSalt(t, "integration_stock_category"),
			CategoryName:          "電子機器",
			PublishedAt:           ptr.To(time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)),
			ImagePath:             ptr.To("products/integration_stock.png"),
			Version:               version,
		}
	}

	availableAdmin := func(t *testing.T, e *echo.Echo) http.Header {
		t.Helper()
		return MakeAvailableUserID(t, e, uuid.NewTestFromSalt(t, "integration_stock_admin"))
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("PATCH /v1/products/{productId}/stock が admin の補充で更新後の在庫と version を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var captured productuc.UpdateProductStockParams
			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().UpdateProductStock(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(
					_ context.Context, authn *auth.Authn, _ uuid.UUID, params productuc.UpdateProductStockParams,
				) (productuc.ProductView, error) {
					require.NotNil(t, authn)
					captured = params
					return stockedView(t, 150, 2), nil
				},
			)

			productsdetail.BindHandler(e, tf, mockUC)

			headers := availableAdmin(t, e)
			body := &productsdetailgen.PatchProductsStockJSONRequestBody{Delta: 50}

			actual := StartServer(t, e).DoJSON(http.MethodPatch, productStockExistingPath, body, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)
			assert.Equal(t, 50, captured.Delta)

			var res productsdetailgen.ProductResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&res))
			assert.Equal(t, int32(150), res.Quantity)
			assert.Equal(t, int32(2), res.Version)
		})

		t.Run("PATCH /v1/products/{productId}/stock は負の delta を符号のままユースケースへ渡す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var captured productuc.UpdateProductStockParams
			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().UpdateProductStock(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(
					_ context.Context, _ *auth.Authn, _ uuid.UUID, params productuc.UpdateProductStockParams,
				) (productuc.ProductView, error) {
					captured = params
					return stockedView(t, 90, 2), nil
				},
			)

			productsdetail.BindHandler(e, tf, mockUC)

			headers := availableAdmin(t, e)
			body := &productsdetailgen.PatchProductsStockJSONRequestBody{Delta: -10}

			actual := StartServer(t, e).DoJSON(http.MethodPatch, productStockExistingPath, body, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)
			assert.Equal(t, -10, captured.Delta)

			var res productsdetailgen.ProductResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&res))
			assert.Equal(t, int32(90), res.Quantity)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未認証は 401 を返しユースケースへ到達しない", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().UpdateProductStock(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			productsdetail.BindHandler(e, tf, mockUC)

			body := &productsdetailgen.PatchProductsStockJSONRequestBody{Delta: 10}
			actual := StartServer(t, e).DoJSON(http.MethodPatch, productStockExistingPath, body, nil)
			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("非 admin の権限エラーは 403 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().UpdateProductStock(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductView{}, apperror.ErrPermissionDenied)

			productsdetail.BindHandler(e, tf, mockUC)

			headers := MakeAvailableUserID(t, e, uuid.NewTestFromSalt(t, "integration_stock_member"))
			body := &productsdetailgen.PatchProductsStockJSONRequestBody{Delta: 10}
			actual := StartServer(t, e).DoJSON(http.MethodPatch, productStockExistingPath, body, headers)
			AssertErrorResponse(t, actual, http.StatusForbidden)
		})

		t.Run("未存在の productId は 404 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().UpdateProductStock(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductView{}, apperror.ErrNotFound)

			productsdetail.BindHandler(e, tf, mockUC)

			headers := availableAdmin(t, e)
			body := &productsdetailgen.PatchProductsStockJSONRequestBody{Delta: 10}
			actual := StartServer(t, e).DoJSON(http.MethodPatch, productStockMissingPath, body, headers)
			AssertErrorResponse(t, actual, http.StatusNotFound)
		})

		t.Run("在庫が負になる delta の不変条件違反は 422 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().UpdateProductStock(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductView{}, apperror.ErrValidation)

			productsdetail.BindHandler(e, tf, mockUC)

			headers := availableAdmin(t, e)
			body := &productsdetailgen.PatchProductsStockJSONRequestBody{Delta: -1000}
			actual := StartServer(t, e).DoJSON(http.MethodPatch, productStockExistingPath, body, headers)
			AssertErrorResponse(t, actual, http.StatusUnprocessableEntity)
		})

		t.Run("取得後に他者が更新していた場合は 409 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().UpdateProductStock(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductView{}, domainproduct.ErrVersionConflict)

			productsdetail.BindHandler(e, tf, mockUC)

			headers := availableAdmin(t, e)
			body := &productsdetailgen.PatchProductsStockJSONRequestBody{Delta: 10}
			actual := StartServer(t, e).DoJSON(http.MethodPatch, productStockExistingPath, body, headers)
			AssertErrorResponse(t, actual, http.StatusConflict)
		})

		t.Run("行競合が解消できなかった場合は 503 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().UpdateProductStock(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductView{}, apperror.ErrUnavailable)

			productsdetail.BindHandler(e, tf, mockUC)

			headers := availableAdmin(t, e)
			body := &productsdetailgen.PatchProductsStockJSONRequestBody{Delta: 10}
			actual := StartServer(t, e).DoJSON(http.MethodPatch, productStockExistingPath, body, headers)
			AssertErrorResponse(t, actual, http.StatusServiceUnavailable)
		})
	})
}
