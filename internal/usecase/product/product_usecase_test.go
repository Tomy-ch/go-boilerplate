package product

import (
	"context"
	"strings"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/lexicon/money"
	domainproduct "go-boilerplate/internal/domain/product"
	mock_category "go-boilerplate/internal/domain/product/category/mock"
	mock_product "go-boilerplate/internal/domain/product/mock"
	mock_status "go-boilerplate/internal/domain/product/status/mock"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	mock_authz "go-boilerplate/internal/usecase/boundary/authz/mock"
	"go-boilerplate/internal/usecase/boundary/objectstorage"
	mock_objectstorage "go-boilerplate/internal/usecase/boundary/objectstorage/mock"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	"go-boilerplate/internal/usecase/testkit"
	"go-boilerplate/internal/usecase/tools/paging"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/ptr"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// mustPrice は、テスト用に十進文字列から非負の money.Price を構築します。
func mustPrice(t *testing.T, s string) money.Price {
	t.Helper()
	p, err := money.NewPrice(decimaltestkit.MustParse(t, s))
	require.NoError(t, err)
	return p
}

func newTestProduct(t *testing.T, salt string, publishedAt time.Time) *domainproduct.Product {
	t.Helper()
	status, err := domainproduct.NewStatusRef(uuidtestkit.NewTestFromSalt(t, salt+"_status"), "在庫あり")
	require.NoError(t, err)
	category, err := domainproduct.NewCategoryRef(uuidtestkit.NewTestFromSalt(t, salt+"_category"), "電子機器")
	require.NoError(t, err)
	p, err := domainproduct.New(uuidtestkit.NewTestFromSalt(t, salt), domainproduct.Attributes{
		Name:                  "商品-" + salt,
		Description:           ptr.To("説明-" + salt),
		Price:                 mustPrice(t, "10.00"),
		Quantity:              5,
		StockWarningThreshold: ptr.To(2),
		Status:                status,
		Category:              category,
		PublishedAt:           ptr.To(publishedAt),
	})
	require.NoError(t, err)
	return p
}

// newDefaultCursor は、first=2（limit=2）の先頭ページ用カーソルを生成するテストヘルパーです。
func newDefaultCursor(t *testing.T) *paging.Cursor {
	t.Helper()
	first := 2
	c, err := paging.NewCursor(nil, &first)
	require.NoError(t, err)
	return c
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		tf := observability.NewNoopTracerFactory(t)
		txm := mock_tx.NewMockManager(ctrl)
		repo := mock_product.NewMockRepository(ctrl)
		categoryRepo := mock_category.NewMockRepository(ctrl)
		statusRepo := mock_status.NewMockRepository(ctrl)
		storage := mock_objectstorage.NewMockStorage(ctrl)
		authorizer := mock_authz.NewMockAuthorizer(ctrl)

		expected := &usecase{
			tracer:         tf.Usecase(),
			txm:            txm,
			repo:           repo,
			categoryRepo:   categoryRepo,
			statusRepo:     statusRepo,
			storage:        storage,
			authorizer:     authorizer,
			maxUploadBytes: 5242880,
		}
		actual := New(txm, repo, categoryRepo, statusRepo, storage, authorizer, 5242880, tf)

		assert.Equal(t, expected, actual)
	})
}

func Test_parseProductPriceFilter(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("十進文字列をPriceへ変換する", func(t *testing.T) {
			t.Parallel()

			actual, err := parseProductPriceFilter("minPrice", "10.50")

			require.NoError(t, err)
			require.NotNil(t, actual)
			assert.True(t, actual.Decimal().Equal(decimaltestkit.MustParse(t, "10.50")))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("負数の場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := parseProductPriceFilter("maxPrice", "-1")

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			assert.Nil(t, actual)
		})

		t.Run("長すぎる場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := parseProductPriceFilter("minPrice", strings.Repeat("1", 41))

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			assert.Nil(t, actual)
		})
	})
}

