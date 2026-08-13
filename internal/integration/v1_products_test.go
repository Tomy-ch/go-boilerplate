package integration

import (
	"context"
	"encoding/json"
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
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestV1Products_Integration(t *testing.T) {
	t.Parallel()

	sampleView := func(t *testing.T) productuc.ProductView {
		t.Helper()
		return productuc.ProductView{
			ID:          uuidtestkit.NewTestFromSalt(t, "integration_product"),
			Name:        "商品",
			Description: ptr.To("説明"),
			Price:       decimaltestkit.MustParse(t, "19.99"),
			Quantity:    100,
			StatusID:    uuidtestkit.NewTestFromSalt(t, "integration_status"),
			CategoryID:  uuidtestkit.NewTestFromSalt(t, "integration_category"),
			PublishedAt: ptr.To(time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)),
			ImagePath:   ptr.To("products/integration_product.png"),
		}
	}

	productCreateBody := func(t *testing.T) *gen.PostProductsJSONRequestBody {
		t.Helper()
		return &gen.PostProductsJSONRequestBody{
			Name:        "商品",
			Description: ptr.To("<p>リッチテキスト説明</p>"),
			Price:       "19.99",
			Quantity:    100,
			CategoryId:  uuidtestkit.NewTestFromSalt(t, "integration_category").ToPrimitive(),
			StatusId:    uuidtestkit.NewTestFromSalt(t, "integration_status").ToPrimitive(),
			ImagePath:   ptr.To("products/integration_product.png"),
		}
	}

	availableAdmin := func(t *testing.T, e *echo.Echo) http.Header {
		t.Helper()
		return MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "integration_admin"))
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("POST /v1/products が admin で 201 を返し imagePath を往復する", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().CreateProduct(gomock.Any(), gomock.Any(), gomock.Any()).Return(sampleView(t), nil)

			productshandler.BindHandler(e, tf, mockUC)

			headers := availableAdmin(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPost, "/v1/products", productCreateBody(t), headers)
			require.Equal(t, http.StatusCreated, actual.StatusCode)

			var body gen.ProductResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&body))
			require.NotNil(t, body.ImagePath)
			assert.Equal(t, "products/integration_product.png", *body.ImagePath)
			require.NotNil(t, body.Description)
			assert.Equal(t, "説明", *body.Description)
		})

		t.Run("GET /v1/products が未認証でも ProductListResponse を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			nextCursor := "next-opaque-cursor"
			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().ListProducts(gomock.Any(), gomock.Any()).Return(
				productuc.ProductListView{Items: []productuc.ProductView{sampleView(t)}, NextCursor: &nextCursor}, nil,
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
				productuc.ProductListView{Items: []productuc.ProductView{sampleView(t)}, NextCursor: nil}, nil,
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
				productuc.ProductListView{Items: []productuc.ProductView{}, NextCursor: nil}, nil,
			)

			productshandler.BindHandler(e, tf, mockUC)

			after := paging.EncodeCursor("2026-01-01T00:00:00Z", uuidtestkit.NewTestFromSalt(t, "integration_after").String())
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products?after="+after, nil, nil)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
		})

		t.Run("検索・範囲・sortクエリがハンドラのパラメータへバインドされる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			categoryID := uuidtestkit.NewTestFromSalt(t, "integration_filter_category")
			statusID := uuidtestkit.NewTestFromSalt(t, "integration_filter_status")

			var captured productuc.ListProductsParams
			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().ListProducts(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params productuc.ListProductsParams) (productuc.ProductListView, error) {
					captured = params
					return productuc.ProductListView{Items: []productuc.ProductView{}, NextCursor: nil}, nil
				},
			)

			productshandler.BindHandler(e, tf, mockUC)

			path := "/v1/products?categoryId=" + categoryID.String() +
				"&statusId=" + statusID.String() + "&keyword=%E5%95%86%E5%93%81&minPrice=10.50&maxPrice=99.99" +
				"&minQuantity=2&maxQuantity=20&sort=publishedAt"
			actual := StartServer(t, e).DoJSON(http.MethodGet, path, nil, nil)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			assert.True(t, captured.Ascending)
			require.NotNil(t, captured.CategoryID)
			assert.Equal(t, categoryID, *captured.CategoryID)
			require.NotNil(t, captured.StatusID)
			assert.Equal(t, statusID, *captured.StatusID)
			require.NotNil(t, captured.Keyword)
			assert.Equal(t, "商品", *captured.Keyword)
			require.NotNil(t, captured.MinPrice)
			assert.Equal(t, "10.50", *captured.MinPrice)
			require.NotNil(t, captured.MaxPrice)
			assert.Equal(t, "99.99", *captured.MaxPrice)
			require.NotNil(t, captured.MinQuantity)
			assert.Equal(t, int32(2), *captured.MinQuantity)
			require.NotNil(t, captured.MaxQuantity)
			assert.Equal(t, int32(20), *captured.MaxQuantity)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("POST /v1/products が権限エラー(usecase)を403へ変換する", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().CreateProduct(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductView{}, apperror.ErrPermissionDenied)

			productshandler.BindHandler(e, tf, mockUC)

			headers := availableAdmin(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPost, "/v1/products", productCreateBody(t), headers)
			AssertErrorResponse(t, actual, http.StatusForbidden)
		})

		t.Run("POST /v1/products が負価格・負在庫などの検証違反で422を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().CreateProduct(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductView{}, apperror.ErrValidation)

			productshandler.BindHandler(e, tf, mockUC)

			headers := availableAdmin(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPost, "/v1/products", productCreateBody(t), headers)
			AssertErrorResponse(t, actual, http.StatusUnprocessableEntity)
		})

		t.Run("POST /v1/products が status/category 不在(整合性異常)で500を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().CreateProduct(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductView{}, apperror.ErrInternal)

			productshandler.BindHandler(e, tf, mockUC)

			headers := availableAdmin(t, e)
			actual := StartServer(t, e).DoJSON(http.MethodPost, "/v1/products", productCreateBody(t), headers)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})

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
			mockUC.EXPECT().ListProducts(gomock.Any(), gomock.Any()).Return(productuc.ProductListView{}, apperror.ErrInternal)

			productshandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products", nil, nil)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}
