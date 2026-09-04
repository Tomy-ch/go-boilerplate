package productranking

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
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

const (
	quantityPath = "/v1/products/ranking/quantity"
	amountPath   = "/v1/products/ranking/amount"
)

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

	got := make(map[string]bool, len(e.Router().Routes()))
	for _, r := range e.Router().Routes() {
		got[r.Method+" "+r.Path] = true
	}

	expected := []string{
		http.MethodGet + " " + quantityPath,
		http.MethodGet + " " + amountPath,
	}

	assert.Len(t, e.Router().Routes(), len(expected))
	for _, route := range expected {
		assert.Contains(t, got, route)
	}
}

func Test_server_GetProductsRankingQuantity(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("usecaseのRankingViewをレスポンスへ詰め替える", func(t *testing.T) {
			t.Parallel()

			productID, err := uuid.Parse("a1000000-0000-4000-8000-000000000001")
			require.NoError(t, err)

			s, mockUC := newServer(t)
			mockUC.EXPECT().GetQuantityRanking(gomock.Any(), gomock.Any()).Return(rankinguc.QuantityRankingView{
				Rankings: []rankinguc.QuantityRankingItemView{
					{ProductID: productID, Name: "商品A", Price: decimaltestkit.MustParse(t, "19.99"), SoldQuantity: 8},
				},
			}, nil)

			resp, err := s.GetProductsRankingQuantity(context.Background(), gen.GetProductsRankingQuantityRequestObject{
				Params: gen.GetProductsRankingQuantityParams{Limit: ptr.To(10)},
			})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetProductsRankingQuantity200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, gen.GetProductsRankingQuantity200JSONResponse{
				Rankings: []gen.ProductQuantityRankingItem{
					{ProductId: productID.ToPrimitive(), Name: "商品A", Price: "19.99", SoldQuantity: 8},
				},
			}, actual)
		})

		t.Run("期間/limit未指定は境界を持たない対象期間と0でusecaseへ渡す", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().GetQuantityRanking(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params rankinguc.GetRankingParams) (rankinguc.QuantityRankingView, error) {
					assert.Nil(t, params.Window.After())
					assert.Nil(t, params.Window.Before())
					assert.Equal(t, 0, params.Limit)
					return rankinguc.QuantityRankingView{Rankings: []rankinguc.QuantityRankingItemView{}}, nil
				},
			)

			resp, err := s.GetProductsRankingQuantity(context.Background(), gen.GetProductsRankingQuantityRequestObject{
				Params: gen.GetProductsRankingQuantityParams{},
			})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetProductsRankingQuantity200JSONResponse)
			require.True(t, ok)
			assert.NotNil(t, actual.Rankings)
			assert.Empty(t, actual.Rankings)
		})

		t.Run("orderedAfter・orderedBeforeを対象期間へ変換してusecaseへ渡す", func(t *testing.T) {
			t.Parallel()

			after := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
			before := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)

			s, mockUC := newServer(t)
			mockUC.EXPECT().GetQuantityRanking(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params rankinguc.GetRankingParams) (rankinguc.QuantityRankingView, error) {
					assert.Equal(t, after, *params.Window.After())
					assert.Equal(t, before, *params.Window.Before())
					return rankinguc.QuantityRankingView{Rankings: []rankinguc.QuantityRankingItemView{}}, nil
				},
			)

			_, err := s.GetProductsRankingQuantity(context.Background(), gen.GetProductsRankingQuantityRequestObject{
				Params: gen.GetProductsRankingQuantityParams{OrderedAfter: &after, OrderedBefore: &before},
			})
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("orderedBeforeがorderedAfter以前の場合、usecaseを呼ばずErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().GetQuantityRanking(gomock.Any(), gomock.Any()).Times(0)

			after := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
			before := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)

			_, err := s.GetProductsRankingQuantity(context.Background(), gen.GetProductsRankingQuantityRequestObject{
				Params: gen.GetProductsRankingQuantityParams{OrderedAfter: &after, OrderedBefore: &before},
			})
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("usecaseのエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().GetQuantityRanking(gomock.Any(), gomock.Any()).Return(rankinguc.QuantityRankingView{}, apperror.ErrInternal)

			resp, err := s.GetProductsRankingQuantity(context.Background(), gen.GetProductsRankingQuantityRequestObject{})
			require.ErrorIs(t, err, apperror.ErrInternal)
			assert.Nil(t, resp)
		})
	})
}

