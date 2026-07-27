package productranking

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/controller/handler/v1/products/ranking/gen"
	"go-boilerplate/internal/observability"
	rankinguc "go-boilerplate/internal/usecase/product/ranking"
	mock_ranking "go-boilerplate/internal/usecase/product/ranking/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/products/ranking"

func newServer(t *testing.T) (*server, *mock_ranking.MockUsecase) {
	t.Helper()
	mockUC := mock_ranking.NewMockUsecase(gomock.NewController(t))
	return &server{tracer: observability.NewMockControllerLayerTracer(t), uc: mockUC}, mockUC
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	mockUC := mock_ranking.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, mockUC)

	testassert.AssertEchoRouterPath(t, targetPath, e.Router().Routes())
	testassert.AssertEchoRouterMethods(t, []string{http.MethodGet}, e.Router().Routes())
}

func Test_server_GetProductsRanking(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("usecaseのRankingViewをレスポンスへ詰め替える", func(t *testing.T) {
			t.Parallel()

			productID, err := uuid.Parse("a1000000-0000-4000-8000-000000000001")
			require.NoError(t, err)

			s, mockUC := newServer(t)
			mockUC.EXPECT().GetProductsRanking(gomock.Any(), gomock.Any()).Return(rankinguc.RankingView{
				Rankings: []rankinguc.RankingItemView{
					{ProductID: productID, Name: "商品A", Price: decimaltestkit.MustParse(t, "19.99"), SoldQuantity: 8},
				},
			}, nil)

			resp, err := s.GetProductsRanking(context.Background(), gen.GetProductsRankingRequestObject{
				Params: gen.GetProductsRankingParams{Period: ptr.To(gen.GetProductsRankingParamsPeriodAll), Limit: ptr.To(10)},
			})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetProductsRanking200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, gen.GetProductsRanking200JSONResponse{
				Rankings: []gen.ProductRankingItem{
					{ProductId: productID.ToPrimitive(), Name: "商品A", Price: "19.99", SoldQuantity: 8},
				},
			}, actual)
		})

		t.Run("period/limit未指定は空文字と0でusecaseへ渡す", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().GetProductsRanking(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params rankinguc.GetRankingParams) (rankinguc.RankingView, error) {
					assert.Empty(t, params.Period)
					assert.Equal(t, 0, params.Limit)
					return rankinguc.RankingView{Rankings: []rankinguc.RankingItemView{}}, nil
				})

			resp, err := s.GetProductsRanking(context.Background(), gen.GetProductsRankingRequestObject{
				Params: gen.GetProductsRankingParams{},
			})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetProductsRanking200JSONResponse)
			require.True(t, ok)
			assert.NotNil(t, actual.Rankings)
			assert.Empty(t, actual.Rankings)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("usecaseのエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().GetProductsRanking(gomock.Any(), gomock.Any()).Return(rankinguc.RankingView{}, apperror.ErrInternal)

			resp, err := s.GetProductsRanking(context.Background(), gen.GetProductsRankingRequestObject{})
			require.ErrorIs(t, err, apperror.ErrInternal)
			assert.Nil(t, resp)
		})
	})
}

func Test_periodParam(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未指定は空文字、指定時はその文字列表現を返す", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, periodParam(nil))
			assert.Equal(t, "30d", periodParam(ptr.To(gen.GetProductsRankingParamsPeriodN30d)))
		})
	})
}

func Test_limitParam(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未指定は0、指定時はその値を返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, 0, limitParam(nil))
			assert.Equal(t, 25, limitParam(ptr.To(25)))
		})
	})
}

func Test_toProductRankingResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("RankingViewの各項目をレスポンス項目へ写像する", func(t *testing.T) {
			t.Parallel()

			productID, err := uuid.Parse("a1000000-0000-4000-8000-000000000001")
			require.NoError(t, err)

			got := toProductRankingResponse(rankinguc.RankingView{
				Rankings: []rankinguc.RankingItemView{
					{ProductID: productID, Name: "商品A", Price: decimaltestkit.MustParse(t, "19.99"), SoldQuantity: 8},
				},
			})

			assert.Equal(t, gen.ProductRankingResponse{
				Rankings: []gen.ProductRankingItem{
					{ProductId: productID.ToPrimitive(), Name: "商品A", Price: "19.99", SoldQuantity: 8},
				},
			}, got)
		})

		t.Run("空一覧は空スライスの項目を返す", func(t *testing.T) {
			t.Parallel()

			got := toProductRankingResponse(rankinguc.RankingView{})
			assert.NotNil(t, got.Rankings)
			assert.Empty(t, got.Rankings)
		})
	})
}
