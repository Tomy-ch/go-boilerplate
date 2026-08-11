package lowstock

import (
	"context"
	"math"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/controller/handler/v1/products/lowstock/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	productuc "go-boilerplate/internal/usecase/product"
	mock_product "go-boilerplate/internal/usecase/product/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/products/low-stock"

func newServer(t *testing.T) (*server, *mock_product.MockUsecase) {
	t.Helper()
	mockUC := mock_product.NewMockUsecase(gomock.NewController(t))
	return &server{tracer: observability.NewMockControllerLayerTracer(t), uc: mockUC}, mockUC
}

// authnContext は、内部ユーザー ID を解決済みの認証コンテキストを返すテストヘルパーです。
func authnContext(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()
	ctx := ctxhelper.WithAuthn(context.Background())
	authn, err := auth.New("subject", "issuer", nil, nil)
	require.NoError(t, err)

	resolved, err := authn.WithUserID(userID)
	require.NoError(t, err)
	require.True(t, ctxhelper.SetAuthn(ctx, *resolved))
	return ctx
}

func newProductView(t *testing.T, salt string) productuc.ProductView {
	t.Helper()
	return productuc.ProductView{
		ID:                    uuidtestkit.NewTestFromSalt(t, salt),
		Name:                  "商品-" + salt,
		Description:           ptr.To("説明-" + salt),
		Price:                 decimaltestkit.MustParse(t, "19.99"),
		Quantity:              3,
		StockWarningThreshold: ptr.To(10),
		StatusID:              uuidtestkit.NewTestFromSalt(t, salt+"_status"),
		StatusName:            "在庫あり",
		CategoryID:            uuidtestkit.NewTestFromSalt(t, salt+"_category"),
		CategoryName:          "電子機器",
		PublishedAt:           ptr.To(time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)),
		Images: []productuc.ProductImageItemView{
			{Path: "products/" + salt + ".png", SortKey: 1},
		},
		Version: 3,
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
		Images:      expectedImageItems(dto.Images),
		//nolint:gosec // G115: テストデータは int32 範囲内の固定値です
		Version: int32(dto.Version),
	}
}

// expectedImageItems は、期待レスポンスの商品画像を組み立てます。
// production の写像と同じ順序・同じ値になることを固定するため、要素は素直に書き下します。
func expectedImageItems(dtos []productuc.ProductImageItemView) []gen.ProductImageItem {
	items := make([]gen.ProductImageItem, len(dtos))
	for i, dto := range dtos {
		//nolint:gosec // G115: テストデータは int32 範囲内の固定値です
		items[i] = gen.ProductImageItem{ImagePath: dto.Path, SortKey: int32(dto.SortKey)}
	}
	return items
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	mockUC := mock_product.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, mockUC)

	testassert.AssertEchoRouterPath(t, targetPath, e.Router().Routes())
	testassert.AssertEchoRouterMethods(t, []string{http.MethodGet}, e.Router().Routes())
}

func Test_server_GetProductsLowStock(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ユースケースの一覧を順序どおりProductLowStockResponseへ写像する", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			low := newProductView(t, "lowstock_h1")
			lower := newProductView(t, "lowstock_h2")
			lower.Quantity = 0
			mockUC.EXPECT().ListLowStockProducts(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductLowStockListView{Items: []productuc.ProductView{lower, low}}, nil)

			resp, err := s.GetProductsLowStock(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "lowstock_h_user")),
				gen.GetProductsLowStockRequestObject{Params: gen.GetProductsLowStockParams{Limit: ptr.To(20)}},
			)
			require.NoError(t, err)

			actual, ok := resp.(gen.GetProductsLowStock200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, gen.ProductLowStockResponse{
				Products: []gen.ProductResponse{wantProductResponse(lower), wantProductResponse(low)},
			}, gen.ProductLowStockResponse(actual))
		})

		t.Run("価格はdecimal文字列で在庫警告閾値はnullable int32として返る", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			dto := newProductView(t, "lowstock_h_price")
			dto.Price = decimaltestkit.MustParse(t, "0.125")
			dto.StockWarningThreshold = ptr.To(5)
			mockUC.EXPECT().ListLowStockProducts(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductLowStockListView{Items: []productuc.ProductView{dto}}, nil)

			resp, err := s.GetProductsLowStock(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "lowstock_h_price_user")),
				gen.GetProductsLowStockRequestObject{},
			)
			require.NoError(t, err)

			actual, ok := resp.(gen.GetProductsLowStock200JSONResponse)
			require.True(t, ok)
			require.Len(t, actual.Products, 1)
			assert.Equal(t, "0.125", actual.Products[0].Price)
			require.NotNil(t, actual.Products[0].StockWarningThreshold)
			assert.Equal(t, int32(5), *actual.Products[0].StockWarningThreshold)
		})

		t.Run("limitとAuthnがユースケースへ引き渡される", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			userID := uuidtestkit.NewTestFromSalt(t, "lowstock_h_limit_user")

			var captured productuc.ListLowStockProductsParams
			mockUC.EXPECT().ListLowStockProducts(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(
					_ context.Context, authn *auth.Authn, params productuc.ListLowStockProductsParams,
				) (productuc.ProductLowStockListView, error) {
					captured = params
					uid, uerr := authn.UserID()
					require.NoError(t, uerr)
					assert.Equal(t, userID, uid)
					return productuc.ProductLowStockListView{}, nil
				})

			_, err := s.GetProductsLowStock(
				authnContext(t, userID),
				gen.GetProductsLowStockRequestObject{Params: gen.GetProductsLowStockParams{Limit: ptr.To(7)}},
			)
			require.NoError(t, err)
			assert.Equal(t, 7, captured.Limit)
		})

		t.Run("limit未指定の場合、既定件数を意味する0がユースケースへ渡る", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)

			var captured productuc.ListLowStockProductsParams
			mockUC.EXPECT().ListLowStockProducts(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(
					_ context.Context, _ *auth.Authn, params productuc.ListLowStockProductsParams,
				) (productuc.ProductLowStockListView, error) {
					captured = params
					return productuc.ProductLowStockListView{}, nil
				})

			_, err := s.GetProductsLowStock(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "lowstock_h_nolimit_user")),
				gen.GetProductsLowStockRequestObject{Params: gen.GetProductsLowStockParams{}},
			)
			require.NoError(t, err)
			assert.Equal(t, 0, captured.Limit)
		})

		t.Run("対象商品が無い場合、空配列のレスポンスが返る", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().ListLowStockProducts(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductLowStockListView{}, nil)

			resp, err := s.GetProductsLowStock(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "lowstock_h_empty_user")),
				gen.GetProductsLowStockRequestObject{},
			)
			require.NoError(t, err)

			actual, ok := resp.(gen.GetProductsLowStock200JSONResponse)
			require.True(t, ok)
			assert.Empty(t, actual.Products)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証情報が取得できない場合、ErrUnauthenticatedUserを返しユースケースを呼ばない", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().ListLowStockProducts(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			resp, err := s.GetProductsLowStock(context.Background(), gen.GetProductsLowStockRequestObject{})
			assert.Nil(t, resp)
			require.ErrorIs(t, err, ctxhelper.ErrUnauthenticatedUser)
		})

		t.Run("ユースケースが権限エラーを返した場合、そのまま伝播する", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().ListLowStockProducts(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductLowStockListView{}, apperror.ErrPermissionDenied)

			resp, err := s.GetProductsLowStock(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "lowstock_h_forbidden_user")),
				gen.GetProductsLowStockRequestObject{},
			)
			assert.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})
	})
}

