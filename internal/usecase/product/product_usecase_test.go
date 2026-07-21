package product

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	domainproduct "go-boilerplate/internal/domain/product"
	mock_product "go-boilerplate/internal/domain/product/mock"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/testkit"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newTestProduct(t *testing.T, salt string, publishedAt time.Time) *domainproduct.Product {
	t.Helper()
	p, err := domainproduct.New(
		uuid.NewTestFromSalt(t, salt),
		"商品-"+salt,
		ptr.To("説明-"+salt),
		1000,
		5,
		ptr.To(2),
		uuid.NewTestFromSalt(t, salt+"_status"),
		uuid.NewTestFromSalt(t, salt+"_category"),
		publishedAt,
	)
	require.NoError(t, err)
	return p
}

// newDefaultCursor は、first=2（limit=2）の先頭ページ用カーソルを生成するテストヘルパーです。
func newDefaultCursor(t *testing.T) *paging.Cursor {
	t.Helper()
	first := 2
	c, err := paging.NewCursor(nil, &first)
	require.NoError(t, err)
	return c
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		tf := observability.NewNoopTracerFactory(t)
		repo := mock_product.NewMockRepository(ctrl)

		expected := &usecase{tracer: tf.Usecase(), repo: repo}
		actual := New(repo, tf)

		assert.Equal(t, expected, actual)
	})
}

func Test_usecase_ListProducts(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("取得件数がlimitを超える場合、末尾を切り詰めてNextCursorを設定する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_product.NewMockRepository(gomock.NewController(t))

			p1 := newTestProduct(t, "list_p1", base)
			p2 := newTestProduct(t, "list_p2", base.Add(-time.Hour))
			p3 := newTestProduct(t, "list_p3", base.Add(-2*time.Hour))

			// first=2 なので limit+1=3 件で問い合わせ、3 件返れば次ページありと判定する。
			repo.EXPECT().
				FindPublishedList(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, params domainproduct.ListParams) (domainproduct.Products, error) {
					assert.Equal(t, int32(3), params.Limit)
					return domainproduct.Products{p1, p2, p3}, nil
				})

			u := &usecase{tracer: lt, repo: repo}
			actual, err := u.ListProducts(ctx, ListProductsParams{Cursor: newDefaultCursor(t)})
			require.NoError(t, err)
			require.Len(t, actual.Items, 2)
			assert.Equal(t, p1.ID(), actual.Items[0].ID)
			assert.Equal(t, p2.ID(), actual.Items[1].ID)

			require.NotNil(t, actual.NextCursor)
			cursor, cErr := paging.NewCursor(actual.NextCursor, nil)
			require.NoError(t, cErr)
			decoded, dErr := decodeProductCursor(cursor)
			require.NoError(t, dErr)
			// NextCursor は切り詰め後の末尾行(p2)を指す。
			assert.Equal(t, p2.PublishedAt(), decoded.publishedAt)
			assert.Equal(t, p2.ID(), decoded.id)
		})

		t.Run("取得件数がlimit以下の場合、NextCursorはnilになる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_product.NewMockRepository(gomock.NewController(t))

			p1 := newTestProduct(t, "single_p1", base)
			repo.EXPECT().FindPublishedList(gomock.Any(), gomock.Any()).Return(domainproduct.Products{p1}, nil)

			u := &usecase{tracer: lt, repo: repo}
			actual, err := u.ListProducts(ctx, ListProductsParams{Cursor: newDefaultCursor(t)})
			require.NoError(t, err)
			require.Len(t, actual.Items, 1)
			assert.Nil(t, actual.NextCursor)
		})

		t.Run("取得結果が0件の場合、空のItemsとnilのNextCursorを返す", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_product.NewMockRepository(gomock.NewController(t))

			repo.EXPECT().FindPublishedList(gomock.Any(), gomock.Any()).Return(domainproduct.Products{}, nil)

			u := &usecase{tracer: lt, repo: repo}
			actual, err := u.ListProducts(ctx, ListProductsParams{Cursor: newDefaultCursor(t)})
			require.NoError(t, err)
			assert.Empty(t, actual.Items)
			assert.Nil(t, actual.NextCursor)
		})

		t.Run("フィルタ・並び順・カーソル境界がドメインのListParamsへ引き渡される", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_product.NewMockRepository(gomock.NewController(t))

			categoryID := uuid.NewTestFromSalt(t, "filter_category")
			statusID := uuid.NewTestFromSalt(t, "filter_status")
			keyword := "イヤホン"

			afterID := uuid.NewTestFromSalt(t, "after_id")
			afterAt := base.Add(-3 * time.Hour)
			after := paging.EncodeCursor(afterAt.Format(time.RFC3339Nano), afterID.String())

			var captured domainproduct.ListParams
			repo.EXPECT().
				FindPublishedList(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, params domainproduct.ListParams) (domainproduct.Products, error) {
					captured = params
					return domainproduct.Products{}, nil
				})

			cursor, err := paging.NewCursor(&after, nil)
			require.NoError(t, err)

			u := &usecase{tracer: lt, repo: repo}
			_, err = u.ListProducts(ctx, ListProductsParams{
				Cursor:     cursor,
				CategoryID: &categoryID,
				StatusID:   &statusID,
				Keyword:    &keyword,
				Ascending:  true,
			})
			require.NoError(t, err)

			assert.True(t, captured.Ascending)
			require.NotNil(t, captured.CategoryID)
			assert.Equal(t, categoryID, *captured.CategoryID)
			require.NotNil(t, captured.StatusID)
			assert.Equal(t, statusID, *captured.StatusID)
			require.NotNil(t, captured.Keyword)
			assert.Equal(t, keyword, *captured.Keyword)
			require.NotNil(t, captured.AfterPublishedAt)
			assert.Equal(t, afterAt, *captured.AfterPublishedAt)
			require.NotNil(t, captured.AfterID)
			assert.Equal(t, afterID, *captured.AfterID)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("cursorがnilの場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_product.NewMockRepository(gomock.NewController(t))

			u := &usecase{tracer: lt, repo: repo}
			actual, err := u.ListProducts(context.Background(), ListProductsParams{Cursor: nil})
			require.Nil(t, actual)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("カーソルの復号に失敗した場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_product.NewMockRepository(gomock.NewController(t))

			// キーが1つだけの不正カーソル。
			bad := paging.EncodeCursor("only-one-key")
			cursor, err := paging.NewCursor(&bad, nil)
			require.NoError(t, err)

			u := &usecase{tracer: lt, repo: repo}
			actual, err := u.ListProducts(context.Background(), ListProductsParams{Cursor: cursor})
			require.Nil(t, actual)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("リポジトリのエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_product.NewMockRepository(gomock.NewController(t))

			expectedErr := testkit.ExpectedDBError()
			repo.EXPECT().FindPublishedList(gomock.Any(), gomock.Any()).Return(nil, expectedErr)

			u := &usecase{tracer: lt, repo: repo}
			actual, err := u.ListProducts(context.Background(), ListProductsParams{Cursor: newDefaultCursor(t)})
			require.ErrorIs(t, err, expectedErr)
			assert.Nil(t, actual)
		})
	})
}
