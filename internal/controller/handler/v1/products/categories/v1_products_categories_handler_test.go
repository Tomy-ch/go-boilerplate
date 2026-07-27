package productcategories

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/controller/handler/v1/products/categories/gen"
	"go-boilerplate/internal/observability"
	categoryuc "go-boilerplate/internal/usecase/product/category"
	mock_category "go-boilerplate/internal/usecase/product/category/mock"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/products/categories"

func newServer(t *testing.T) (*server, *mock_category.MockUsecase) {
	t.Helper()
	mockUC := mock_category.NewMockUsecase(gomock.NewController(t))
	return &server{tracer: observability.NewMockControllerLayerTracer(t), uc: mockUC}, mockUC
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	mockUC := mock_category.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, mockUC)

	testassert.AssertEchoRouterPath(t, targetPath, e.Routes())
	testassert.AssertEchoRouterMethods(t, []string{http.MethodGet}, e.Routes())
}

func Test_server_GetProductCategories(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("usecaseのDTO一覧をsortKey昇順のレスポンスへ詰め替える", func(t *testing.T) {
			t.Parallel()

			electronicsID, err := uuid.Parse("5dd52d84-78eb-4a52-ba0b-2e11c95c2af2")
			require.NoError(t, err)
			booksID, err := uuid.Parse("b39be992-fe5a-4b4c-9f98-e695f0f5101e")
			require.NoError(t, err)

			s, mockUC := newServer(t)
			mockUC.EXPECT().ListCategories(gomock.Any()).Return(categoryuc.CategoryDTOs{
				{ID: electronicsID, Code: 1, Name: "電子機器", SortKey: 1},
				{ID: booksID, Code: 2, Name: "書籍", SortKey: 2},
			}, nil)

			resp, err := s.GetProductCategories(context.Background(), gen.GetProductCategoriesRequestObject{})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetProductCategories200JSONResponse)
			require.True(t, ok)

			assert.Equal(t, gen.GetProductCategories200JSONResponse{
				{Id: electronicsID.ToPrimitive(), Code: 1, Name: "電子機器", SortKey: 1},
				{Id: booksID.ToPrimitive(), Code: 2, Name: "書籍", SortKey: 2},
			}, actual)
		})

		t.Run("空一覧の場合、空のレスポンスを返す", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().ListCategories(gomock.Any()).Return(categoryuc.CategoryDTOs{}, nil)

			resp, err := s.GetProductCategories(context.Background(), gen.GetProductCategoriesRequestObject{})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetProductCategories200JSONResponse)
			require.True(t, ok)
			assert.NotNil(t, actual)
			assert.Empty(t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("usecaseのエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().ListCategories(gomock.Any()).Return(nil, apperror.ErrInternal)

			resp, err := s.GetProductCategories(context.Background(), gen.GetProductCategoriesRequestObject{})
			require.ErrorIs(t, err, apperror.ErrInternal)
			require.Nil(t, resp)
		})
	})
}

func Test_toProductCategoryResponse(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