func Test_usecase_UploadProductImage(t *testing.T) {
	t.Parallel()

	pngData := []byte("\x89PNG\r\n\x1a\n dummy image bytes")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("adminが有効な画像をアップロードし格納パスを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)
			storage := mock_objectstorage.NewMockStorage(ctrl)

			authorizer.EXPECT().
				Authorize(gomock.Any(), gomock.Any(), authz.ActionProductImageUpload, gomock.Any()).
				Return(nil)
			storage.EXPECT().Put(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, obj objectstorage.PutObject) (objectstorage.Path, error) {
					assert.True(t, strings.HasPrefix(obj.Key, "products/"))
					assert.True(t, strings.HasSuffix(obj.Key, ".png"))
					assert.Equal(t, "image/png", obj.ContentType)
					assert.Equal(t, "public, max-age=31536000, immutable", obj.CacheControl)
					assert.Equal(t, pngData, obj.Body)
					return objectstorage.Path(obj.Key), nil
				},
			)

			u := &usecase{tracer: lt, authorizer: authorizer, storage: storage, maxUploadBytes: 1024}
			view, err := u.UploadProductImage(context.Background(), &auth.Authn{},
				UploadProductImageParams{ContentType: "image/png", Data: pngData})

			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(view.Path, "products/"))
			assert.True(t, strings.HasSuffix(view.Path, ".png"))
		})

		t.Run("サイズ上限ちょうどの画像はアップロードできる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)
			storage := mock_objectstorage.NewMockStorage(ctrl)

			authorizer.EXPECT().
				Authorize(gomock.Any(), gomock.Any(), authz.ActionProductImageUpload, gomock.Any()).
				Return(nil)
			storage.EXPECT().Put(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, obj objectstorage.PutObject) (objectstorage.Path, error) {
					return objectstorage.Path(obj.Key), nil
				},
			)

			u := &usecase{
				tracer: lt, authorizer: authorizer, storage: storage,
				maxUploadBytes: int64(len(pngData)),
			}
			view, err := u.UploadProductImage(context.Background(), &auth.Authn{},
				UploadProductImageParams{ContentType: "image/png", Data: pngData})

			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(view.Path, "products/"))
		})

		t.Run("画像アップロードを所有者なしリソースとして認可する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)
			storage := mock_objectstorage.NewMockStorage(ctrl)

			authorizer.EXPECT().
				Authorize(gomock.Any(), gomock.Any(), authz.ActionProductImageUpload, authz.NewResource("product", nil)).
				Return(nil)
			storage.EXPECT().Put(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, obj objectstorage.PutObject) (objectstorage.Path, error) {
					return objectstorage.Path(obj.Key), nil
				},
			)

			u := &usecase{tracer: lt, authorizer: authorizer, storage: storage, maxUploadBytes: 1024}
			_, err := u.UploadProductImage(context.Background(), &auth.Authn{},
				UploadProductImageParams{ContentType: "image/png", Data: pngData})

			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("authnがnilの場合は認証エラーを返す", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			u := &usecase{tracer: lt, maxUploadBytes: 1024}

			_, err := u.UploadProductImage(context.Background(), nil,
				UploadProductImageParams{ContentType: "image/png", Data: pngData})

			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
		})

		t.Run("認可が拒否された場合は権限エラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)
			authorizer.EXPECT().
				Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(authz.ErrForbidden)

			u := &usecase{tracer: lt, authorizer: authorizer, maxUploadBytes: 1024}
			_, err := u.UploadProductImage(context.Background(), &auth.Authn{},
				UploadProductImageParams{ContentType: "image/png", Data: pngData})

			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})

		t.Run("空データは検証エラー(422)を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)
			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			u := &usecase{tracer: lt, authorizer: authorizer, maxUploadBytes: 1024}
			_, err := u.UploadProductImage(context.Background(), &auth.Authn{},
				UploadProductImageParams{ContentType: "image/png", Data: nil})

			require.ErrorIs(t, err, apperror.ErrValidation)
		})

		t.Run("非対応Content-Typeは415を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)
			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			u := &usecase{tracer: lt, authorizer: authorizer, maxUploadBytes: 1024}
			_, err := u.UploadProductImage(context.Background(), &auth.Authn{},
				UploadProductImageParams{ContentType: "image/gif", Data: pngData})

			require.ErrorIs(t, err, apperror.ErrUnsupportedMediaType)
		})

		t.Run("サイズ超過は413を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)
			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			u := &usecase{tracer: lt, authorizer: authorizer, maxUploadBytes: 4}
			_, err := u.UploadProductImage(context.Background(), &auth.Authn{},
				UploadProductImageParams{ContentType: "image/png", Data: pngData})

			require.ErrorIs(t, err, apperror.ErrPayloadTooLarge)
		})

		t.Run("storageへの格納が失敗した場合はエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)
			storage := mock_objectstorage.NewMockStorage(ctrl)

			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			storage.EXPECT().
				Put(gomock.Any(), gomock.Any()).
				Return(objectstorage.Path(""), apperror.ErrUnavailable)

			u := &usecase{tracer: lt, authorizer: authorizer, storage: storage, maxUploadBytes: 1024}
			_, err := u.UploadProductImage(context.Background(), &auth.Authn{},
				UploadProductImageParams{ContentType: "image/png", Data: pngData})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}