func Test_server_GetProductsRankingAmount(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("usecaseのDTOを金額ランキングのレスポンスへ写像して返す", func(t *testing.T) {
			t.Parallel()

			productID, err := uuid.Parse("a1000000-0000-4000-8000-000000000001")
			require.NoError(t, err)

			s, mockUC := newServer(t)
			mockUC.EXPECT().GetAmountRanking(gomock.Any(), gomock.Any()).Return(rankinguc.AmountRankingView{
				Rankings: []rankinguc.AmountRankingItemView{
					{
						ProductID:   productID,
						Name:        "商品A",
						Price:       decimaltestkit.MustParse(t, "19.995"),
						SalesAmount: decimaltestkit.MustParse(t, "59.985"),
					},
				},
			}, nil)

			resp, err := s.GetProductsRankingAmount(context.Background(), gen.GetProductsRankingAmountRequestObject{
				Params: gen.GetProductsRankingAmountParams{Limit: ptr.To(10)},
			})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetProductsRankingAmount200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, gen.GetProductsRankingAmount200JSONResponse{
				Rankings: []gen.ProductAmountRankingItem{
					{ProductId: productID.ToPrimitive(), Name: "商品A", Price: "19.995", SalesAmount: "59.985"},
				},
			}, actual)
		})

		t.Run("期間/limit未指定は境界を持たない対象期間と0でusecaseへ渡す", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().GetAmountRanking(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params rankinguc.GetRankingParams) (rankinguc.AmountRankingView, error) {
					assert.Nil(t, params.Window.After())
					assert.Nil(t, params.Window.Before())
					assert.Equal(t, 0, params.Limit)
					return rankinguc.AmountRankingView{Rankings: []rankinguc.AmountRankingItemView{}}, nil
				},
			)

			_, err := s.GetProductsRankingAmount(context.Background(), gen.GetProductsRankingAmountRequestObject{
				Params: gen.GetProductsRankingAmountParams{},
			})
			require.NoError(t, err)
		})

		t.Run("orderedAfter・orderedBeforeを対象期間へ変換してusecaseへ渡す", func(t *testing.T) {
			t.Parallel()

			after := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
			before := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)

			s, mockUC := newServer(t)
			mockUC.EXPECT().GetAmountRanking(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params rankinguc.GetRankingParams) (rankinguc.AmountRankingView, error) {
					assert.Equal(t, after, *params.Window.After())
					assert.Equal(t, before, *params.Window.Before())
					return rankinguc.AmountRankingView{Rankings: []rankinguc.AmountRankingItemView{}}, nil
				},
			)

			_, err := s.GetProductsRankingAmount(context.Background(), gen.GetProductsRankingAmountRequestObject{
				Params: gen.GetProductsRankingAmountParams{OrderedAfter: &after, OrderedBefore: &before},
			})
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("orderedBeforeがorderedAfter以前の場合、usecaseを呼ばずErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().GetAmountRanking(gomock.Any(), gomock.Any()).Times(0)

			after := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
			before := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)

			_, err := s.GetProductsRankingAmount(context.Background(), gen.GetProductsRankingAmountRequestObject{
				Params: gen.GetProductsRankingAmountParams{OrderedAfter: &after, OrderedBefore: &before},
			})
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("usecaseのエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().GetAmountRanking(gomock.Any(), gomock.Any()).
				Return(rankinguc.AmountRankingView{}, apperror.ErrInternal)

			resp, err := s.GetProductsRankingAmount(context.Background(), gen.GetProductsRankingAmountRequestObject{})
			require.ErrorIs(t, err, apperror.ErrInternal)
			assert.Nil(t, resp)
		})
	})
}

func Test_toAmountRankingResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("金額と単価を丸めずに文字列へ写像する", func(t *testing.T) {
			t.Parallel()

			productID, err := uuid.Parse("a1000000-0000-4000-8000-000000000002")
			require.NoError(t, err)

			got := toAmountRankingResponse(rankinguc.AmountRankingView{
				Rankings: []rankinguc.AmountRankingItemView{
					{
						ProductID:   productID,
						Name:        "商品B",
						Price:       decimaltestkit.MustParse(t, "0.005"),
						SalesAmount: decimaltestkit.MustParse(t, "0.015"),
					},
				},
			})

			require.Len(t, got.Rankings, 1)
			assert.Equal(t, "0.005", got.Rankings[0].Price)
			assert.Equal(t, "0.015", got.Rankings[0].SalesAmount)
		})

		t.Run("項目が空の場合はnilでない空スライスを返す", func(t *testing.T) {
			t.Parallel()

			got := toAmountRankingResponse(rankinguc.AmountRankingView{})

			assert.NotNil(t, got.Rankings)
			assert.Empty(t, got.Rankings)
		})
	})
}

func Test_limitParam(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未指定の場合、0を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 0, limitParam(nil))
		})

		t.Run("指定時はその値を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 25, limitParam(ptr.To(25)))
		})
	})
}

func Test_toQuantityRankingResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("RankingViewの各項目をレスポンス項目へ写像する", func(t *testing.T) {
			t.Parallel()

			productID, err := uuid.Parse("a1000000-0000-4000-8000-000000000001")
			require.NoError(t, err)

			got := toQuantityRankingResponse(rankinguc.QuantityRankingView{
				Rankings: []rankinguc.QuantityRankingItemView{
					{ProductID: productID, Name: "商品A", Price: decimaltestkit.MustParse(t, "19.99"), SoldQuantity: 8},
				},
			})

			assert.Equal(t, gen.ProductQuantityRankingResponse{
				Rankings: []gen.ProductQuantityRankingItem{
					{ProductId: productID.ToPrimitive(), Name: "商品A", Price: "19.99", SoldQuantity: 8},
				},
			}, got)
		})

		t.Run("空一覧は空スライスの項目を返す", func(t *testing.T) {
			t.Parallel()

			got := toQuantityRankingResponse(rankinguc.QuantityRankingView{})
			assert.NotNil(t, got.Rankings)
			assert.Empty(t, got.Rankings)
		})
	})
}
