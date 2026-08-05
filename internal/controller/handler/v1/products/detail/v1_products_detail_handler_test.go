package detail

import (
	"context"
	"math"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/products/detail/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	productuc "go-boilerplate/internal/usecase/product"
	mock_product "go-boilerplate/internal/usecase/product/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/patch"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	targetPath = "/v1/products/:productId"
	stockPath  = "/v1/products/:productId/stock"
)

// patchPublishedAt は、部分更新リクエストに載せる固定の公開日時です。
var patchPublishedAt = time.Date(2026, time.February, 3, 4, 5, 6, 0, time.UTC)

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
	assert.True(t, registered[http.MethodPatch+" "+targetPath])
	assert.True(t, registered[http.MethodPatch+" "+stockPath])
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
			assert.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrNotFound)
		})
	})
}

// authnContext は、認証済みスロットを仕込んだ context を返します。
func authnContext(t *testing.T) context.Context {
	t.Helper()
	ctx := ctxhelper.WithAuthn(context.Background())
	a, err := auth.New("subject-1", auth.IssuerMock, nil, nil)
	require.NoError(t, err)
	require.True(t, ctxhelper.SetAuthn(ctx, *a))
	return ctx
}

// newPatchProductsDetailRequest は、全フィールドを値指定した商品部分更新の gen リクエストを生成します。
func newPatchProductsDetailRequest(t *testing.T, productID uuid.UUID) gen.PatchProductsDetailRequestObject {
	t.Helper()
	return gen.PatchProductsDetailRequestObject{
		ProductId: productID.ToPrimitive(),
		Body: &gen.ProductPatchRequest{
			Version:               7,
			Name:                  ptr.To("更新後の商品名"),
			Price:                 ptr.To("29.99"),
			Quantity:              ptr.To(int32(50)),
			CategoryId:            ptr.To(uuidtestkit.NewTestFromSalt(t, "patch_category").ToPrimitive()),
			StatusId:              ptr.To(uuidtestkit.NewTestFromSalt(t, "patch_status").ToPrimitive()),
			Description:           nullable.NewNullableWithValue("<p>更新後の説明</p>"),
			StockWarningThreshold: nullable.NewNullableWithValue(int32(5)),
			PublishedAt:           nullable.NewNullableWithValue(patchPublishedAt),
			ImagePath:             nullable.NewNullableWithValue("products/updated.png"),
		},
	}
}

