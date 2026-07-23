package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	productshandler "go-boilerplate/internal/controller/handler/v1/products"
	"go-boilerplate/internal/controller/handler/v1/products/gen"
	"go-boilerplate/internal/observability"
	productuc "go-boilerplate/internal/usecase/product"
	mock_product "go-boilerplate/internal/usecase/product/mock"
	"go-boilerplate/internal/usecase/tools/paging"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestV1Products_Integration(t *testing.T) {
	t.Parallel()

	sampleView := func(t *testing.T) productuc.ProductView {
		t.Helper()
		return productuc.ProductView{
			ID:          uuid.NewTestFromSalt(t, "integration_product"),
			Name:        "商品",
			Description: ptr.To("説明"),
			Price:       decimaltestkit.MustParse(t, "19.99"),
			Quantity:    100,
			StatusID:    uuid.NewTestFromSalt(t, "integration_status"),
			CategoryID:  uuid.NewTestFromSalt(t, "integration_category"),
			PublishedAt: time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/products が未認証でも ProductListResponse を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			nextCursor := "next-opaque-cursor"
			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().ListProducts(gomock.Any(), gomock.Any()).Return(
				&productuc.ProductListView{Items: []productuc.ProductView{sampleView(t)}, NextCursor: &nextCursor}, nil,
			)

			productshandler.BindHandler(e, tf, mockUC)

			// security: [] の公開エンドポイントのため、Authorization ヘッダー無しでも 200 が返る。
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products", nil, nil)
			AssertJSONResponseType[gen.ProductListResponse](t, actual)
		})

		t.Run("GET /v1/products?first=5 の先頭ページが取得できる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().ListProducts(gomock.Any(), gomock.Any()).Return(
				&productuc.ProductListView{Items: []productuc.ProductView{sampleView(t)}, NextCursor: nil}, nil,
			)

			productshandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products?first=5", nil, nil)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
		})

		t.Run("after カーソル指定の継続ページが取得できる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().ListProducts(gomock.Any(), gomock.Any()).Return(
				&productuc.ProductListView{Items: []productuc.ProductView{}, NextCursor: nil}, nil,
			)

			productshandler.BindHandler(e, tf, mockUC)

			after := paging.EncodeCursor("2026-01-01T00:00:00Z", uuid.NewTestFromSalt(t, "integration_after").String())
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products?after="+after, nil, nil)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
		})

		t.Run("categoryId/statusId/keyword/sort クエリがハンドラのパラメータへバインドされる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			categoryID := uuid.NewTestFromSalt(t, "integration_filter_category")
			statusID := uuid.NewTestFromSalt(t, "integration_filter_status")

			var captured productuc.ListProductsParams
			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().ListProducts(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params productuc.ListProductsParams) (*productuc.ProductListView, error) {
					captured = params
					return &productuc.ProductListView{Items: []productuc.ProductView{}, NextCursor: nil}, nil
				},
			)

			productshandler.BindHandler(e, tf, mockUC)

			path := "/v1/products?categoryId=" + categoryID.String() +
				"&statusId=" + statusID.String() + "&keyword=%E5%95%86%E5%93%81&sort=publishedAt"
			actual := StartServer(t, e).DoJSON(http.MethodGet, path, nil, nil)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			assert.True(t, captured.Ascending)
			require.NotNil(t, captured.CategoryID)
			assert.Equal(t, categoryID, *captured.CategoryID)
			require.NotNil(t, captured.StatusID)
			assert.Equal(t, statusID, *captured.StatusID)
			require.NotNil(t, captured.Keyword)
			assert.Equal(t, "商品", *captured.Keyword)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/products が不正な after で 400 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			// NewCursor が失敗するため Usecase は呼ばれない。
			mockUC.EXPECT().ListProducts(gomock.Any(), gomock.Any()).Times(0)

			productshandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products?after=%21%21%21", nil, nil)
			AssertErrorResponse(t, actual, http.StatusBadRequest)
		})

		t.Run("GET /v1/products が ErrInternal で 500 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().ListProducts(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrInternal)

			productshandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products", nil, nil)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}
