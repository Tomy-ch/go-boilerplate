package integration

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	productsrankinghandler "go-boilerplate/internal/controller/handler/v1/products/ranking"
	"go-boilerplate/internal/controller/handler/v1/products/ranking/gen"
	"go-boilerplate/internal/controller/httpstack/oapi"
	"go-boilerplate/internal/controller/httpstack/oapi/validator"
	"go-boilerplate/internal/observability"
	rankinguc "go-boilerplate/internal/usecase/product/ranking"
	mock_ranking "go-boilerplate/internal/usecase/product/ranking/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/uuid"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// useOpenAPIValidation は、OpenAPI スキーマに基づくリクエストバリデーションミドルウェアを Echo に登録します。
// httptest のホストと spec の servers が不一致でも経路解決できるよう servers を空にします。
func useOpenAPIValidation(t *testing.T, e *echo.Echo) {
	t.Helper()

	spec, err := validator.GetValidator()
	require.NoError(t, err)
	spec.Servers = nil

	skipper := func(echo.Context) bool { return false }
	authFunc := func(context.Context, *openapi3filter.AuthenticationInput) error { return nil }
	e.Use(oapi.Middleware(spec, skipper, authFunc))
}

func TestV1ProductsRanking_Integration(t *testing.T) {
	t.Parallel()

	sampleView := func(t *testing.T) rankinguc.RankingView {
		t.Helper()
		return rankinguc.RankingView{
			Rankings: []rankinguc.RankingItemView{
				{
					ProductID:    uuid.NewTestFromSalt(t, "integration_ranking_product"),
					Name:         "商品",
					Price:        decimaltestkit.MustParse(t, "19.99"),
					SoldQuantity: 8,
				},
			},
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/products/ranking が未認証でも ProductRankingResponse を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_ranking.NewMockUsecase(ctrl)
			mockUC.EXPECT().GetProductsRanking(gomock.Any(), gomock.Any()).Return(sampleView(t), nil)

			productsrankinghandler.BindHandler(e, tf, mockUC)

			// security: [] の公開エンドポイントのため、Authorization ヘッダー無しでも 200 が返る。
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products/ranking?period=all&limit=10", nil, nil)
			AssertJSONResponseType[gen.ProductRankingResponse](t, actual)
		})

		t.Run("OpenAPIバリデーション経由でも範囲内パラメータは200で通過する", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_ranking.NewMockUsecase(ctrl)
			mockUC.EXPECT().GetProductsRanking(gomock.Any(), gomock.Any()).Return(sampleView(t), nil)

			productsrankinghandler.BindHandler(e, tf, mockUC)
			useOpenAPIValidation(t, e)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products/ranking?period=30d&limit=100", nil, nil)
			AssertJSONResponseType[gen.ProductRankingResponse](t, actual)
		})

		t.Run("GET /v1/products/ranking?limit=3 が limit を usecase へ伝える", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_ranking.NewMockUsecase(ctrl)
			mockUC.EXPECT().GetProductsRanking(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params rankinguc.GetRankingParams) (rankinguc.RankingView, error) {
					assert.Equal(t, 3, params.Limit)
					assert.Equal(t, "30d", params.Period)
					return sampleView(t), nil
				})

			productsrankinghandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products/ranking?period=30d&limit=3", nil, nil)
			AssertJSONResponseType[gen.ProductRankingResponse](t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/products/ranking が数値でない limit で 400 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_ranking.NewMockUsecase(ctrl)
			mockUC.EXPECT().GetProductsRanking(gomock.Any(), gomock.Any()).Times(0)

			productsrankinghandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products/ranking?limit=abc", nil, nil)
			AssertErrorResponse(t, actual, http.StatusBadRequest)
		})

		t.Run("OpenAPIバリデーションが範囲外・不正なパラメータを400で弾く", func(t *testing.T) {
			t.Parallel()

			cases := map[string]string{
				"limitが下限未満(0)":      "/v1/products/ranking?limit=0",
				"limitが上限超過(101)":    "/v1/products/ranking?limit=101",
				"periodが列挙外(weekly)": "/v1/products/ranking?period=weekly",
			}
			for name, path := range cases {
				t.Run(name, func(t *testing.T) {
					t.Parallel()

					e := echo.New()
					UseAppErrorHandler(t, e)
					ctrl := gomock.NewController(t)
					tf := observability.NewNoopTracerFactory(t)

					mockUC := mock_ranking.NewMockUsecase(ctrl)
					mockUC.EXPECT().GetProductsRanking(gomock.Any(), gomock.Any()).Times(0)

					productsrankinghandler.BindHandler(e, tf, mockUC)
					useOpenAPIValidation(t, e)

					actual := StartServer(t, e).DoJSON(http.MethodGet, path, nil, nil)
					AssertErrorResponse(t, actual, http.StatusBadRequest)
				})
			}
		})

		t.Run("GET /v1/products/ranking が ErrInternal で 500 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_ranking.NewMockUsecase(ctrl)
			mockUC.EXPECT().GetProductsRanking(gomock.Any(), gomock.Any()).Return(rankinguc.RankingView{}, apperror.ErrInternal)

			productsrankinghandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products/ranking", nil, nil)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}