func Test_parseProductListRange(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("価格と在庫数の上下限を変換する", func(t *testing.T) {
			t.Parallel()
			params := ListProductsParams{
				MinPrice: ptr.To("10.50"), MaxPrice: ptr.To("99.99"),
				MinQuantity: ptr.To[int32](2), MaxQuantity: ptr.To[int32](20),
			}
			actual, err := parseProductListRange(params)
			require.NoError(t, err)
			require.NotNil(t, actual.minPrice)
			assert.True(t, actual.minPrice.Decimal().Equal(decimaltestkit.MustParse(t, "10.50")))
			require.NotNil(t, actual.maxPrice)
			assert.True(t, actual.maxPrice.Decimal().Equal(decimaltestkit.MustParse(t, "99.99")))
			assert.Equal(t, int32(2), *actual.minQuantity)
			assert.Equal(t, int32(20), *actual.maxQuantity)
		})

		t.Run("価格の下限と上限が等しい場合、単一価格の絞り込みとして受理する", func(t *testing.T) {
			t.Parallel()
			actual, err := parseProductListRange(ListProductsParams{
				MinPrice: ptr.To("10.00"), MaxPrice: ptr.To("10.00"),
			})
			require.NoError(t, err)
			require.NotNil(t, actual.minPrice)
			assert.True(t, actual.minPrice.Decimal().Equal(decimaltestkit.MustParse(t, "10.00")))
			require.NotNil(t, actual.maxPrice)
			assert.True(t, actual.maxPrice.Decimal().Equal(decimaltestkit.MustParse(t, "10.00")))
		})

		t.Run("在庫数の下限と上限が等しい場合、単一在庫数の絞り込みとして受理する", func(t *testing.T) {
			t.Parallel()
			actual, err := parseProductListRange(ListProductsParams{
				MinQuantity: ptr.To[int32](10), MaxQuantity: ptr.To[int32](10),
			})
			require.NoError(t, err)
			assert.Equal(t, int32(10), *actual.minQuantity)
			assert.Equal(t, int32(10), *actual.maxQuantity)
		})

		t.Run("在庫数の下限が0の場合、非負の境界として受理する", func(t *testing.T) {
			t.Parallel()
			actual, err := parseProductListRange(ListProductsParams{MinQuantity: ptr.To[int32](0)})
			require.NoError(t, err)
			assert.Equal(t, int32(0), *actual.minQuantity)
		})

		t.Run("在庫数の上限が0の場合、非負の境界として受理する", func(t *testing.T) {
			t.Parallel()
			actual, err := parseProductListRange(ListProductsParams{MaxQuantity: ptr.To[int32](0)})
			require.NoError(t, err)
			assert.Equal(t, int32(0), *actual.maxQuantity)
		})

		t.Run("最低価格のみを指定した場合、上限を課さない", func(t *testing.T) {
			t.Parallel()
			actual, err := parseProductListRange(ListProductsParams{MinPrice: ptr.To("10.50")})
			require.NoError(t, err)
			require.NotNil(t, actual.minPrice)
			assert.True(t, actual.minPrice.Decimal().Equal(decimaltestkit.MustParse(t, "10.50")))
			assert.Nil(t, actual.maxPrice)
		})

		t.Run("最高価格のみを指定した場合、下限を課さない", func(t *testing.T) {
			t.Parallel()
			actual, err := parseProductListRange(ListProductsParams{MaxPrice: ptr.To("99.99")})
			require.NoError(t, err)
			assert.Nil(t, actual.minPrice)
			require.NotNil(t, actual.maxPrice)
			assert.True(t, actual.maxPrice.Decimal().Equal(decimaltestkit.MustParse(t, "99.99")))
		})

		t.Run("最低在庫数のみを指定した場合、上限を課さない", func(t *testing.T) {
			t.Parallel()
			actual, err := parseProductListRange(ListProductsParams{MinQuantity: ptr.To[int32](2)})
			require.NoError(t, err)
			assert.Equal(t, int32(2), *actual.minQuantity)
			assert.Nil(t, actual.maxQuantity)
		})

		t.Run("最高在庫数のみを指定した場合、下限を課さない", func(t *testing.T) {
			t.Parallel()
			actual, err := parseProductListRange(ListProductsParams{MaxQuantity: ptr.To[int32](20)})
			require.NoError(t, err)
			assert.Nil(t, actual.minQuantity)
			assert.Equal(t, int32(20), *actual.maxQuantity)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最低価格が非数値の場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := parseProductListRange(ListProductsParams{MinPrice: ptr.To("invalid")})
			assert.Empty(t, actual)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("最低価格が長すぎる場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := parseProductListRange(ListProductsParams{MinPrice: ptr.To(strings.Repeat("1", 41))})
			assert.Empty(t, actual)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("最高価格が負数の場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := parseProductListRange(ListProductsParams{MaxPrice: ptr.To("-1")})
			assert.Empty(t, actual)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("最高価格が長すぎる場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := parseProductListRange(ListProductsParams{MaxPrice: ptr.To(strings.Repeat("1", 41))})
			assert.Empty(t, actual)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("最低価格が最高価格を超える場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := parseProductListRange(ListProductsParams{MinPrice: ptr.To("20"), MaxPrice: ptr.To("10")})
			assert.Empty(t, actual)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("最低在庫数が負数の場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := parseProductListRange(ListProductsParams{MinQuantity: ptr.To[int32](-1)})
			assert.Empty(t, actual)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("最高在庫数が負数の場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := parseProductListRange(ListProductsParams{MaxQuantity: ptr.To[int32](-1)})
			assert.Empty(t, actual)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("最低在庫数が最高在庫数を超える場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := parseProductListRange(ListProductsParams{
				MinQuantity: ptr.To[int32](20), MaxQuantity: ptr.To[int32](10),
			})
			assert.Empty(t, actual)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})
	})
}

func Test_usecase_CountProducts(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("検索条件を検証してRepositoryへ渡し一致件数を返す", func(t *testing.T) {
			t.Parallel()

			repo := mock_product.NewMockRepository(gomock.NewController(t))
			categoryID := uuidtestkit.NewTestFromSalt(t, "count_category")
			statusID := uuidtestkit.NewTestFromSalt(t, "count_status")
			repo.EXPECT().CountPublished(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params domainproduct.CountPublishedParams) (int64, error) {
					assert.Equal(t, categoryID, *params.CategoryID)
					assert.Equal(t, statusID, *params.StatusID)
					assert.Equal(t, "イヤホン", *params.Keyword)
					assert.True(t, params.MinPrice.Decimal().Equal(decimaltestkit.MustParse(t, "10.50")))
					assert.True(t, params.MaxPrice.Decimal().Equal(decimaltestkit.MustParse(t, "99.99")))
					assert.Equal(t, int32(2), *params.MinQuantity)
					assert.Equal(t, int32(20), *params.MaxQuantity)
					return 7, nil
				})

			u := &usecase{tracer: observability.NewMockUsecaseLayerTracer(t), repo: repo}
			actual, err := u.CountProducts(context.Background(), CountProductsParams{
				CategoryID: &categoryID, StatusID: &statusID, Keyword: ptr.To("イヤホン"),
				MinPrice: ptr.To("10.50"), MaxPrice: ptr.To("99.99"),
				MinQuantity: ptr.To[int32](2), MaxQuantity: ptr.To[int32](20),
			})

			require.NoError(t, err)
			assert.Equal(t, ProductCountView{Count: 7}, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("範囲が不正な場合はRepositoryを呼ばずErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			u := &usecase{
				tracer: observability.NewMockUsecaseLayerTracer(t),
				repo:   mock_product.NewMockRepository(gomock.NewController(t)),
			}
			actual, err := u.CountProducts(context.Background(), CountProductsParams{
				MinPrice: ptr.To("20"), MaxPrice: ptr.To("10"),
			})

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			assert.Empty(t, actual)
		})

		t.Run("Repositoryのエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			repo := mock_product.NewMockRepository(gomock.NewController(t))
			expectedErr := testkit.ExpectedDBError()
			repo.EXPECT().CountPublished(gomock.Any(), gomock.Any()).Return(int64(0), expectedErr)
			u := &usecase{tracer: observability.NewMockUsecaseLayerTracer(t), repo: repo}

			actual, err := u.CountProducts(context.Background(), CountProductsParams{})

			require.ErrorIs(t, err, expectedErr)
			assert.Empty(t, actual)
		})
	})
}

