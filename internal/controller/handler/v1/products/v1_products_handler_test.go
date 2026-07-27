package products

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/handler/v1/products/gen"
	"go-boilerplate/internal/observability"
	productuc "go-boilerplate/internal/usecase/product"
	mock_product "go-boilerplate/internal/usecase/product/mock"
	"go-boilerplate/internal/usecase/tools/paging"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/products"

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
		Price:                 decimaltestkit.MustParse(t, "19.99"),
		Quantity:              100,
		StockWarningThreshold: ptr.To(10),
		StatusID:              uuid.NewTestFromSalt(t, salt+"_status"),
		StatusName:            "在庫あり",
		CategoryID:            uuid.NewTestFromSalt(t, salt+"_category"),
		CategoryName:          "電子機器",
		PublishedAt:           ptr.To(time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)),
		ImagePath:             ptr.To("products/" + salt + ".png"),
		Version:               3,
	}
}

// wantProductResponse は、本番 toProductResponse とは独立な検証用オラクル（フィールド取り違え検出）。
func wantProductResponse(dto productuc.ProductView) gen.ProductResponse {
	//nolint:gosec // G115: テストデータは int32 範囲内の固定値です
	quantity := int32(dto.Quantity)
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
		Price:                 dto.Price.String(),
		Quantity:              quantity,
		StockWarningThreshold: threshold,
		Status: gen.ProductStatusRef{
			Id:   dto.StatusID.ToPrimitive(),
			Name: dto.StatusName,
		},
		Category: gen.ProductCategoryRef{
			Id:   dto.CategoryID.ToPrimitive(),
			Name: dto.CategoryName,
		},
		PublishedAt: dto.PublishedAt,
		ImagePath:   dto.ImagePath,
		//nolint:gosec // G115: テストデータは int32 範囲内の固定値です
		Version: int32(dto.Version),
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
	require.Len(t, routes, 3)
	registered := make(map[string]bool, len(routes))
	for _, r := range routes {
		registered[r.Method+" "+r.Path] = true
	}
	assert.True(t, registered[http.MethodGet+" "+targetPath])
	assert.True(t, registered[http.MethodPost+" "+targetPath])
	assert.True(t, registered[http.MethodPost+" "+targetPath+"/images"])
}

func Test_server_GetProducts(t *testing.T) {
	t.Parallel()

	dto1 := newProductView(t, "handler_p1")
	dto2 := newProductView(t, "handler_p2")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("次ページが無い場合、NextCursorがnilでHasNextがfalseのレスポンスが返る", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)

			mockApp.EXPECT().
				ListProducts(gomock.Any(), gomock.AssignableToTypeOf(productuc.ListProductsParams{})).
				Return(productuc.ProductListView{Items: []productuc.ProductView{dto1, dto2}, NextCursor: nil}, nil)

			resp, err := s.GetProducts(context.Background(), gen.GetProductsRequestObject{Params: gen.GetProductsParams{}})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetProducts200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, gen.ProductListResponse{
				Products:   []gen.ProductResponse{wantProductResponse(dto1), wantProductResponse(dto2)},
				NextCursor: nil,
				HasNext:    false,
			}, gen.ProductListResponse(actual))
		})

		t.Run("次ページがある場合、NextCursorが設定されHasNextがtrueのレスポンスが返る", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)

			after := paging.EncodeCursor("2026-01-01T00:00:00Z", uuid.NewTestFromSalt(t, "handler_after").String())
			first := 20
			nextCursor := "next-opaque-cursor"

			mockApp.EXPECT().
				ListProducts(gomock.Any(), gomock.AssignableToTypeOf(productuc.ListProductsParams{})).
				Return(productuc.ProductListView{Items: []productuc.ProductView{dto1}, NextCursor: &nextCursor}, nil)

			resp, err := s.GetProducts(context.Background(), gen.GetProductsRequestObject{
				Params: gen.GetProductsParams{After: &after, First: &first},
			})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetProducts200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, gen.ProductListResponse{
				Products:   []gen.ProductResponse{wantProductResponse(dto1)},
				NextCursor: &nextCursor,
				HasNext:    true,
			}, gen.ProductListResponse(actual))
		})

		t.Run("sortとフィルタがUsecaseのパラメータへ引き渡される", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)

			categoryID := uuid.NewTestFromSalt(t, "handler_filter_category")
			statusID := uuid.NewTestFromSalt(t, "handler_filter_status")
			keyword := "イヤホン"
			cid := categoryID.ToPrimitive()
			sid := statusID.ToPrimitive()
			sort := gen.GetProductsParamsSortPublishedAt

			var captured productuc.ListProductsParams
			mockApp.EXPECT().
				ListProducts(gomock.Any(), gomock.AssignableToTypeOf(productuc.ListProductsParams{})).
				DoAndReturn(func(_ context.Context, params productuc.ListProductsParams) (productuc.ProductListView, error) {
					captured = params
					return productuc.ProductListView{Items: []productuc.ProductView{}, NextCursor: nil}, nil
				})

			_, err := s.GetProducts(context.Background(), gen.GetProductsRequestObject{
				Params: gen.GetProductsParams{CategoryId: &cid, StatusId: &sid, Keyword: &keyword, Sort: &sort},
			})
			require.NoError(t, err)

			assert.True(t, captured.Ascending)
			require.NotNil(t, captured.CategoryID)
			assert.Equal(t, categoryID, *captured.CategoryID)
			require.NotNil(t, captured.StatusID)
			assert.Equal(t, statusID, *captured.StatusID)
			require.NotNil(t, captured.Keyword)
			assert.Equal(t, keyword, *captured.Keyword)
		})

		t.Run("sort未指定の場合、Ascendingはfalse(降順)になる", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)

			var captured productuc.ListProductsParams
			mockApp.EXPECT().
				ListProducts(gomock.Any(), gomock.AssignableToTypeOf(productuc.ListProductsParams{})).
				DoAndReturn(func(_ context.Context, params productuc.ListProductsParams) (productuc.ProductListView, error) {
					captured = params
					return productuc.ProductListView{Items: []productuc.ProductView{}, NextCursor: nil}, nil
				})

			_, err := s.GetProducts(context.Background(), gen.GetProductsRequestObject{Params: gen.GetProductsParams{}})
			require.NoError(t, err)
			assert.False(t, captured.Ascending)
			assert.Nil(t, captured.CategoryID)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("カーソル生成が失敗した場合、ErrInvalidArgumentが返る", func(t *testing.T) {
			t.Parallel()
			s, _ := newServer(t)

			invalidAfter := "!!!not-base64!!!"
			resp, err := s.GetProducts(context.Background(), gen.GetProductsRequestObject{
				Params: gen.GetProductsParams{After: &invalidAfter},
			})
			assert.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("Usecaseがエラーを返した場合、エラーが返る", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)

			expectedErr := apperror.ErrInternal
			mockApp.EXPECT().
				ListProducts(gomock.Any(), gomock.AssignableToTypeOf(productuc.ListProductsParams{})).
				Return(productuc.ProductListView{}, expectedErr)

			resp, err := s.GetProducts(context.Background(), gen.GetProductsRequestObject{Params: gen.GetProductsParams{}})
			assert.Nil(t, resp)
			require.ErrorIs(t, err, expectedErr)
		})
	})
}
