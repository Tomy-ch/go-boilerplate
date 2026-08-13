package productcount

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/controller/handler/v1/products/count/gen"
	"go-boilerplate/internal/observability"
	productuc "go-boilerplate/internal/usecase/product"
	mock_product "go-boilerplate/internal/usecase/product/mock"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/products/count"

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	BindHandler(e, observability.NewNoopTracerFactory(t), mock_product.NewMockUsecase(gomock.NewController(t)))

	testassert.AssertEchoRouterPath(t, targetPath, e.Router().Routes())
	testassert.AssertEchoRouterMethods(t, []string{http.MethodGet}, e.Router().Routes())
}

func Test_server_GetProductsCount(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("検索条件をUsecaseへ渡し件数レスポンスを返す", func(t *testing.T) {
			t.Parallel()

			mockUC := mock_product.NewMockUsecase(gomock.NewController(t))
			categoryID, err := uuid.Parse("5dd52d84-78eb-4a52-ba0b-2e11c95c2af2")
			require.NoError(t, err)
			statusID, err := uuid.Parse("093170fb-83a2-4864-a2b3-53236eaf3597")
			require.NoError(t, err)
			mockUC.EXPECT().CountProducts(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params productuc.CountProductsParams) (productuc.ProductCountView, error) {
					require.NotNil(t, params.CategoryID)
					assert.Equal(t, categoryID, *params.CategoryID)
					require.NotNil(t, params.StatusID)
					assert.Equal(t, statusID, *params.StatusID)
					require.NotNil(t, params.Keyword)
					assert.Equal(t, "phone", *params.Keyword)
					require.NotNil(t, params.MinPrice)
					assert.Equal(t, "10.50", *params.MinPrice)
					require.NotNil(t, params.MaxPrice)
					assert.Equal(t, "99.99", *params.MaxPrice)
					require.NotNil(t, params.MinQuantity)
					assert.Equal(t, int32(2), *params.MinQuantity)
					require.NotNil(t, params.MaxQuantity)
					assert.Equal(t, int32(20), *params.MaxQuantity)
					return productuc.ProductCountView{Count: 4}, nil
				})
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: mockUC}

			resp, err := s.GetProductsCount(context.Background(), gen.GetProductsCountRequestObject{
				Params: gen.GetProductsCountParams{
					CategoryId: ptr.To(categoryID.ToPrimitive()), StatusId: ptr.To(statusID.ToPrimitive()),
					Keyword:  ptr.To("phone"),
					MinPrice: ptr.To("10.50"), MaxPrice: ptr.To("99.99"),
					MinQuantity: ptr.To[int32](2), MaxQuantity: ptr.To[int32](20),
				},
			})

			require.NoError(t, err)
			actual, ok := resp.(gen.GetProductsCount200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, int64(4), actual.Count)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Usecaseのエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			mockUC := mock_product.NewMockUsecase(gomock.NewController(t))
			mockUC.EXPECT().CountProducts(gomock.Any(), gomock.Any()).Return(productuc.ProductCountView{}, apperror.ErrInternal)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: mockUC}

			resp, err := s.GetProductsCount(context.Background(), gen.GetProductsCountRequestObject{})

			require.ErrorIs(t, err, apperror.ErrInternal)
			assert.Nil(t, resp)
		})
	})
}
