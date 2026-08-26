package ranking

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/product/ranking/query"
	mock_query "go-boilerplate/internal/usecase/product/ranking/query/mock"
	"go-boilerplate/internal/usecase/tools/timewindow"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

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

func Test_usecase_GetQuantityRanking(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("対象期間と正規化した件数でQSを呼び集計結果をRankingViewへ写像して返す", func(t *testing.T) {
			t.Parallel()

			productID, err := uuid.Parse("a1000000-0000-4000-8000-000000000001")
			require.NoError(t, err)
			publishedAt := time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC)

			after := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
			window, err := timewindow.New(timewindow.Bounds{After: &after})
			require.NoError(t, err)

			u, mockQS := newUsecase(t)
			mockQS.EXPECT().ListQuantityRanking(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params query.RankingQueryParams) ([]query.QuantityRankingResult, error) {
					assert.Equal(t, after, *params.Window.After())
					assert.Nil(t, params.Window.Before())
					assert.Equal(t, defaultLimit, params.Limit)
					return []query.QuantityRankingResult{
						{
							ProductID:    productID,
							Name:         "商品A",
							Price:        decimaltestkit.MustParse(t, "19.99"),
							PublishedAt:  ptr.To(publishedAt),
							SoldQuantity: 8,
						},
					}, nil
				})

			got, err := u.GetQuantityRanking(context.Background(), GetRankingParams{Window: window, Limit: 0})
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
			mockQS.EXPECT().ListQuantityRanking(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params query.RankingQueryParams) ([]query.QuantityRankingResult, error) {
					assert.Equal(t, maxLimit, params.Limit)
					return []query.QuantityRankingResult{}, nil
				})

			_, err := u.GetQuantityRanking(context.Background(), GetRankingParams{Limit: 1000})
			require.NoError(t, err)
		})

		t.Run("QSが空を返す場合RankingViewはnilでない空スライスを返す", func(t *testing.T) {
			t.Parallel()

			u, mockQS := newUsecase(t)
			mockQS.EXPECT().ListQuantityRanking(gomock.Any(), gomock.Any()).Return([]query.QuantityRankingResult{}, nil)

			got, err := u.GetQuantityRanking(context.Background(), GetRankingParams{Limit: 10})
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
			mockQS.EXPECT().ListQuantityRanking(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrInternal)

			got, err := u.GetQuantityRanking(context.Background(), GetRankingParams{Limit: 10})
			require.ErrorIs(t, err, apperror.ErrInternal)
			assert.Empty(t, got.Rankings)
		})

		t.Run("非公開の商品が集計結果に混ざる場合、ドリフトとしてErrInternalを返し結果を出力しない", func(t *testing.T) {
			t.Parallel()

			productID, err := uuid.Parse("a1000000-0000-4000-8000-000000000002")
			require.NoError(t, err)
			publishedAt := time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC)

			u, mockQS := newUsecase(t)
			mockQS.EXPECT().ListQuantityRanking(gomock.Any(), gomock.Any()).Return([]query.QuantityRankingResult{
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

			got, err := u.GetQuantityRanking(context.Background(), GetRankingParams{Limit: 10})
			require.ErrorIs(t, err, apperror.ErrInternal)
			assert.Contains(t, err.Error(), productID.String())
			assert.Empty(t, got.Rankings)
		})
	})
}

