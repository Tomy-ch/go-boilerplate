package product

import (
	"context"
	"strings"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/kernel/money"
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
	"go-boilerplate/pkg/uuid"

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
	status, err := domainproduct.NewStatusRef(uuid.NewTestFromSalt(t, salt+"_status"), "在庫あり")
	require.NoError(t, err)
	category, err := domainproduct.NewCategoryRef(uuid.NewTestFromSalt(t, salt+"_category"), "電子機器")
	require.NoError(t, err)
	p, err := domainproduct.New(
		uuid.NewTestFromSalt(t, salt),
		"商品-"+salt,
		ptr.To("説明-"+salt),
		mustPrice(t, "10.00"),
		5,
		ptr.To(2),
		status,
		category,
		ptr.To(publishedAt),
		nil,
	)
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
					assert.Equal(t, pngData, obj.Body)
					return objectstorage.Path(obj.Key), nil
				})

			u := &usecase{tracer: lt, authorizer: authorizer, storage: storage, maxUploadBytes: 1024}
			view, err := u.UploadProductImage(context.Background(), &auth.Authn{},
				UploadProductImageParams{ContentType: "image/png", Data: pngData})

			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(view.Path, "products/"))
			assert.True(t, strings.HasSuffix(view.Path, ".png"))
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

		t.Run("フィルタ・並び順・カーソル境界がドメインのListParamsへ引き渡される", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_product.NewMockRepository(gomock.NewController(t))

			categoryID := uuid.NewTestFromSalt(t, "filter_category")
			statusID := uuid.NewTestFromSalt(t, "filter_status")
			keyword := "イヤホン"

			afterID := uuid.NewTestFromSalt(t, "after_id")
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
				Cursor:     cursor,
				CategoryID: &categoryID,
				StatusID:   &statusID,
				Keyword:    &keyword,
				Ascending:  true,
			})
			require.NoError(t, err)

			assert.True(t, captured.Ascending)
			require.NotNil(t, captured.CategoryID)
			assert.Equal(t, categoryID, *captured.CategoryID)
			require.NotNil(t, captured.StatusID)
			assert.Equal(t, statusID, *captured.StatusID)
			require.NotNil(t, captured.Keyword)
			assert.Equal(t, keyword, *captured.Keyword)
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
			assert.Nil(t, actual)
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
			assert.Nil(t, actual)
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
			assert.Nil(t, actual)
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

			id := uuid.NewTestFromSalt(t, "get_product_missing")
			repo.EXPECT().FindPublishedByID(gomock.Any(), id).Return(nil, apperror.ErrNotFound)

			u := &usecase{tracer: lt, repo: repo}
			actual, err := u.GetProduct(context.Background(), id)
			require.ErrorIs(t, err, apperror.ErrNotFound)
			assert.Equal(t, ProductView{}, actual)
		})
	})
}
