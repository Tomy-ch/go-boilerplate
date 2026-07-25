package ranking

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/product/ranking/query"
	mock_query "go-boilerplate/internal/usecase/product/ranking/query/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newUsecase(t *testing.T) (*usecase, *mock_query.MockProductRankingQueryService) {
	t.Helper()
	mockQS := mock_query.NewMockProductRankingQueryService(gomock.NewController(t))
	return &usecase{tracer: observability.NewMockUsecaseLayerTracer(t), qs: mockQS}, mockQS
}

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)
	qs := mock_query.NewMockProductRankingQueryService(ctrl)

	expected := &usecase{
		tracer: tf.Usecase(),
		qs:     qs,
	}
	actual := New(qs, tf)

	assert.Equal(t, expected, actual)
}

func Test_usecase_GetProductsRanking(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("正規化した期間と件数でQSを呼び集計結果をRankingViewへ写像して返す", func(t *testing.T) {
			t.Parallel()

			productID, err := uuid.Parse("a1000000-0000-4000-8000-000000000001")
			require.NoError(t, err)

			u, mockQS := newUsecase(t)
			mockQS.EXPECT().ListRanking(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params query.RankingQueryParams) ([]query.RankingResult, error) {
					assert.Equal(t, query.Period30d, params.Period)
					assert.Equal(t, defaultLimit, params.Limit)
					return []query.RankingResult{
						{ProductID: productID, Name: "商品A", Price: decimaltestkit.MustParse(t, "19.99"), SoldQuantity: 8},
					}, nil
				})

			got, err := u.GetProductsRanking(context.Background(), GetRankingParams{Period: "30d", Limit: 0})
			require.NoError(t, err)

			require.Len(t, got.Rankings, 1)
			assert.Equal(t, productID, got.Rankings[0].ProductID)
			assert.Equal(t, "商品A", got.Rankings[0].Name)
			assert.Equal(t, "19.99", got.Rankings[0].Price.String())
			assert.Equal(t, int64(8), got.Rankings[0].SoldQuantity)
		})

		t.Run("QSが空を返す場合RankingViewはnilでない空スライスを返す", func(t *testing.T) {
			t.Parallel()

			u, mockQS := newUsecase(t)
			mockQS.EXPECT().ListRanking(gomock.Any(), gomock.Any()).Return([]query.RankingResult{}, nil)

			got, err := u.GetProductsRanking(context.Background(), GetRankingParams{Period: "all", Limit: 10})
			require.NoError(t, err)

			assert.NotNil(t, got.Rankings)
			assert.Empty(t, got.Rankings)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("QSのエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			u, mockQS := newUsecase(t)
			mockQS.EXPECT().ListRanking(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrInternal)

			got, err := u.GetProductsRanking(context.Background(), GetRankingParams{Period: "all", Limit: 10})
			require.ErrorIs(t, err, apperror.ErrInternal)
			assert.Empty(t, got.Rankings)
		})
	})
}

func Test_normalizePeriod(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("30dは集計区分30dへ、それ以外は全期間へ正規化する", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, query.Period30d, normalizePeriod("30d"))
			assert.Equal(t, query.PeriodAll, normalizePeriod("all"))
			assert.Equal(t, query.PeriodAll, normalizePeriod(""))
			assert.Equal(t, query.PeriodAll, normalizePeriod("weekly"))
		})
	})
}

func Test_normalizeLimit(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("0以下は既定値、範囲外はクランプ、範囲内はそのまま返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, defaultLimit, normalizeLimit(0))
			assert.Equal(t, defaultLimit, normalizeLimit(-5))
			assert.Equal(t, minLimit, normalizeLimit(1))
			assert.Equal(t, 50, normalizeLimit(50))
			assert.Equal(t, maxLimit, normalizeLimit(100))
			assert.Equal(t, maxLimit, normalizeLimit(101))
		})
	})
}
