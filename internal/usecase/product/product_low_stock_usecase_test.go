package product

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	domainproduct "go-boilerplate/internal/domain/product"
	mock_product "go-boilerplate/internal/domain/product/mock"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	mock_authz "go-boilerplate/internal/usecase/boundary/authz/mock"
	"go-boilerplate/pkg/ptr"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// lowStockTestDeps は、ListLowStockProducts のテストで注入する依存モック一式です。
type lowStockTestDeps struct {
	repo       *mock_product.MockRepository
	authorizer *mock_authz.MockAuthorizer
}

// newLowStockTestUsecase は、モック依存のみで構成した usecase とそのモック一式を返します。
func newLowStockTestUsecase(t *testing.T) (*usecase, *lowStockTestDeps) {
	t.Helper()

	ctrl := gomock.NewController(t)
	deps := &lowStockTestDeps{
		repo:       mock_product.NewMockRepository(ctrl),
		authorizer: mock_authz.NewMockAuthorizer(ctrl),
	}
	u := &usecase{
		tracer:     observability.NewMockUsecaseLayerTracer(t),
		repo:       deps.repo,
		authorizer: deps.authorizer,
	}

	return u, deps
}

// newLowStockProduct は、在庫僅少一覧の返却要素となる商品エンティティを構築します。
func newLowStockProduct(t *testing.T, salt string, quantity int) *domainproduct.Product {
	t.Helper()

	statusRef, err := domainproduct.NewStatusRef(uuidtestkit.NewTestFromSalt(t, salt+"_status"), "在庫わずか")
	require.NoError(t, err)
	categoryRef, err := domainproduct.NewCategoryRef(uuidtestkit.NewTestFromSalt(t, salt+"_category"), "電子機器")
	require.NoError(t, err)

	p, err := domainproduct.Reconstruct(uuidtestkit.NewTestFromSalt(t, salt), domainproduct.Attributes{
		Name:                  "商品-" + salt,
		Description:           ptr.To("説明-" + salt),
		Price:                 mustPrice(t, "10.5"),
		Quantity:              quantity,
		StockWarningThreshold: ptr.To(10),
		Status:                statusRef,
		Category:              categoryRef,
		PublishedAt:           nil,
		ImagePath:             ptr.To("products/" + salt + ".png"),
	}, 1)
	require.NoError(t, err)

	return p
}