func Test_usecase_ListProducts(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("取得件数がlimitを超える場合、末尾を切り詰めてNextCursorを設定する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_product.NewMockRepository(gomock.NewController(t))

			p1 := newTestProduct(t, "list_p1", base)
			p2 := newTestProduct(t, "list_p2", base.Add(-time.Hour))
			p3 := newTestProduct(t, "list_p3", base.Add(-2*time.Hour))

			// first=2 なので limit+1=3 件で問い合わせ、3 件返れば次ページありと判定する。
			repo.EXPECT().
				FindPublishedList(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, params domainproduct.ListParams) (domainproduct.Products, error) {
					assert.Equal(t, int32(3), params.Limit)
					return domainproduct.Products{p1, p2, p3}, nil
				})

			u := &usecase{tracer: lt, repo: repo}
			actual, err := u.ListProducts(ctx, ListProductsParams{Cursor: newDefaultCursor(t)})
			require.NoError(t, err)
			require.Len(t, actual.Items, 2)
			assert.Equal(t, p1.ID(), actual.Items[0].ID)
			assert.Equal(t, p2.ID(), actual.Items[1].ID)

			require.NotNil(t, actual.NextCursor)
			cursor, cErr := paging.NewCursor(actual.NextCursor, nil)
			require.NoError(t, cErr)
			decoded, dErr := decodeProductCursor(cursor)
			require.NoError(t, dErr)
			// NextCursor は切り詰め後の末尾行(p2)を指す。
			assert.Equal(t, ptr.Deref(p2.PublishedAt(), time.Time{}), decoded.publishedAt)
			assert.Equal(t, p2.ID(), decoded.id)
		})

		t.Run("取得件数がlimit以下の場合、NextCursorはnilになる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_product.NewMockRepository(gomock.NewController(t))

			p1 := newTestProduct(t, "single_p1", base)
			repo.EXPECT().FindPublishedList(gomock.Any(), gomock.Any()).Return(domainproduct.Products{p1}, nil)

			u := &usecase{tracer: lt, repo: repo}
			actual, err := u.ListProducts(ctx, ListProductsParams{Cursor: newDefaultCursor(t)})
			require.NoError(t, err)
			require.Len(t, actual.Items, 1)
			assert.Nil(t, actual.NextCursor)
		})

		t.Run("取得結果が0件の場合、空のItemsとnilのNextCursorを返す", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_product.NewMockRepository(gomock.NewController(t))

			repo.EXPECT().FindPublishedList(gomock.Any(), gomock.Any()).Return(domainproduct.Products{}, nil)

			u := &usecase{tracer: lt, repo: repo}
			actual, err := u.ListProducts(ctx, ListProductsParams{Cursor: newDefaultCursor(t)})
			require.NoError(t, err)
			assert.Empty(t, actual.Items)
			assert.Nil(t, actual.NextCursor)
		})

		t.Run("ゼロ値Cursor混入時もproducts末尾のpanicを避け空ItemsとnilのNextCursorを返す", func(t *testing.T) {
			t.Parallel()

			// ゼロ値 Cursor（limit=0）では Limit()=0 かつ Repository へは limit+1=1 件要求される。
			// 1 件返ると hasNext=len(1)>0=true・切り詰めで products が空になり、len(products)>0 の安全弁が
			// 無いと products[len-1] が products[-1] で panic する。安全弁により panic せず空一覧を返す。
			ctx := context.Background()
			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_product.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().FindPublishedList(gomock.Any(), gomock.Any()).Return(
				domainproduct.Products{newTestProduct(t, "zero_cursor", base)}, nil,
			)

			u := &usecase{tracer: lt, repo: repo}
			actual, err := u.ListProducts(ctx, ListProductsParams{Cursor: &paging.Cursor{}})
			require.NoError(t, err)
			assert.Empty(t, actual.Items)
			assert.Nil(t, actual.NextCursor)
		})

		t.Run("フィルタ・並び順・カーソル境界がドメインのListParamsへ引き渡される", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_product.NewMockRepository(gomock.NewController(t))

			categoryID := uuidtestkit.NewTestFromSalt(t, "filter_category")
			statusID := uuidtestkit.NewTestFromSalt(t, "filter_status")
			keyword := "イヤホン"
			minPrice := "10.50"
			maxPrice := "99.99"
			minQuantity := int32(2)
			maxQuantity := int32(20)

			afterID := uuidtestkit.NewTestFromSalt(t, "after_id")
			afterAt := base.Add(-3 * time.Hour)
			after := paging.EncodeCursor(afterAt.Format(time.RFC3339Nano), afterID.String())

			var captured domainproduct.ListParams
			repo.EXPECT().
				FindPublishedList(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, params domainproduct.ListParams) (domainproduct.Products, error) {
					captured = params
					return domainproduct.Products{}, nil
				})

			cursor, err := paging.NewCursor(&after, nil)
			require.NoError(t, err)

			u := &usecase{tracer: lt, repo: repo}
			_, err = u.ListProducts(ctx, ListProductsParams{
				Cursor:      cursor,
				CategoryID:  &categoryID,
				StatusID:    &statusID,
				Keyword:     &keyword,
				MinPrice:    &minPrice,
				MaxPrice:    &maxPrice,
				MinQuantity: &minQuantity,
				MaxQuantity: &maxQuantity,
				Ascending:   true,
			})
			require.NoError(t, err)

			assert.True(t, captured.Ascending)
			require.NotNil(t, captured.CategoryID)
			assert.Equal(t, categoryID, *captured.CategoryID)
			require.NotNil(t, captured.StatusID)
			assert.Equal(t, statusID, *captured.StatusID)
			require.NotNil(t, captured.Keyword)
			assert.Equal(t, keyword, *captured.Keyword)
			require.NotNil(t, captured.MinPrice)
			assert.True(t, captured.MinPrice.Decimal().Equal(decimaltestkit.MustParse(t, minPrice)))
			require.NotNil(t, captured.MaxPrice)
			assert.True(t, captured.MaxPrice.Decimal().Equal(decimaltestkit.MustParse(t, maxPrice)))
			require.NotNil(t, captured.MinQuantity)
			assert.Equal(t, minQuantity, *captured.MinQuantity)
			require.NotNil(t, captured.MaxQuantity)
			assert.Equal(t, maxQuantity, *captured.MaxQuantity)
			require.NotNil(t, captured.AfterPublishedAt)
			assert.Equal(t, afterAt, *captured.AfterPublishedAt)
			require.NotNil(t, captured.AfterID)
			assert.Equal(t, afterID, *captured.AfterID)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("cursorがnilの場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_product.NewMockRepository(gomock.NewController(t))

			u := &usecase{tracer: lt, repo: repo}
			actual, err := u.ListProducts(context.Background(), ListProductsParams{Cursor: nil})
			assert.Empty(t, actual)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("価格範囲が不正な場合、Repositoryを呼ばずErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			u := &usecase{
				tracer: observability.NewMockUsecaseLayerTracer(t),
				repo:   mock_product.NewMockRepository(gomock.NewController(t)),
			}
			actual, err := u.ListProducts(context.Background(), ListProductsParams{
				Cursor: newDefaultCursor(t), MinPrice: ptr.To("20"), MaxPrice: ptr.To("10"),
			})
			assert.Empty(t, actual)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("カーソルの復号に失敗した場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_product.NewMockRepository(gomock.NewController(t))

			// キーが1つだけの不正カーソル。
			bad := paging.EncodeCursor("only-one-key")
			cursor, err := paging.NewCursor(&bad, nil)
			require.NoError(t, err)

			u := &usecase{tracer: lt, repo: repo}
			actual, err := u.ListProducts(context.Background(), ListProductsParams{Cursor: cursor})
			assert.Empty(t, actual)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("リポジトリのエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_product.NewMockRepository(gomock.NewController(t))

			expectedErr := testkit.ExpectedDBError()
			repo.EXPECT().FindPublishedList(gomock.Any(), gomock.Any()).Return(nil, expectedErr)

			u := &usecase{tracer: lt, repo: repo}
			actual, err := u.ListProducts(context.Background(), ListProductsParams{Cursor: newDefaultCursor(t)})
			require.ErrorIs(t, err, expectedErr)
			assert.Empty(t, actual)
		})
	})
}

