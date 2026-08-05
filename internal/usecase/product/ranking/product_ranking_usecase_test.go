package ranking

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/product/ranking/query"
	mock_query "go-boilerplate/internal/usecase/product/ranking/query/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/ptr"
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
			publishedAt := time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC)

			u, mockQS := newUsecase(t)
			mockQS.EXPECT().ListRanking(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params query.RankingQueryParams) ([]query.RankingResult, error) {
					assert.Equal(t, query.Period30d, params.Period)
					assert.Equal(t, defaultLimit, params.Limit)
					return []query.RankingResult{
						{
							ProductID:    productID,
							Name:         "商品A",
							Price:        decimaltestkit.MustParse(t, "19.99"),
							PublishedAt:  ptr.To(publishedAt),
							SoldQuantity: 8,
						},
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

		t.Run("件数が上限を超える場合、上限へクランプした件数でQSを呼ぶ", func(t *testing.T) {
			t.Parallel()

			u, mockQS := newUsecase(t)
			mockQS.EXPECT().ListRanking(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params query.RankingQueryParams) ([]query.RankingResult, error) {
					assert.Equal(t, maxLimit, params.Limit)
					return []query.RankingResult{}, nil
				})

			_, err := u.GetProductsRanking(context.Background(), GetRankingParams{Period: "all", Limit: 1000})
			require.NoError(t, err)
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

		t.Run("非公開の商品が集計結果に混ざる場合、ドリフトとしてErrInternalを返し結果を出力しない", func(t *testing.T) {
			t.Parallel()

			productID, err := uuid.Parse("a1000000-0000-4000-8000-000000000002")
			require.NoError(t, err)
			publishedAt := time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC)

			u, mockQS := newUsecase(t)
			mockQS.EXPECT().ListRanking(gomock.Any(), gomock.Any()).Return([]query.RankingResult{
				{
					ProductID:    uuid.UUID{},
					Name:         "公開中の商品",
					Price:        decimaltestkit.MustParse(t, "19.99"),
					PublishedAt:  ptr.To(publishedAt),
					SoldQuantity: 8,
				},
				{
					ProductID:    productID,
					Name:         "非公開の商品",
					Price:        decimaltestkit.MustParse(t, "29.99"),
					PublishedAt:  nil,
					SoldQuantity: 3,
				},
			}, nil)

			got, err := u.GetProductsRanking(context.Background(), GetRankingParams{Period: "all", Limit: 10})
			require.ErrorIs(t, err, apperror.ErrInternal)
			assert.Contains(t, err.Error(), productID.String())
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

func Test_ensurePublished(t *testing.T) {
	t.Parallel()

	publishedAt := time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全ての行が公開中の場合、nilを返す", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, ensurePublished([]query.RankingResult{
				{PublishedAt: ptr.To(publishedAt)},
				{PublishedAt: ptr.To(publishedAt)},
			}))
		})

		t.Run("行が無い場合、nilを返す", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, ensurePublished(nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("公開日時が未設定の行がある場合、ErrInternalを返す", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, ensurePublished([]query.RankingResult{{PublishedAt: nil}}), apperror.ErrInternal)
		})
	})
}