func Test_usecase_ListLowStockProducts(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("リポジトリの返却順を保ったままDTOへ写す", func(t *testing.T) {
			t.Parallel()

			u, deps := newLowStockTestUsecase(t)
			first := newLowStockProduct(t, "lowstock_uc_first", 0)
			second := newLowStockProduct(t, "lowstock_uc_second", 5)

			deps.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			deps.repo.EXPECT().FindAllLowStock(gomock.Any(), gomock.Any()).
				Return(domainproduct.Products{first, second}, nil)

			actual, err := u.ListLowStockProducts(context.Background(), &auth.Authn{}, ListLowStockProductsParams{Limit: 20})
			require.NoError(t, err)
			require.Len(t, actual.Items, 2)
			assert.Equal(t, first.ID(), actual.Items[0].ID)
			assert.Equal(t, 0, actual.Items[0].Quantity)
			assert.Equal(t, second.ID(), actual.Items[1].ID)
			assert.Equal(t, 5, actual.Items[1].Quantity)
			assert.Equal(t, "10.5", actual.Items[0].Price.String())
			require.NotNil(t, actual.Items[0].StockWarningThreshold)
			assert.Equal(t, 10, *actual.Items[0].StockWarningThreshold)
		})

		t.Run("在庫僅少一覧の参照を所有者なしリソースとして認可する", func(t *testing.T) {
			t.Parallel()

			u, deps := newLowStockTestUsecase(t)
			var capturedAction authz.Action
			var capturedResource *authz.Resource
			deps.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *auth.Authn, action authz.Action, resource *authz.Resource) error {
					capturedAction = action
					capturedResource = resource
					return nil
				})
			deps.repo.EXPECT().FindAllLowStock(gomock.Any(), gomock.Any()).Return(domainproduct.Products{}, nil)

			_, err := u.ListLowStockProducts(context.Background(), &auth.Authn{}, ListLowStockProductsParams{})
			require.NoError(t, err)
			assert.Equal(t, authz.ActionProductListLowStock, capturedAction)
			require.NotNil(t, capturedResource)
			assert.Equal(t, "product", capturedResource.Kind())
			assert.Nil(t, capturedResource.OwnerID())
		})

		t.Run("limit未指定の場合、既定件数がリポジトリへ渡る", func(t *testing.T) {
			t.Parallel()

			u, deps := newLowStockTestUsecase(t)
			deps.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			var capturedLimit int32
			deps.repo.EXPECT().FindAllLowStock(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, limit int32) (domainproduct.Products, error) {
					capturedLimit = limit
					return domainproduct.Products{}, nil
				})

			_, err := u.ListLowStockProducts(context.Background(), &auth.Authn{}, ListLowStockProductsParams{Limit: 0})
			require.NoError(t, err)
			assert.Equal(t, int32(lowStockDefaultLimit), capturedLimit)
		})

		t.Run("limitが上限超過の場合、上限へクランプした件数がリポジトリへ渡る", func(t *testing.T) {
			t.Parallel()

			u, deps := newLowStockTestUsecase(t)
			deps.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			var capturedLimit int32
			deps.repo.EXPECT().FindAllLowStock(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, limit int32) (domainproduct.Products, error) {
					capturedLimit = limit
					return domainproduct.Products{}, nil
				})

			_, err := u.ListLowStockProducts(context.Background(), &auth.Authn{}, ListLowStockProductsParams{Limit: 1000})
			require.NoError(t, err)
			assert.Equal(t, int32(lowStockMaxLimit), capturedLimit)
		})

		t.Run("対象商品が無い場合、空の一覧を返す", func(t *testing.T) {
			t.Parallel()

			u, deps := newLowStockTestUsecase(t)
			deps.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			deps.repo.EXPECT().FindAllLowStock(gomock.Any(), gomock.Any()).Return(domainproduct.Products{}, nil)

			actual, err := u.ListLowStockProducts(context.Background(), &auth.Authn{}, ListLowStockProductsParams{})
			require.NoError(t, err)
			assert.Empty(t, actual.Items)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証情報がnilの場合、ErrUnauthenticatedを返し認可もリポジトリも呼ばない", func(t *testing.T) {
			t.Parallel()

			u, deps := newLowStockTestUsecase(t)
			deps.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			deps.repo.EXPECT().FindAllLowStock(gomock.Any(), gomock.Any()).Times(0)

			actual, err := u.ListLowStockProducts(context.Background(), nil, ListLowStockProductsParams{})
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
			assert.Empty(t, actual.Items)
		})

		t.Run("認可が拒否された場合、ErrForbiddenを伝播しリポジトリを呼ばない", func(t *testing.T) {
			t.Parallel()

			u, deps := newLowStockTestUsecase(t)
			deps.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(authz.ErrForbidden)
			deps.repo.EXPECT().FindAllLowStock(gomock.Any(), gomock.Any()).Times(0)

			actual, err := u.ListLowStockProducts(context.Background(), &auth.Authn{}, ListLowStockProductsParams{})
			require.ErrorIs(t, err, authz.ErrForbidden)
			assert.Empty(t, actual.Items)
		})

		t.Run("リポジトリがエラーを返した場合、そのまま伝播する", func(t *testing.T) {
			t.Parallel()

			u, deps := newLowStockTestUsecase(t)
			deps.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			deps.repo.EXPECT().FindAllLowStock(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrInternal)

			actual, err := u.ListLowStockProducts(context.Background(), &auth.Authn{}, ListLowStockProductsParams{})
			require.ErrorIs(t, err, apperror.ErrInternal)
			assert.Empty(t, actual.Items)
		})
	})
}