func Test_usecase_GetProduct(t *testing.T) {
	t.Parallel()

	published := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("公開商品を取得しProductViewへ写像して返す", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_product.NewMockRepository(gomock.NewController(t))

			p := newTestProduct(t, "get_product", published)
			repo.EXPECT().FindPublishedByID(gomock.Any(), p.ID()).Return(p, nil)

			u := &usecase{tracer: lt, repo: repo}
			actual, err := u.GetProduct(context.Background(), p.ID())
			require.NoError(t, err)
			assert.Equal(t, p.ID(), actual.ID)
			assert.Equal(t, p.Name(), actual.Name)
			assert.Equal(t, p.Description(), actual.Description)
			assert.True(t, p.Price().Decimal().Equal(actual.Price))
			assert.Equal(t, p.Quantity(), actual.Quantity)
			assert.Equal(t, p.StockWarningThreshold(), actual.StockWarningThreshold)
			assert.Equal(t, p.Status().ID(), actual.StatusID)
			assert.Equal(t, p.Status().Name(), actual.StatusName)
			assert.Equal(t, p.Category().ID(), actual.CategoryID)
			assert.Equal(t, p.Category().Name(), actual.CategoryName)
			assert.Equal(t, p.PublishedAt(), actual.PublishedAt)
			assert.Equal(t, p.Version(), actual.Version)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Repositoryが未存在・非公開でNotFoundを返す場合_そのまま伝播する", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_product.NewMockRepository(gomock.NewController(t))

			id := uuidtestkit.NewTestFromSalt(t, "get_product_missing")
			repo.EXPECT().FindPublishedByID(gomock.Any(), id).Return(nil, apperror.ErrNotFound)

			u := &usecase{tracer: lt, repo: repo}
			actual, err := u.GetProduct(context.Background(), id)
			require.ErrorIs(t, err, apperror.ErrNotFound)
			assert.Equal(t, ProductView{}, actual)
		})
	})
}

