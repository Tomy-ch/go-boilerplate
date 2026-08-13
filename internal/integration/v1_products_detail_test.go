package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	responsegen "go-boilerplate/internal/controller/error/response/gen"
	productsdetail "go-boilerplate/internal/controller/handler/v1/products/detail"
	productsdetailgen "go-boilerplate/internal/controller/handler/v1/products/detail/gen"
	domainproduct "go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	productuc "go-boilerplate/internal/usecase/product"
	mock_product "go-boilerplate/internal/usecase/product/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/patch"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			ID:          uuidtestkit.NewTestFromSalt(t, "integration_product_detail"),
			Name:        "商品",
			Description: ptr.To("説明"),
			Price:       decimaltestkit.MustParse(t, "19.99"),
			Quantity:    100,
			StatusID:    uuidtestkit.NewTestFromSalt(t, "integration_detail_status"),
			CategoryID:  uuidtestkit.NewTestFromSalt(t, "integration_detail_category"),
			PublishedAt: ptr.To(time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)),
			Images: []productuc.ProductImageItemView{
				{Path: "products/integration_detail.png", SortKey: 1},
			},
		}
	}

	updatedView := func(t *testing.T) productuc.ProductView {
		t.Helper()
		view := sampleView(t)
		view.Name = "更新後の商品"
		view.Price = decimaltestkit.MustParse(t, "29.99")
		view.Quantity = 50
		view.Version = 2
		return view
	}

	availableAdmin := func(t *testing.T, e *echo.Echo) http.Header {
		t.Helper()
		return MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "integration_detail_admin"))
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

			actual := StartServer(t, e).DoJSON(http.MethodGet, productDetailExistingPath, nil, nil)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[productsdetailgen.ProductResponse](t, actual)
		})

		t.Run("PATCH /v1/products/{productId} が admin で更新後の値と 1 つ進んだ version を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var captured productuc.UpdateProductParams
			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().UpdateProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *auth.Authn, _ uuid.UUID, params productuc.UpdateProductParams) (productuc.ProductView, error) {
					captured = params
					return updatedView(t), nil
				},
			)

			productsdetail.BindHandler(e, tf, mockUC)

			headers := availableAdmin(t, e)
			body := &productsdetailgen.PatchProductsDetailJSONRequestBody{
				Version:  1,
				Name:     ptr.To("更新後の商品"),
				Price:    ptr.To("29.99"),
				Quantity: ptr.To(int32(50)),
			}

			actual := StartServer(t, e).DoJSON(http.MethodPatch, productDetailExistingPath, body, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			assert.Equal(t, 1, captured.Version)

			var res productsdetailgen.ProductResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&res))
			assert.Equal(t, "更新後の商品", res.Name)
			assert.Equal(t, "29.99", res.Price)
			assert.Equal(t, int32(50), res.Quantity)
			assert.Equal(t, int32(2), res.Version)
		})

		t.Run("PATCH /v1/products/{productId} の未送信フィールドは未指定として Usecase へ渡る", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var captured productuc.UpdateProductParams
			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().UpdateProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *auth.Authn, _ uuid.UUID, params productuc.UpdateProductParams) (productuc.ProductView, error) {
					captured = params
					return updatedView(t), nil
				},
			)

			productsdetail.BindHandler(e, tf, mockUC)

			headers := availableAdmin(t, e)
			body := &productsdetailgen.PatchProductsDetailJSONRequestBody{
				Version: 1,
				Name:    ptr.To("名称のみ更新"),
			}

			actual := StartServer(t, e).DoJSON(http.MethodPatch, productDetailExistingPath, body, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			require.NotNil(t, captured.Name)
			assert.Equal(t, "名称のみ更新", *captured.Name)
			assert.Nil(t, captured.Price)
			assert.Nil(t, captured.Quantity)
			assert.Nil(t, captured.CategoryID)
			assert.Nil(t, captured.StatusID)
			assert.Equal(t, patch.Unspecified[string](), captured.Description)
			assert.Equal(t, patch.Unspecified[int](), captured.StockWarningThreshold)
			assert.Equal(t, patch.Unspecified[time.Time](), captured.PublishedAt)
			assert.Equal(t, patch.Unspecified[[]productuc.ProductImageParams](), captured.Images)
		})

		t.Run("PATCH /v1/products/{productId} の null 指定がクリアとして渡り null で返る", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			cleared := updatedView(t)
			cleared.Description = nil
			cleared.StockWarningThreshold = nil
			cleared.PublishedAt = nil
			cleared.Images = nil

			var captured productuc.UpdateProductParams
			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().UpdateProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *auth.Authn, _ uuid.UUID, params productuc.UpdateProductParams) (productuc.ProductView, error) {
					captured = params
					return cleared, nil
				},
			)

			productsdetail.BindHandler(e, tf, mockUC)

			headers := availableAdmin(t, e)
			body := &productsdetailgen.PatchProductsDetailJSONRequestBody{
				Version:               1,
				Description:           nullable.NewNullNullable[string](),
				StockWarningThreshold: nullable.NewNullNullable[int32](),
				PublishedAt:           nullable.NewNullNullable[time.Time](),
				Images:                nullable.NewNullNullable[[]productsdetailgen.ProductImageInput](),
			}

			actual := StartServer(t, e).DoJSON(http.MethodPatch, productDetailExistingPath, body, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			assert.Equal(t, patch.Null[string](), captured.Description)
			assert.Equal(t, patch.Null[int](), captured.StockWarningThreshold)
			assert.Equal(t, patch.Null[time.Time](), captured.PublishedAt)
			assert.Equal(t, patch.Null[[]productuc.ProductImageParams](), captured.Images)

			var res productsdetailgen.ProductResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&res))
			assert.Nil(t, res.Description)
			assert.Nil(t, res.StockWarningThreshold)
			assert.Nil(t, res.PublishedAt)
			assert.Empty(t, res.Images)
		})

		t.Run("PATCH /v1/products/{productId} は未公開商品へ publishedAt を指定して公開状態にできる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			publishedAt := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)
			published := updatedView(t)
			published.PublishedAt = ptr.To(publishedAt)

			var captured productuc.UpdateProductParams
			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().UpdateProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *auth.Authn, _ uuid.UUID, params productuc.UpdateProductParams) (productuc.ProductView, error) {
					captured = params
					return published, nil
				},
			)

			productsdetail.BindHandler(e, tf, mockUC)

			headers := availableAdmin(t, e)
			body := &productsdetailgen.PatchProductsDetailJSONRequestBody{
				Version:     1,
				PublishedAt: nullable.NewNullableWithValue(publishedAt),
			}

			actual := StartServer(t, e).DoJSON(http.MethodPatch, productDetailUnpublishedID, body, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			capturedPublishedAt := captured.PublishedAt.Resolve(nil)
			require.NotNil(t, capturedPublishedAt)
			assert.True(t, publishedAt.Equal(*capturedPublishedAt))

			var res productsdetailgen.ProductResponse
			require.NoError(t, json.NewDecoder(actual.Body).Decode(&res))
			require.NotNil(t, res.PublishedAt)
			assert.True(t, publishedAt.Equal(*res.PublishedAt))
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

		t.Run("PATCH /v1/products/{productId} が未認証で 401 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().UpdateProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			productsdetail.BindHandler(e, tf, mockUC)

			body := &productsdetailgen.PatchProductsDetailJSONRequestBody{Version: 1, Name: ptr.To("更新後の商品")}
			actual := StartServer(t, e).DoJSON(http.MethodPatch, productDetailExistingPath, body, nil)
			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("PATCH /v1/products/{productId} が非 admin の権限エラーで 403 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().UpdateProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductView{}, apperror.ErrPermissionDenied)

			productsdetail.BindHandler(e, tf, mockUC)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "integration_detail_member"))
			body := &productsdetailgen.PatchProductsDetailJSONRequestBody{Version: 1, Name: ptr.To("更新後の商品")}
			actual := StartServer(t, e).DoJSON(http.MethodPatch, productDetailExistingPath, body, headers)
			AssertErrorResponse(t, actual, http.StatusForbidden)
		})

		t.Run("PATCH /v1/products/{productId} が未存在の productId で 404 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().UpdateProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductView{}, apperror.ErrNotFound)

			productsdetail.BindHandler(e, tf, mockUC)

			headers := availableAdmin(t, e)
			body := &productsdetailgen.PatchProductsDetailJSONRequestBody{Version: 1, Name: ptr.To("更新後の商品")}
			actual := StartServer(t, e).DoJSON(http.MethodPatch, productDetailMissingPath, body, headers)
			AssertErrorResponse(t, actual, http.StatusNotFound)
		})

		t.Run("PATCH /v1/products/{productId} が負価格・負在庫の検証違反で 422 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			mockUC.EXPECT().UpdateProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductView{}, apperror.ErrValidation)

			productsdetail.BindHandler(e, tf, mockUC)

			headers := availableAdmin(t, e)
			body := &productsdetailgen.PatchProductsDetailJSONRequestBody{
				Version:  1,
				Price:    ptr.To("-1.00"),
				Quantity: ptr.To(int32(-1)),
			}
			actual := StartServer(t, e).DoJSON(http.MethodPatch, productDetailExistingPath, body, headers)
			AssertErrorResponse(t, actual, http.StatusUnprocessableEntity)
		})

		t.Run("PATCH /v1/products/{productId} は version が進んだ後の古い version で 409 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_product.NewMockUsecase(ctrl)
			gomock.InOrder(
				mockUC.EXPECT().UpdateProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(updatedView(t), nil),
				mockUC.EXPECT().UpdateProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(productuc.ProductView{}, domainproduct.ErrVersionConflict),
			)

			productsdetail.BindHandler(e, tf, mockUC)

			headers := availableAdmin(t, e)
			srv := StartServer(t, e)
			staleBody := &productsdetailgen.PatchProductsDetailJSONRequestBody{Version: 1, Name: ptr.To("更新後の商品")}

			first := srv.DoJSON(http.MethodPatch, productDetailExistingPath, staleBody, headers)
			require.Equal(t, http.StatusOK, first.StatusCode)

			var updated productsdetailgen.ProductResponse
			require.NoError(t, json.NewDecoder(first.Body).Decode(&updated))
			require.Equal(t, int32(2), updated.Version)

			second := srv.DoJSON(http.MethodPatch, productDetailExistingPath, staleBody, headers)
			AssertErrorResponse(t, second, http.StatusConflict)
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
