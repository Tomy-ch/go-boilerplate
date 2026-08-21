package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	productsrankinghandler "go-boilerplate/internal/controller/handler/v1/products/ranking"
	"go-boilerplate/internal/controller/handler/v1/products/ranking/gen"
	"go-boilerplate/internal/controller/httpstack/oapi"
	"go-boilerplate/internal/controller/httpstack/oapi/validator"
	"go-boilerplate/internal/observability"
	rankinguc "go-boilerplate/internal/usecase/product/ranking"
	mock_ranking "go-boilerplate/internal/usecase/product/ranking/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// useOpenAPIValidation は、OpenAPI スキーマに基づくリクエストバリデーションミドルウェアを Echo に登録します。
func useOpenAPIValidation(t *testing.T, e *echo.Echo) {
	t.Helper()

	spec, err := validator.GetValidator()
	require.NoError(t, err)

	skipper := func(*echo.Context) bool { return false }
	authFunc := func(context.Context, *openapi3filter.AuthenticationInput) error { return nil }
	e.Use(oapi.Middleware(spec, skipper, authFunc))
}

func TestV1ProductsRankingQuantity_Integration(t *testing.T) {
	t.Parallel()

	sampleView := func(t *testing.T) rankinguc.QuantityRankingView {
		t.Helper()
		return rankinguc.QuantityRankingView{
			Rankings: []rankinguc.QuantityRankingItemView{
				{
					ProductID:    uuidtestkit.NewTestFromSalt(t, "integration_ranking_product"),
					Name:         "商品",
					Price:        decimaltestkit.MustParse(t, "19.99"),
					SoldQuantity: 8,
				},
			},
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/products/ranking/quantity が未認証でも数量ランキングを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_ranking.NewMockUsecase(ctrl)
			mockUC.EXPECT().GetQuantityRanking(gomock.Any(), gomock.Any()).Return(sampleView(t), nil)

			productsrankinghandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products/ranking/quantity?limit=10", nil, nil)
			AssertJSONResponseType[gen.ProductQuantityRankingResponse](t, actual)
		})

		t.Run("OpenAPIバリデーション経由でも範囲内パラメータは200で通過する", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_ranking.NewMockUsecase(ctrl)
			mockUC.EXPECT().GetQuantityRanking(gomock.Any(), gomock.Any()).Return(sampleView(t), nil)

			productsrankinghandler.BindHandler(e, tf, mockUC)
			useOpenAPIValidation(t, e)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products/ranking/quantity?orderedAfter=2026-01-21T00:00:00Z&limit=100", nil, nil)
			AssertJSONResponseType[gen.ProductQuantityRankingResponse](t, actual)
		})

		t.Run("GET /v1/products/ranking?limit=3 が limit と対象期間を usecase へ伝える", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_ranking.NewMockUsecase(ctrl)
			mockUC.EXPECT().GetQuantityRanking(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params rankinguc.GetRankingParams) (rankinguc.QuantityRankingView, error) {
					assert.Equal(t, 3, params.Limit)
					require.NotNil(t, params.Window.After())
					assert.True(t, time.Date(2026, time.January, 21, 0, 0, 0, 0, time.UTC).Equal(*params.Window.After()))
					return sampleView(t), nil
				})

			productsrankinghandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(
				http.MethodGet, "/v1/products/ranking/quantity?orderedAfter=2026-01-21T00:00:00Z&limit=3", nil, nil)
			AssertJSONResponseType[gen.ProductQuantityRankingResponse](t, actual)
		})

		t.Run("廃止したperiodを送っても無視され全期間として扱われる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_ranking.NewMockUsecase(ctrl)
			mockUC.EXPECT().GetQuantityRanking(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params rankinguc.GetRankingParams) (rankinguc.QuantityRankingView, error) {
					assert.Nil(t, params.Window.After())
					assert.Nil(t, params.Window.Before())
					return sampleView(t), nil
				})

			productsrankinghandler.BindHandler(e, tf, mockUC)
			useOpenAPIValidation(t, e)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products/ranking/quantity?period=30d", nil, nil)
			AssertJSONResponseType[gen.ProductQuantityRankingResponse](t, actual)
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
			mockUC.EXPECT().GetQuantityRanking(gomock.Any(), gomock.Any()).Times(0)

			productsrankinghandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products/ranking/quantity?limit=abc", nil, nil)
			AssertErrorResponse(t, actual, http.StatusBadRequest)
		})

		t.Run("OpenAPIバリデーションが範囲外・不正なパラメータを400で弾く", func(t *testing.T) {
			t.Parallel()

			cases := map[string]string{
				"limitが下限未満(0)":         "/v1/products/ranking/quantity?limit=0",
				"limitが上限超過(101)":       "/v1/products/ranking/quantity?limit=101",
				"orderedAfterが日時形式でない":  "/v1/products/ranking/quantity?orderedAfter=2026-01-21",
				"orderedBeforeが日時形式でない": "/v1/products/ranking/quantity?orderedBefore=yesterday",
			}
			for name, path := range cases {
				t.Run(name, func(t *testing.T) {
					t.Parallel()

					e := echo.New()
					UseAppErrorHandler(t, e)
					ctrl := gomock.NewController(t)
					tf := observability.NewNoopTracerFactory(t)

					mockUC := mock_ranking.NewMockUsecase(ctrl)
					mockUC.EXPECT().GetQuantityRanking(gomock.Any(), gomock.Any()).Times(0)

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
			mockUC.EXPECT().GetQuantityRanking(gomock.Any(), gomock.Any()).Return(rankinguc.QuantityRankingView{}, apperror.ErrInternal)

			productsrankinghandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products/ranking/quantity", nil, nil)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}

func TestV1ProductsRankingAmount_Integration(t *testing.T) {
	t.Parallel()

	sampleView := func(t *testing.T) rankinguc.AmountRankingView {
		t.Helper()
		return rankinguc.AmountRankingView{
			Rankings: []rankinguc.AmountRankingItemView{
				{
					ProductID:   uuidtestkit.NewTestFromSalt(t, "integration_amount_product"),
					Name:        "商品",
					Price:       decimaltestkit.MustParse(t, "19.995"),
					SalesAmount: decimaltestkit.MustParse(t, "59.985"),
				},
			},
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/products/ranking/amount が金額ランキングを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_ranking.NewMockUsecase(ctrl)
			mockUC.EXPECT().GetAmountRanking(gomock.Any(), gomock.Any()).Return(sampleView(t), nil)

			productsrankinghandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products/ranking/amount?limit=10", nil, nil)
			AssertJSONResponseType[gen.ProductAmountRankingResponse](t, actual)
		})

		t.Run("OpenAPIバリデーション経由でもサブセントの金額が丸められずに返る", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_ranking.NewMockUsecase(ctrl)
			mockUC.EXPECT().GetAmountRanking(gomock.Any(), gomock.Any()).Return(sampleView(t), nil)

			productsrankinghandler.BindHandler(e, tf, mockUC)
			useOpenAPIValidation(t, e)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/products/ranking/amount?limit=10", nil, nil)
			require.Equal(t, http.StatusOK, actual.StatusCode)

			raw, rerr := io.ReadAll(actual.Body)
			require.NoError(t, rerr)
			var body gen.ProductAmountRankingResponse
			require.NoError(t, json.Unmarshal(raw, &body))
			require.Len(t, body.Rankings, 1)
			// 契約の pattern を通ったうえで、決済スケールへ丸められていないことを HTTP 経路で固定する。
			assert.Equal(t, "59.985", body.Rankings[0].SalesAmount)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("OpenAPIバリデーションが範囲外・不正なパラメータを400で弾く", func(t *testing.T) {
			t.Parallel()

			cases := map[string]string{
				"limitが上限超過(101)":      "/v1/products/ranking/amount?limit=101",
				"orderedAfterが日時形式でない": "/v1/products/ranking/amount?orderedAfter=2026-01-21",
			}
			for name, path := range cases {
				t.Run(name, func(t *testing.T) {
					t.Parallel()

					e := echo.New()
					UseAppErrorHandler(t, e)
					ctrl := gomock.NewController(t)
					tf := observability.NewNoopTracerFactory(t)

					mockUC := mock_ranking.NewMockUsecase(ctrl)
					mockUC.EXPECT().GetAmountRanking(gomock.Any(), gomock.Any()).Times(0)

					productsrankinghandler.BindHandler(e, tf, mockUC)
					useOpenAPIValidation(t, e)

					actual := StartServer(t, e).DoJSON(http.MethodGet, path, nil, nil)
					AssertErrorResponse(t, actual, http.StatusBadRequest)
				})
			}
		})
	})
}
