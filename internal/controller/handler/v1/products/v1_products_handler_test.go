package products

import (
	"context"
	"math"
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
	"go-boilerplate/pkg/safecast"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
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
		ID:                    uuidtestkit.NewTestFromSalt(t, salt),
		Name:                  "商品-" + salt,
		Description:           ptr.To("説明-" + salt),
		Price:                 decimaltestkit.MustParse(t, "19.99"),
		Quantity:              100,
		StockWarningThreshold: ptr.To(10),
		StatusID:              uuidtestkit.NewTestFromSalt(t, salt+"_status"),
		StatusName:            "在庫あり",
		CategoryID:            uuidtestkit.NewTestFromSalt(t, salt+"_category"),
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

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	mockApp := mock_product.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, mockApp)

	routes := e.Router().Routes()
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

			after := paging.EncodeCursor("2026-01-01T00:00:00Z", uuidtestkit.NewTestFromSalt(t, "handler_after").String())
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

			categoryID := uuidtestkit.NewTestFromSalt(t, "handler_filter_category")
			statusID := uuidtestkit.NewTestFromSalt(t, "handler_filter_status")
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

func Test_isAscending(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("sortがnilの場合はfalse(降順)を返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isAscending(nil))
		})

		t.Run("sortがpublishedAtの場合はtrue(昇順)を返す", func(t *testing.T) {
			t.Parallel()
			sort := gen.GetProductsParamsSortPublishedAt
			assert.True(t, isAscending(&sort))
		})

		t.Run("sortが-publishedAtの場合はfalse(降順)を返す", func(t *testing.T) {
			t.Parallel()
			sort := gen.GetProductsParamsSortMinusPublishedAt
			assert.False(t, isAscending(&sort))
		})
	})
}

func Test_toProductResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全項目が設定されたDTOをレスポンスへ写像する", func(t *testing.T) {
			t.Parallel()
			dto := newProductView(t, "conv_full")

			actual, err := toProductResponse(dto)
			require.NoError(t, err)
			assert.Equal(t, wantProductResponse(dto), actual)
		})

		t.Run("任意項目がnilのDTOはレスポンスでもnilのまま写像する", func(t *testing.T) {
			t.Parallel()
			dto := newProductView(t, "conv_nil")
			dto.Description = nil
			dto.StockWarningThreshold = nil
			dto.PublishedAt = nil
			dto.ImagePath = nil

			actual, err := toProductResponse(dto)
			require.NoError(t, err)
			assert.Nil(t, actual.Description)
			assert.Nil(t, actual.StockWarningThreshold)
			assert.Nil(t, actual.PublishedAt)
			assert.Nil(t, actual.ImagePath)
			assert.Equal(t, dto.ID.ToPrimitive(), actual.Id)
		})

		t.Run("int32の境界値も値を保ったまま写像する", func(t *testing.T) {
			t.Parallel()
			dto := newProductView(t, "conv_boundary")
			dto.Quantity = math.MaxInt32
			dto.Version = math.MaxInt32
			dto.StockWarningThreshold = ptr.To(math.MaxInt32)

			actual, err := toProductResponse(dto)
			require.NoError(t, err)
			assert.Equal(t, int32(math.MaxInt32), actual.Quantity)
			assert.Equal(t, int32(math.MaxInt32), actual.Version)
			require.NotNil(t, actual.StockWarningThreshold)
			assert.Equal(t, int32(math.MaxInt32), *actual.StockWarningThreshold)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("在庫数がint32の範囲を超える場合、オーバーフローとして返す", func(t *testing.T) {
			t.Parallel()
			dto := newProductView(t, "conv_over_quantity")
			dto.Quantity = math.MaxInt32 + 1

			_, err := toProductResponse(dto)
			require.ErrorIs(t, err, safecast.ErrOverflow)
		})

		t.Run("在庫警告閾値がint32の範囲を超える場合、オーバーフローとして返す", func(t *testing.T) {
			t.Parallel()
			dto := newProductView(t, "conv_over_threshold")
			dto.StockWarningThreshold = ptr.To(math.MaxInt32 + 1)

			_, err := toProductResponse(dto)
			require.ErrorIs(t, err, safecast.ErrOverflow)
		})

		t.Run("バージョンがint32の範囲を超える場合、オーバーフローとして返す", func(t *testing.T) {
			t.Parallel()
			dto := newProductView(t, "conv_over_version")
			dto.Version = math.MaxInt32 + 1

			_, err := toProductResponse(dto)
			require.ErrorIs(t, err, safecast.ErrOverflow)
		})
	})
}