func Test_toProductView(t *testing.T) {
	t.Parallel()

	published := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("エンティティの属性とステータス・カテゴリの名称を展開したDTOへ変換する", func(t *testing.T) {
			t.Parallel()

			status, err := domainproduct.NewStatusRef(uuidtestkit.NewTestFromSalt(t, "to_view_status"), "在庫あり")
			require.NoError(t, err)
			category, err := domainproduct.NewCategoryRef(uuidtestkit.NewTestFromSalt(t, "to_view_category"), "電子機器")
			require.NoError(t, err)
			p, err := domainproduct.Reconstruct(uuidtestkit.NewTestFromSalt(t, "to_view"), domainproduct.Attributes{
				Name:                  "商品-to_view",
				Description:           ptr.To("説明-to_view"),
				Price:                 mustPrice(t, "12.34"),
				Quantity:              5,
				StockWarningThreshold: ptr.To(2),
				Status:                status,
				Category:              category,
				PublishedAt:           ptr.To(published),
				Images: []domainproduct.Image{
					domainproduct.NewImage(
						uuidtestkit.NewTestFromSalt(t, "to_view_image"),
						domainproduct.ImageAttributes{ImagePath: "products/to_view.png", SortKey: 1},
					),
				},
			}, 7)
			require.NoError(t, err)

			actual := toProductView(p)

			assert.Equal(t, p.ID(), actual.ID)
			assert.Equal(t, "商品-to_view", actual.Name)
			require.NotNil(t, actual.Description)
			assert.Equal(t, "説明-to_view", *actual.Description)
			assert.Equal(t, "12.34", actual.Price.String())
			assert.Equal(t, 5, actual.Quantity)
			require.NotNil(t, actual.StockWarningThreshold)
			assert.Equal(t, 2, *actual.StockWarningThreshold)
			assert.Equal(t, status.ID(), actual.StatusID)
			assert.Equal(t, "在庫あり", actual.StatusName)
			assert.Equal(t, category.ID(), actual.CategoryID)
			assert.Equal(t, "電子機器", actual.CategoryName)
			require.NotNil(t, actual.PublishedAt)
			assert.Equal(t, published, *actual.PublishedAt)
			require.Len(t, actual.Images, 1)
			assert.Equal(t, "products/to_view.png", actual.Images[0].Path)
			assert.Equal(t, 1, actual.Images[0].SortKey)
			assert.Equal(t, 7, actual.Version)
		})

		t.Run("任意項目が未設定のエンティティはDTOでもnilのまま変換する", func(t *testing.T) {
			t.Parallel()

			status, err := domainproduct.NewStatusRef(uuidtestkit.NewTestFromSalt(t, "to_view_nil_status"), "在庫なし")
			require.NoError(t, err)
			category, err := domainproduct.NewCategoryRef(uuidtestkit.NewTestFromSalt(t, "to_view_nil_category"), "書籍")
			require.NoError(t, err)
			p, err := domainproduct.New(uuidtestkit.NewTestFromSalt(t, "to_view_nil"), domainproduct.Attributes{
				Name:     "商品-to_view_nil",
				Price:    mustPrice(t, "0.00"),
				Quantity: 0,
				Status:   status,
				Category: category,
			})
			require.NoError(t, err)

			actual := toProductView(p)

			assert.Nil(t, actual.Description)
			assert.Nil(t, actual.StockWarningThreshold)
			assert.Nil(t, actual.PublishedAt)
			assert.Empty(t, actual.Images)
			assert.Equal(t, "在庫なし", actual.StatusName)
		})
	})
}