func Test_limitParam(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilの場合は既定件数を意味する0を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 0, limitParam(nil))
		})

		t.Run("非nilの場合は指定値をそのまま返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 42, limitParam(ptr.To(42)))
		})
	})
}

func Test_toProductResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全項目が設定されたDTOをレスポンスへ写像する", func(t *testing.T) {
			t.Parallel()
			dto := newProductView(t, "lowstock_conv_full")

			actual, err := toProductResponse(dto)
			require.NoError(t, err)
			assert.Equal(t, wantProductResponse(dto), actual)
		})

		t.Run("任意項目がnilのDTOはレスポンスでもnilのまま写像する", func(t *testing.T) {
			t.Parallel()
			dto := newProductView(t, "lowstock_conv_nil")
			dto.Description = nil
			dto.StockWarningThreshold = nil
			dto.PublishedAt = nil
			dto.Images = nil

			actual, err := toProductResponse(dto)
			require.NoError(t, err)
			assert.Nil(t, actual.Description)
			assert.Nil(t, actual.StockWarningThreshold)
			assert.Nil(t, actual.PublishedAt)
			assert.Empty(t, actual.Images)
			assert.Equal(t, dto.ID.ToPrimitive(), actual.Id)
		})

		t.Run("int32の境界値も値を保ったまま写像する", func(t *testing.T) {
			t.Parallel()
			dto := newProductView(t, "lowstock_conv_boundary")
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
			dto := newProductView(t, "lowstock_conv_over_quantity")
			dto.Quantity = math.MaxInt32 + 1

			_, err := toProductResponse(dto)
			require.ErrorIs(t, err, safecast.ErrOverflow)
		})

		t.Run("在庫警告閾値がint32の範囲を超える場合、オーバーフローとして返す", func(t *testing.T) {
			t.Parallel()
			dto := newProductView(t, "lowstock_conv_over_threshold")
			dto.StockWarningThreshold = ptr.To(math.MaxInt32 + 1)

			_, err := toProductResponse(dto)
			require.ErrorIs(t, err, safecast.ErrOverflow)
		})

		t.Run("バージョンがint32の範囲を超える場合、オーバーフローとして返す", func(t *testing.T) {
			t.Parallel()
			dto := newProductView(t, "lowstock_conv_over_version")
			dto.Version = math.MaxInt32 + 1

			_, err := toProductResponse(dto)
			require.ErrorIs(t, err, safecast.ErrOverflow)
		})
	})
}

func Test_toProductImageItems(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DTOの並びのままレスポンスへ変換する", func(t *testing.T) {
			t.Parallel()

			actual, err := toProductImageItems([]productuc.ProductImageItemView{
				{Path: "products/a.png", SortKey: 1},
				{Path: "products/b.png", SortKey: 5},
			})

			require.NoError(t, err)
			require.Len(t, actual, 2)
			assert.Equal(t, gen.ProductImageItem{ImagePath: "products/a.png", SortKey: 1}, actual[0])
			assert.Equal(t, gen.ProductImageItem{ImagePath: "products/b.png", SortKey: 5}, actual[1])
		})

		t.Run("画像が空の場合、空を返す", func(t *testing.T) {
			t.Parallel()

			actual, err := toProductImageItems(nil)

			require.NoError(t, err)
			assert.Empty(t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("表示順がint32に収まらない場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := toProductImageItems([]productuc.ProductImageItemView{
				{Path: "products/a.png", SortKey: math.MaxInt32 + 1},
			})

			require.Error(t, err)
			assert.Nil(t, actual)
		})
	})
}
