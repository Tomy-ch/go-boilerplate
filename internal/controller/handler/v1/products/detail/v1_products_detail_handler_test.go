package detail

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/handler/v1/products/detail/gen"
	"go-boilerplate/internal/observability"
	productuc "go-boilerplate/internal/usecase/product"
	mock_product "go-boilerplate/internal/usecase/product/mock"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/products/:productId"

func newServer(t *testing.T) (*server, *mock_product.MockUsecase) {
	t.Helper()
	mockApp := mock_product.NewMockUsecase(gomock.NewController(t))
	return &server{tracer: observability.NewMockControllerLayerTracer(t), uc: mockApp}, mockApp
}

func newProductView(t *testing.T, salt string) productuc.ProductView {
	t.Helper()
	return productuc.ProductView{
		ID:                    uuid.NewTestFromSalt(t, salt),
		Name:                  "商品-" + salt,
		Description:           ptr.To("説明-" + salt),
		Price:                 1999,
		Quantity:              100,
		StockWarningThreshold: ptr.To(10),
		StatusID:              uuid.NewTestFromSalt(t, salt+"_status"),
		CategoryID:            uuid.NewTestFromSalt(t, salt+"_category"),
		PublishedAt:           time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
	}
}

// wantProductResponse は、本番 toProductResponse とは独立な検証用オラクル（フィールド取り違え検出）。
func wantProductResponse(dto productuc.ProductView) gen.ProductResponse {
	//nolint:gosec // G115: テストデータは int32 範囲内の固定値です
	price, quantity := int32(dto.Price), int32(dto.Quantity)
	var threshold *int32
	if dto.StockWarningThreshold != nil {
		//nolint:gosec // G115: テストデータは int32 範囲内の固定値です
		v := int32(*dto.StockWarningThreshold)
		threshold = &v
	}
	return gen.ProductResponse{
		Id:                    dto.ID.ToPrimitive(),
		Name:                  dto.Name,
		Description:           dto.Description,
		Price:                 price,
		Quantity:              quantity,
		StockWarningThreshold: threshold,
		StatusId:              dto.StatusID.ToPrimitive(),
		CategoryId:            dto.CategoryID.ToPrimitive(),
		PublishedAt:           dto.PublishedAt,
	}
}

func Test_intPtrToInt32Ptr(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilの場合はnilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, intPtrToInt32Ptr(nil))
		})

		t.Run("非nilの場合は値を保持したint32ポインタを返す", func(t *testing.T) {
			t.Parallel()
			got := intPtrToInt32Ptr(ptr.To(42))
			require.NotNil(t, got)
			assert.Equal(t, int32(42), *got)
		})
	})
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	mockApp := mock_product.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, mockApp)

	routes := e.Routes()
	require.Len(t, routes, 1)
	assert.Equal(t, http.MethodGet, routes[0].Method)
	assert.Equal(t, targetPath, routes[0].Path)
}

func Test_server_GetProductsDetail(t *testing.T) {
	t.Parallel()

	dto := newProductView(t, "handler_detail")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("公開商品が存在する場合_詳細のProductResponseが返る", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)
			mockApp.EXPECT().GetProduct(gomock.Any(), gomock.Any()).Return(dto, nil)

			resp, err := s.GetProductsDetail(
				context.Background(),
				gen.GetProductsDetailRequestObject{ProductId: dto.ID.ToPrimitive()},
			)
			require.NoError(t, err)

			actual, ok := resp.(gen.GetProductsDetail200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, wantProductResponse(dto), gen.ProductResponse(actual))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Usecaseが未存在・非公開でNotFoundを返す場合_そのまま伝播する", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)
			mockApp.EXPECT().GetProduct(gomock.Any(), gomock.Any()).Return(productuc.ProductView{}, apperror.ErrNotFound)

			resp, err := s.GetProductsDetail(
				context.Background(),
				gen.GetProductsDetailRequestObject{ProductId: dto.ID.ToPrimitive()},
			)
			require.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrNotFound)
		})
	})
}
