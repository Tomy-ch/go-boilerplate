package integration

import (
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	productcategorieshandler "go-boilerplate/internal/controller/handler/v1/products/categories"
	"go-boilerplate/internal/controller/handler/v1/products/categories/gen"
	"go-boilerplate/internal/observability"
	categoryuc "go-boilerplate/internal/usecase/product/category"
	mock_category "go-boilerplate/internal/usecase/product/category/mock"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestV1ProductCategories_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/product-categories が未認証でも ProductCategoryResponse 一覧を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			id, err := uuid.New()
			require.NoError(t, err)

			mockUC := mock_category.NewMockUsecase(ctrl)
			mockUC.EXPECT().ListCategories(gomock.Any()).Return(
				categoryuc.CategoryDTOs{{ID: id, Code: 1, Name: "商品カテゴリ", SortKey: 1}}, nil,
			)

			productcategorieshandler.BindHandler(e, tf, mockUC)

			// security: [] の公開エンドポイントのため、Authorization ヘッダー無しでも 200 が返る。
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products/categories", nil, nil)
			AssertJSONResponseType[[]gen.ProductCategoryResponse](t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/products/categories が ErrInternal で 500 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_category.NewMockUsecase(ctrl)
			mockUC.EXPECT().ListCategories(gomock.Any()).Return(nil, apperror.ErrInternal)

			productcategorieshandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products/categories", nil, nil)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}