func Test_usecase_GetAmountRanking(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("集計結果をAmountRankingViewへ写像し金額を丸めずに返す", func(t *testing.T) {
			t.Parallel()

			productID, err := uuid.Parse("a1000000-0000-4000-8000-000000000001")
			require.NoError(t, err)
			publishedAt := time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC)

			u, mockQS := newUsecase(t)
			mockQS.EXPECT().ListAmountRanking(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params query.RankingQueryParams) ([]query.AmountRankingResult, error) {
					assert.Equal(t, defaultLimit, params.Limit)
					return []query.AmountRankingResult{
						{
							ProductID:   productID,
							Name:        "商品A",
							Price:       decimaltestkit.MustParse(t, "19.995"),
							PublishedAt: ptr.To(publishedAt),
							SalesAmount: decimaltestkit.MustParse(t, "59.985"),
						},
					}, nil
				})

			got, err := u.GetAmountRanking(context.Background(), GetRankingParams{Limit: 0})
			require.NoError(t, err)

			require.Len(t, got.Rankings, 1)
			assert.Equal(t, productID, got.Rankings[0].ProductID)
			assert.Equal(t, "商品A", got.Rankings[0].Name)
			// サブセントの桁が決済スケールへ丸められず、そのまま出力 DTO へ届くことを固定する。
			assert.Equal(t, "59.985", got.Rankings[0].SalesAmount.String())
			assert.Equal(t, "19.995", got.Rankings[0].Price.String())
		})

		t.Run("集計結果が空の場合はnilでない空スライスを返す", func(t *testing.T) {
			t.Parallel()

			u, mockQS := newUsecase(t)
			mockQS.EXPECT().ListAmountRanking(gomock.Any(), gomock.Any()).
				Return([]query.AmountRankingResult{}, nil)

			got, err := u.GetAmountRanking(context.Background(), GetRankingParams{Limit: 10})
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
			mockQS.EXPECT().ListAmountRanking(gomock.Any(), gomock.Any()).
				Return(nil, apperror.ErrInternal)

			_, err := u.GetAmountRanking(context.Background(), GetRankingParams{Limit: 10})
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("2行目に非公開商品が混ざっていた場合、ErrInternalを返す", func(t *testing.T) {
			t.Parallel()

			publishedAt := time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC)
			unpublishedID := uuidtestkit.NewTestFromSalt(t, "amount_unpublished")

			u, mockQS := newUsecase(t)
			mockQS.EXPECT().ListAmountRanking(gomock.Any(), gomock.Any()).
				Return([]query.AmountRankingResult{
					{ProductID: uuidtestkit.NewTestFromSalt(t, "amount_published"), PublishedAt: ptr.To(publishedAt)},
					{ProductID: unpublishedID, PublishedAt: nil},
				}, nil)

			// 1 行目を通過してから 2 行目で落ちる経路。先頭固定の検査へ縮退したらここが緑のまま通る。
			_, err := u.GetAmountRanking(context.Background(), GetRankingParams{Limit: 10})
			require.ErrorIs(t, err, apperror.ErrInternal)
			assert.Contains(t, err.Error(), unpublishedID.String())
		})
	})
}

func Test_ensurePublished(t *testing.T) {
	t.Parallel()

	publishedAt := time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC)
	productID := uuidtestkit.NewTestFromSalt(t, "ensure_published")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("公開中の行はnilを返す", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, ensurePublished(productID, ptr.To(publishedAt)))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("公開日時が未設定の場合、ErrInternalを返す", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, ensurePublished(productID, nil), apperror.ErrInternal)
		})

		t.Run("エラーは対象の商品IDを含む", func(t *testing.T) {
			t.Parallel()
			err := ensurePublished(productID, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), productID.String())
		})
	})
}

func Test_toQueryParams(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("対象期間はそのまま渡し件数だけを正規化する", func(t *testing.T) {
			t.Parallel()

			after := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
			window, err := timewindow.New(timewindow.Bounds{After: &after})
			require.NoError(t, err)

			got := toQueryParams(GetRankingParams{Window: window, Limit: 0})

			assert.Equal(t, after, *got.Window.After())
			assert.Equal(t, defaultLimit, got.Limit)
		})

		t.Run("範囲内の件数はそのまま渡る", func(t *testing.T) {
			t.Parallel()

			got := toQueryParams(GetRankingParams{Limit: 5})

			assert.Equal(t, 5, got.Limit)
		})

		t.Run("上限を超える件数はクランプする", func(t *testing.T) {
			t.Parallel()

			got := toQueryParams(GetRankingParams{Limit: 1000})

			assert.Equal(t, maxLimit, got.Limit)
		})
	})
}