func Test_server_PatchProductsDetail(t *testing.T) {
	t.Parallel()

	targetID := uuidtestkit.NewTestFromSalt(t, "patch_target")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全フィールドが値指定の場合_UpdateProductParamsへ忠実に詰め替えて200と更新結果を返す", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)
			view := newProductView(t, "patch_updated")

			mockApp.EXPECT().
				UpdateProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(productuc.UpdateProductParams{})).
				DoAndReturn(func(_ context.Context, authn *auth.Authn, id uuid.UUID, p productuc.UpdateProductParams) (productuc.ProductView, error) {
					require.NotNil(t, authn)
					assert.Equal(t, "subject-1", authn.Subject())
					assert.Equal(t, targetID, id)

					assert.Equal(t, 7, p.Version)
					require.NotNil(t, p.Name)
					assert.Equal(t, "更新後の商品名", *p.Name)
					require.NotNil(t, p.Price)
					assert.Equal(t, "29.99", *p.Price)
					require.NotNil(t, p.Quantity)
					assert.Equal(t, 50, *p.Quantity)
					require.NotNil(t, p.CategoryID)
					assert.Equal(t, uuidtestkit.NewTestFromSalt(t, "patch_category"), *p.CategoryID)
					require.NotNil(t, p.StatusID)
					assert.Equal(t, uuidtestkit.NewTestFromSalt(t, "patch_status"), *p.StatusID)

					assert.Equal(t, patch.Value("<p>更新後の説明</p>"), p.Description)
					assert.Equal(t, patch.Value(5), p.StockWarningThreshold)
					assert.Equal(t, patch.Value(patchPublishedAt), p.PublishedAt)
					assert.Equal(t, patch.Value("products/updated.png"), p.ImagePath)

					return view, nil
				})

			resp, err := s.PatchProductsDetail(authnContext(t), newPatchProductsDetailRequest(t, targetID))
			require.NoError(t, err)

			actual, ok := resp.(gen.PatchProductsDetail200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, wantProductResponse(view), gen.ProductResponse(actual))
		})

		t.Run("versionのみ指定の場合_他フィールドは未指定のままユースケースへ渡る", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)

			mockApp.EXPECT().
				UpdateProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(productuc.UpdateProductParams{})).
				DoAndReturn(func(_ context.Context, _ *auth.Authn, _ uuid.UUID, p productuc.UpdateProductParams) (productuc.ProductView, error) {
					assert.Equal(t, 7, p.Version)
					assert.Nil(t, p.Name)
					assert.Nil(t, p.Price)
					assert.Nil(t, p.Quantity)
					assert.Nil(t, p.CategoryID)
					assert.Nil(t, p.StatusID)

					assert.Equal(t, patch.Unspecified[string](), p.Description)
					assert.Equal(t, patch.Unspecified[int](), p.StockWarningThreshold)
					assert.Equal(t, patch.Unspecified[time.Time](), p.PublishedAt)
					assert.Equal(t, patch.Unspecified[string](), p.ImagePath)

					return newProductView(t, "patch_unspecified"), nil
				})

			req := gen.PatchProductsDetailRequestObject{
				ProductId: targetID.ToPrimitive(),
				Body:      &gen.ProductPatchRequest{Version: 7},
			}
			_, err := s.PatchProductsDetail(authnContext(t), req)
			require.NoError(t, err)
		})

		t.Run("3状態フィールドがnull指定の場合_クリア指定としてユースケースへ渡る", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)

			mockApp.EXPECT().
				UpdateProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(productuc.UpdateProductParams{})).
				DoAndReturn(func(_ context.Context, _ *auth.Authn, _ uuid.UUID, p productuc.UpdateProductParams) (productuc.ProductView, error) {
					assert.Equal(t, patch.Null[string](), p.Description)
					assert.Equal(t, patch.Null[int](), p.StockWarningThreshold)
					assert.Equal(t, patch.Null[time.Time](), p.PublishedAt)
					assert.Equal(t, patch.Null[string](), p.ImagePath)

					return newProductView(t, "patch_cleared"), nil
				})

			req := gen.PatchProductsDetailRequestObject{
				ProductId: targetID.ToPrimitive(),
				Body: &gen.ProductPatchRequest{
					Version:               7,
					Description:           nullable.NewNullNullable[string](),
					StockWarningThreshold: nullable.NewNullNullable[int32](),
					PublishedAt:           nullable.NewNullNullable[time.Time](),
					ImagePath:             nullable.NewNullNullable[string](),
				},
			}
			_, err := s.PatchProductsDetail(authnContext(t), req)
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("authnが無い場合_認証エラーを返しユースケースを呼ばない", func(t *testing.T) {
			t.Parallel()
			s, _ := newServer(t)

			resp, err := s.PatchProductsDetail(context.Background(), newPatchProductsDetailRequest(t, targetID))
			assert.Nil(t, resp)
			require.ErrorIs(t, err, ctxhelper.ErrUnauthenticatedUser)
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
		})

		t.Run("Usecaseがバージョン競合でConflictを返す場合_そのまま伝播する", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)
			mockApp.EXPECT().
				UpdateProduct(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductView{}, apperror.ErrConflict)

			resp, err := s.PatchProductsDetail(authnContext(t), newPatchProductsDetailRequest(t, targetID))
			assert.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrConflict)
		})
	})
}

