package productcategory

import (
	"context"
	"testing"

	domainproductcategory "go-boilerplate/internal/domain/product_category"
	mock_product_category "go-boilerplate/internal/domain/product_category/mock"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/testkit"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		tf := observability.NewNoopTracerFactory(t)
		repo := mock_product_category.NewMockRepository(ctrl)

		expected := &usecase{
			tracer: tf.Usecase(),
			repo:   repo,
		}
		actual := New(repo, tf)

		assert.Equal(t, expected, actual)
	})
}

func Test_usecase_ListProductCategories(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("取得した商品カテゴリエンティティをsortKey昇順のDTOへ写像して返す", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)

			electronicsID, err := uuid.Parse("5dd52d84-78eb-4a52-ba0b-2e11c95c2af2")
			require.NoError(t, err)
			booksID, err := uuid.Parse("b39be992-fe5a-4b4c-9f98-e695f0f5101e")
			require.NoError(t, err)

			electronics, err := domainproductcategory.New(electronicsID, "電子機器", 1, 1)
			require.NoError(t, err)
			books, err := domainproductcategory.New(booksID, "書籍", 2, 2)
			require.NoError(t, err)

			repo := mock_product_category.NewMockRepository(ctrl)
			repo.EXPECT().FindAll(gomock.Any()).Return(domainproductcategory.ProductCategories{electronics, books}, nil).Times(1)

			u := &usecase{tracer: lt, repo: repo}

			actual, err := u.ListProductCategories(ctx)
			require.NoError(t, err)
			assert.Equal(t, ProductCategoryDTOs{
				{ID: electronicsID, Code: 1, Name: "電子機器", SortKey: 1},
				{ID: booksID, Code: 2, Name: "書籍", SortKey: 2},
			}, actual)
		})

		t.Run("取得結果が0件の場合、nilではない空のDTO一覧を返す", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)

			repo := mock_product_category.NewMockRepository(ctrl)
			repo.EXPECT().FindAll(gomock.Any()).Return(domainproductcategory.ProductCategories{}, nil).Times(1)

			u := &usecase{tracer: lt, repo: repo}

			actual, err := u.ListProductCategories(ctx)
			require.NoError(t, err)
			assert.NotNil(t, actual)
			assert.Empty(t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("リポジトリのエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)

			expectedErr := testkit.ExpectedDBError()

			repo := mock_product_category.NewMockRepository(ctrl)
			repo.EXPECT().FindAll(gomock.Any()).Return(nil, expectedErr).Times(1)

			u := &usecase{tracer: lt, repo: repo}

			actual, err := u.ListProductCategories(ctx)
			require.ErrorIs(t, err, expectedErr)
			assert.Nil(t, actual)
		})
	})
}
