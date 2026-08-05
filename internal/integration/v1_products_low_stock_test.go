package integration

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	productslowstockhandler "go-boilerplate/internal/controller/handler/v1/products/lowstock"
	"go-boilerplate/internal/controller/handler/v1/products/lowstock/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	productuc "go-boilerplate/internal/usecase/product"
	mock_product "go-boilerplate/internal/usecase/product/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/ptr"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const productsLowStockPath = "/v1/products/low-stock"

func TestV1ProductsLowStock_Integration(t *testing.T) {
	t.Parallel()

	// lowStockView は、在庫の少ない順に並んだ在庫僅少商品（閾値ちょうどの境界行を含む）を返します。
	lowStockView := func(t *testing.T) productuc.ProductLowStockListView {
		t.Helper()
		newItem := func(salt string, quantity, threshold int) productuc.ProductView {
			return productuc.ProductView{
				ID:                    uuidtestkit.NewTestFromSalt(t, salt),
				Name:                  "商品-" + salt,
				Description:           ptr.To("説明-" + salt),
				Price:                 decimaltestkit.MustParse(t, "19.99"),
				Quantity:              quantity,
				StockWarningThreshold: ptr.To(threshold),
				StatusID:              uuidtestkit.NewTestFromSalt(t, salt+"_status"),
				StatusName:            "在庫わずか",
				CategoryID:            uuidtestkit.NewTestFromSalt(t, salt+"_category"),
				CategoryName:          "電子機器",
				PublishedAt:           nil,
				ImagePath:             nil,
				Version:               1,
			}
		}
		return productuc.ProductLowStockListView{
			Items: []productuc.ProductView{
				newItem("int_lowstock_zero", 0, 3),
				newItem("int_lowstock_equal", 3, 3),
			},
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みで在庫僅少一覧がProductLowStockResponseで返る", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_product.NewMockUsecase(ctrl)
			uc.EXPECT().ListLowStockProducts(gomock.Any(), gomock.Any(), gomock.Any()).Return(lowStockView(t), nil)

			productslowstockhandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_lowstock_admin"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, productsLowStockPath, nil, headers)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[gen.ProductLowStockResponse](t, actual)
		})

		t.Run("limitがユースケースへ伝わる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var captured productuc.ListLowStockProductsParams
			uc := mock_product.NewMockUsecase(ctrl)
			uc.EXPECT().ListLowStockProducts(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(
					_ context.Context, _ *auth.Authn, params productuc.ListLowStockProductsParams,
				) (productuc.ProductLowStockListView, error) {
					captured = params
					return lowStockView(t), nil
				})

			productslowstockhandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_lowstock_limit"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, productsLowStockPath+"?limit=5", nil, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)
			assert.Equal(t, 5, captured.Limit)
		})

		t.Run("OpenAPIバリデーション経由でも範囲内のlimitは200で通過する", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_product.NewMockUsecase(ctrl)
			uc.EXPECT().ListLowStockProducts(gomock.Any(), gomock.Any(), gomock.Any()).Return(lowStockView(t), nil)

			productslowstockhandler.BindHandler(e, tf, uc)
			useOpenAPIValidation(t, e)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_lowstock_valid"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, productsLowStockPath+"?limit=100", nil, headers)
			AssertJSONResponseType[gen.ProductLowStockResponse](t, actual)
		})

		t.Run("対象商品が無い場合は空配列を200で返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_product.NewMockUsecase(ctrl)
			uc.EXPECT().ListLowStockProducts(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductLowStockListView{}, nil)

			productslowstockhandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_lowstock_empty"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, productsLowStockPath, nil, headers)
			// 対象が空でも products は null ではなく [] でシリアライズされる（AssertJSONResponseType が検査）。
			AssertJSONResponseType[gen.ProductLowStockResponse](t, actual)
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

			uc := mock_product.NewMockUsecase(ctrl)
			uc.EXPECT().ListLowStockProducts(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			productslowstockhandler.BindHandler(e, tf, uc)

			actual := StartServer(t, e).DoJSON(http.MethodGet, productsLowStockPath, nil, nil)
			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("非adminの権限エラーを403へ変換する", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_product.NewMockUsecase(ctrl)
			uc.EXPECT().ListLowStockProducts(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductLowStockListView{}, apperror.ErrPermissionDenied)

			productslowstockhandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_lowstock_forbidden"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, productsLowStockPath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusForbidden)
		})

		t.Run("OpenAPIバリデーションが範囲外のlimitを400で弾く", func(t *testing.T) {
			t.Parallel()

			t.Run("limitが下限未満(0)", func(t *testing.T) {
				t.Parallel()
				assertLowStockLimitRejected(t, productsLowStockPath+"?limit=0")
			})

			t.Run("limitが上限超過(101)", func(t *testing.T) {
				t.Parallel()
				assertLowStockLimitRejected(t, productsLowStockPath+"?limit=101")
			})
		})

		t.Run("ErrInternalで500を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_product.NewMockUsecase(ctrl)
			uc.EXPECT().ListLowStockProducts(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductLowStockListView{}, apperror.ErrInternal)

			productslowstockhandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_lowstock_err"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, productsLowStockPath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}

// assertLowStockLimitRejected は、OpenAPI バリデータが limit を 400 で弾き Usecase へ到達しないことを検証します。
func assertLowStockLimitRejected(t *testing.T, path string) {
	t.Helper()

	e := echo.New()
	UseAppErrorHandler(t, e)
	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)

	uc := mock_product.NewMockUsecase(ctrl)
	uc.EXPECT().ListLowStockProducts(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	productslowstockhandler.BindHandler(e, tf, uc)
	useOpenAPIValidation(t, e)

	headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_lowstock_badlimit"))
	actual := StartServer(t, e).DoJSON(http.MethodGet, path, nil, headers)
	AssertErrorResponse(t, actual, http.StatusBadRequest)
}