func Test_server_PatchProductsStock(t *testing.T) {
	t.Parallel()

	targetID := uuidtestkit.NewTestFromSalt(t, "stock_target")

	newRequest := func(delta int32) gen.PatchProductsStockRequestObject {
		return gen.PatchProductsStockRequestObject{
			ProductId: targetID.ToPrimitive(),
			Body:      &gen.ProductStockPatchRequest{Delta: delta},
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("正のdeltaの場合_UpdateProductStockParamsへ詰め替えて200と更新結果を返す", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)
			view := newProductView(t, "stock_replenished")

			mockApp.EXPECT().
				UpdateProductStock(gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.AssignableToTypeOf(productuc.UpdateProductStockParams{})).
				DoAndReturn(func(
					_ context.Context, authn *auth.Authn, id uuid.UUID, p productuc.UpdateProductStockParams,
				) (productuc.ProductView, error) {
					require.NotNil(t, authn)
					assert.Equal(t, "subject-1", authn.Subject())
					assert.Equal(t, targetID, id)
					assert.Equal(t, 50, p.Delta)

					return view, nil
				})

			resp, err := s.PatchProductsStock(authnContext(t), newRequest(50))
			require.NoError(t, err)

			actual, ok := resp.(gen.PatchProductsStock200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, wantProductResponse(view), gen.ProductResponse(actual))
		})

		t.Run("負のdeltaの場合_符号を保ったままユースケースへ渡る", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)

			mockApp.EXPECT().
				UpdateProductStock(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(
					_ context.Context, _ *auth.Authn, _ uuid.UUID, p productuc.UpdateProductStockParams,
				) (productuc.ProductView, error) {
					assert.Equal(t, -3, p.Delta)
					return newProductView(t, "stock_decreased"), nil
				})

			_, err := s.PatchProductsStock(authnContext(t), newRequest(-3))
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("authnが無い場合_認証エラーを返しユースケースを呼ばない", func(t *testing.T) {
			t.Parallel()
			s, _ := newServer(t)

			resp, err := s.PatchProductsStock(context.Background(), newRequest(1))
			assert.Nil(t, resp)
			require.ErrorIs(t, err, ctxhelper.ErrUnauthenticatedUser)
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
		})

		t.Run("Usecaseが不変条件違反を返す場合_そのまま伝播する", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)
			mockApp.EXPECT().
				UpdateProductStock(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductView{}, apperror.ErrValidation)

			resp, err := s.PatchProductsStock(authnContext(t), newRequest(-100))
			assert.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrValidation)
		})
	})
}

func Test_toPatchField(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未指定の場合は未指定のFieldを返す", func(t *testing.T) {
			t.Parallel()
			var v nullable.Nullable[string]
			assert.Equal(t, patch.Unspecified[string](), toPatchField(v))
		})

		t.Run("null指定の場合はnull指定のFieldを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, patch.Null[string](), toPatchField(nullable.NewNullNullable[string]()))
		})

		t.Run("値指定の場合は値を保持したFieldを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, patch.Value("products/a.png"), toPatchField(nullable.NewNullableWithValue("products/a.png")))
		})
	})
}

func Test_toPatchFieldInt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未指定の場合は未指定のFieldを返す", func(t *testing.T) {
			t.Parallel()
			var v nullable.Nullable[int32]
			assert.Equal(t, patch.Unspecified[int](), toPatchFieldInt(v))
		})

		t.Run("null指定の場合はnull指定のFieldを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, patch.Null[int](), toPatchFieldInt(nullable.NewNullNullable[int32]()))
		})

		t.Run("値指定の場合はintへ変換した値を保持したFieldを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, patch.Value(5), toPatchFieldInt(nullable.NewNullableWithValue(int32(5))))
		})
	})
}

func Test_int32PtrToIntPtr(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilの場合はnilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, int32PtrToIntPtr(nil))
		})

		t.Run("非nilの場合は値を保持したintポインタを返す", func(t *testing.T) {
			t.Parallel()
			got := int32PtrToIntPtr(ptr.To(int32(42)))
			require.NotNil(t, got)
			assert.Equal(t, 42, *got)
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
