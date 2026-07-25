package integration

import (
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	responsegen "go-boilerplate/internal/controller/error/response/gen"
	productsdetail "go-boilerplate/internal/controller/handler/v1/products/detail"
	productsdetailgen "go-boilerplate/internal/controller/handler/v1/products/detail/gen"
	"go-boilerplate/internal/observability"
	productuc "go-boilerplate/internal/usecase/product"
	mock_product "go-boilerplate/internal/usecase/product/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const (
	productDetailExistingPath  = "/v1/products/b1d4e0f2-3c5a-4b6d-8e7f-1a2b3c4d5e6f"
	productDetailMissingPath   = "/v1/products/00000000-0000-0000-0000-000000000000"
	productDetailUnpublishedID = "/v1/products/d42d659d-21f9-4b5c-b05d-3130de157a94"
)

func TestV1ProductsDetail_Integration(t *testing.T) {
	t.Parallel()

	sampleView := func(t *testing.T) productuc.ProductView {
		t.Helper()
		return productuc.ProductView{
			ID:          uuid.NewTestFromSalt(t, "integration_product_detail"),
			Name:        "商品",
			Description: ptr.To("説明"),
			Price:       decimaltestkit.MustParse(t, "19.99"),
			Quantity:    100,
			StatusID:    uuid.NewTestFromSalt(t, "integration_detail_status"),
			CategoryID:  uuid.NewTestFromSalt(t, "integration_detail_category"),
			PublishedAt: ptr.To(time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)),
			ImagePath:   ptr.To("products/integration_detail.png"),
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/products/{productId} が未認証でも ProductResponse を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().GetProduct(gomock.Any(), gomock.Any()).Return(sampleView(t), nil)

			productsdetail.BindHandler(e, tf, mockUC)

			// security: [] の公開エンドポイントのため、Authorization ヘッダー無しでも 200 が返る。
			actual := StartServer(t, e).DoJSON(http.MethodGet, productDetailExistingPath, nil, nil)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[productsdetailgen.ProductResponse](t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未存在の productId は 404 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().GetProduct(gomock.Any(), gomock.Any()).Return(productuc.ProductView{}, apperror.ErrNotFound)

			productsdetail.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, productDetailMissingPath, nil, nil)
			AssertErrorResponse(t, actual, http.StatusNotFound)
		})

		t.Run("未存在と非公開の 404 はレスポンス body が区別できない（存在秘匿）", func(t *testing.T) {
			t.Parallel()

			// ハンドラ経路が未存在・非公開の NotFound を同一の Code/Message に落とすことを検証する
			// （未ログイン利用者へ商品の存在を秘匿する）。非公開が SQL 述語で未存在と同一の取得失敗に
			// 落ちること自体は infra 層の DB テスト（Test_repository_FindPublishedByID）が担保する。
			// requestId は各リクエストで変わりうるため body 全体ではなく Code/Message のみ比較する。
			missingBody := doNotFoundProductDetail(t, productDetailMissingPath)
			unpublishedBody := doNotFoundProductDetail(t, productDetailUnpublishedID)

			assert.Equal(t, missingBody.Code, unpublishedBody.Code)
			assert.Equal(t, missingBody.Message, unpublishedBody.Message)
		})

		t.Run("Usecase が ErrInternal を返すと 500 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().GetProduct(gomock.Any(), gomock.Any()).Return(productuc.ProductView{}, apperror.ErrInternal)

			productsdetail.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, productDetailMissingPath, nil, nil)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}

// doNotFoundProductDetail は、GetProduct が NotFound を返す状況で指定パスへ GET し、404 のエラーボディを返します。
func doNotFoundProductDetail(t *testing.T, path string) responsegen.ErrorResponseWithDetails {
	t.Helper()

	e := echo.New()
	UseAppErrorHandler(t, e)
	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)

	mockUC := mock_product.NewMockUsecase(ctrl)
	mockUC.EXPECT().GetProduct(gomock.Any(), gomock.Any()).Return(productuc.ProductView{}, apperror.ErrNotFound)

	productsdetail.BindHandler(e, tf, mockUC)

	actual := StartServer(t, e).DoJSON(http.MethodGet, path, nil, nil)
	return AssertErrorResponseBody(t, actual, http.StatusNotFound)
}
