package products

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/handler/v1/products/gen"
	"go-boilerplate/internal/usecase/boundary/auth"
	productuc "go-boilerplate/internal/usecase/product"
	"go-boilerplate/pkg/ptr"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newPostProductsRequest は、商品作成の gen リクエストを生成します。
func newPostProductsRequest(t *testing.T) gen.PostProductsRequestObject {
	t.Helper()
	return gen.PostProductsRequestObject{
		Body: &gen.ProductsPostRequest{
			Name:                  "ワイヤレスイヤホン",
			Description:           ptr.To("<p>ノイズキャンセリング対応</p>"),
			Price:                 "19.99",
			Quantity:              100,
			StockWarningThreshold: ptr.To(int32(10)),
			CategoryId:            uuidtestkit.NewTestFromSalt(t, "post_category").ToPrimitive(),
			StatusId:              uuidtestkit.NewTestFromSalt(t, "post_status").ToPrimitive(),
			PublishedAt:           ptr.To(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)),
			ImagePath:             ptr.To("products/earphone.png"),
		},
	}
}

func Test_server_PostProducts(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みで商品を作成し201と作成結果を返す", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)

			view := newProductView(t, "post_created")
			mockApp.EXPECT().
				CreateProduct(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(productuc.CreateProductParams{})).
				DoAndReturn(func(_ context.Context, _ *auth.Authn, p productuc.CreateProductParams) (productuc.ProductView, error) {
					assert.Equal(t, "ワイヤレスイヤホン", p.Name)
					assert.Equal(t, "19.99", p.Price)
					assert.Equal(t, 100, p.Quantity)
					require.NotNil(t, p.ImagePath)
					assert.Equal(t, "products/earphone.png", *p.ImagePath)
					require.NotNil(t, p.Description)
					assert.Equal(t, "<p>ノイズキャンセリング対応</p>", *p.Description)
					require.NotNil(t, p.PublishedAt)
					assert.Equal(t, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC), *p.PublishedAt)
					return view, nil
				})

			resp, err := s.PostProducts(authnContext(t), newPostProductsRequest(t))
			require.NoError(t, err)

			actual, ok := resp.(gen.PostProducts201JSONResponse)
			require.True(t, ok)
			assert.Equal(t, view.Name, actual.Name)
			require.NotNil(t, actual.ImagePath)
			assert.Equal(t, *view.ImagePath, *actual.ImagePath)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("authnが無い場合は認証エラーを返す", func(t *testing.T) {
			t.Parallel()
			s, _ := newServer(t)

			resp, err := s.PostProducts(context.Background(), newPostProductsRequest(t))
			assert.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
		})

		t.Run("Usecaseがエラーを返した場合はエラーを返す", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)

			mockApp.EXPECT().
				CreateProduct(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductView{}, apperror.ErrValidation)

			resp, err := s.PostProducts(authnContext(t), newPostProductsRequest(t))
			assert.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrValidation)
		})
	})
}
