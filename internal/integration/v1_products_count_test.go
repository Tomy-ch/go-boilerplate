package integration

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	productscounthandler "go-boilerplate/internal/controller/handler/v1/products/count"
	"go-boilerplate/internal/controller/handler/v1/products/count/gen"
	"go-boilerplate/internal/observability"
	productuc "go-boilerplate/internal/usecase/product"
	mock_product "go-boilerplate/internal/usecase/product/mock"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestV1ProductsCount_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/products/count が未認証で検索条件を渡し件数を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			mockUC := mock_product.NewMockUsecase(gomock.NewController(t))
			mockUC.EXPECT().CountProducts(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params productuc.SearchFilter) (productuc.ProductCountView, error) {
					assert.Equal(t, "10.50", *params.MinPrice)
					assert.Equal(t, int32(2), *params.MinQuantity)
					return productuc.ProductCountView{Count: 3}, nil
				})
			productscounthandler.BindHandler(e, observability.NewNoopTracerFactory(t), mockUC)

			actual := StartServer(t, e).DoJSON(
				http.MethodGet, "/v1/products/count?minPrice=10.50&minQuantity=2", nil, nil,
			)

			AssertJSONResponseType[gen.ProductCountResponse](t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("OpenAPIバリデーションが負の在庫数を400で弾く", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			mockUC := mock_product.NewMockUsecase(gomock.NewController(t))
			mockUC.EXPECT().CountProducts(gomock.Any(), gomock.Any()).Times(0)
			productscounthandler.BindHandler(e, observability.NewNoopTracerFactory(t), mockUC)
			useOpenAPIValidation(t, e)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products/count?minQuantity=-1", nil, nil)

			AssertErrorResponse(t, actual, http.StatusBadRequest)
		})

		t.Run("Usecaseエラーを500へ変換する", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			mockUC := mock_product.NewMockUsecase(gomock.NewController(t))
			mockUC.EXPECT().CountProducts(gomock.Any(), gomock.Any()).Return(
				productuc.ProductCountView{}, apperror.ErrInternal,
			)
			productscounthandler.BindHandler(e, observability.NewNoopTracerFactory(t), mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products/count", nil, nil)

			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}