// newUnpublishedTestProduct は、公開日時を持たない商品エンティティを構築します。
func newUnpublishedTestProduct(t *testing.T, salt string) *domainproduct.Product {
	t.Helper()
	status, err := domainproduct.NewStatusRef(uuidtestkit.NewTestFromSalt(t, salt+"_status"), "在庫あり")
	require.NoError(t, err)
	category, err := domainproduct.NewCategoryRef(uuidtestkit.NewTestFromSalt(t, salt+"_category"), "電子機器")
	require.NoError(t, err)
	p, err := domainproduct.New(uuidtestkit.NewTestFromSalt(t, salt), domainproduct.Attributes{
		Name:                  "商品-" + salt,
		Description:           ptr.To("説明-" + salt),
		Price:                 mustPrice(t, "10.00"),
		Quantity:              5,
		StockWarningThreshold: ptr.To(2),
		Status:                status,
		Category:              category,
		PublishedAt:           nil,
	})
	require.NoError(t, err)
	return p
}

func Test_ensurePublished(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("すべて公開中の場合、エラーを返さない", func(t *testing.T) {
			t.Parallel()

			publishedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
			products := domainproduct.Products{
				newTestProduct(t, "published_a", publishedAt),
				newTestProduct(t, "published_b", publishedAt),
			}

			assert.NoError(t, ensurePublished(products))
		})

		t.Run("空の場合、エラーを返さない", func(t *testing.T) {
			t.Parallel()

			assert.NoError(t, ensurePublished(domainproduct.Products{}))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未公開が混じる場合、SQLとドメイン定義の乖離としてエラーを返す", func(t *testing.T) {
			t.Parallel()

			publishedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
			products := domainproduct.Products{
				newTestProduct(t, "published_c", publishedAt),
				newUnpublishedTestProduct(t, "drifted"),
			}

			err := ensurePublished(products)
			require.ErrorIs(t, err, apperror.ErrInternal)
			assert.Contains(t, err.Error(), uuidtestkit.NewTestFromSalt(t, "drifted").String())
		})
	})
}

func Test_toProductImageItemViews(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("集約が保持する並びのまま出力DTOへ変換する", func(t *testing.T) {
			t.Parallel()

			images := []domainproduct.Image{
				domainproduct.NewImage(
					uuidtestkit.NewTestFromSalt(t, "item_view_1"),
					domainproduct.ImageAttributes{ImagePath: "products/a.png", SortKey: 1},
				),
				domainproduct.NewImage(
					uuidtestkit.NewTestFromSalt(t, "item_view_2"),
					domainproduct.ImageAttributes{ImagePath: "products/b.png", SortKey: 5},
				),
			}

			actual := toProductImageItemViews(images)

			require.Len(t, actual, 2)
			assert.Equal(t, ProductImageItemView{Path: "products/a.png", SortKey: 1}, actual[0])
			assert.Equal(t, ProductImageItemView{Path: "products/b.png", SortKey: 5}, actual[1])
		})

		t.Run("画像が空の場合、空を返す", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, toProductImageItemViews(nil))
		})
	})
}
