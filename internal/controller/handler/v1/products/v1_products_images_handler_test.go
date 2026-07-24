package products

import (
	"bytes"
	"context"
	"mime/multipart"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/products/gen"
	"go-boilerplate/internal/usecase/boundary/auth"
	productuc "go-boilerplate/internal/usecase/product"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// pngBytes は、Content-Type sniff が image/png になるテスト用バイト列です。
var pngBytes = []byte("\x89PNG\r\n\x1a\n dummy image bytes")

// newImageMultipartReader は、指定フィールド名で 1 件のファイルを持つ multipart.Reader を生成します。
func newImageMultipartReader(t *testing.T, fieldName string, data []byte) *multipart.Reader {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile(fieldName, "image")
	require.NoError(t, err)
	_, err = fw.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return multipart.NewReader(&buf, w.Boundary())
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

func Test_server_PostProductsImages(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みで画像をアップロードし201とパスを返す", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)

			mockApp.EXPECT().
				UploadProductImage(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(productuc.UploadProductImageParams{})).
				DoAndReturn(func(_ context.Context, _ *auth.Authn, p productuc.UploadProductImageParams) (productuc.ProductImageView, error) {
					assert.Equal(t, "image/png", p.ContentType)
					assert.Equal(t, pngBytes, p.Data)
					return productuc.ProductImageView{Path: "products/abc.png"}, nil
				})

			resp, err := s.PostProductsImages(authnContext(t), gen.PostProductsImagesRequestObject{
				Body: newImageMultipartReader(t, formFieldImage, pngBytes),
			})
			require.NoError(t, err)

			actual, ok := resp.(gen.PostProductsImages201JSONResponse)
			require.True(t, ok)
			assert.Equal(t, "products/abc.png", actual.ImagePath)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("authnが無い場合は認証エラーを返す", func(t *testing.T) {
			t.Parallel()
			s, _ := newServer(t)

			resp, err := s.PostProductsImages(context.Background(), gen.PostProductsImagesRequestObject{
				Body: newImageMultipartReader(t, formFieldImage, pngBytes),
			})
			assert.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
		})

		t.Run("imageフィールドが無い場合は422を返す", func(t *testing.T) {
			t.Parallel()
			s, _ := newServer(t)

			resp, err := s.PostProductsImages(authnContext(t), gen.PostProductsImagesRequestObject{
				Body: newImageMultipartReader(t, "other", pngBytes),
			})
			assert.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrValidation)
		})

		t.Run("Usecaseがエラーを返した場合はエラーを返す", func(t *testing.T) {
			t.Parallel()
			s, mockApp := newServer(t)

			mockApp.EXPECT().
				UploadProductImage(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(productuc.ProductImageView{}, apperror.ErrUnsupportedMediaType)

			resp, err := s.PostProductsImages(authnContext(t), gen.PostProductsImagesRequestObject{
				Body: newImageMultipartReader(t, formFieldImage, pngBytes),
			})
			assert.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrUnsupportedMediaType)
		})
	})
}

func Test_readImagePart(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("imageフィールドのバイト列を読み出す", func(t *testing.T) {
			t.Parallel()

			data, err := readImagePart(newImageMultipartReader(t, formFieldImage, pngBytes))
			require.NoError(t, err)
			assert.Equal(t, pngBytes, data)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("readerがnilの場合は422を返す", func(t *testing.T) {
			t.Parallel()

			_, err := readImagePart(nil)
			require.ErrorIs(t, err, apperror.ErrValidation)
		})

		t.Run("imageフィールドが無い場合は422を返す", func(t *testing.T) {
			t.Parallel()

			_, err := readImagePart(newImageMultipartReader(t, "other", pngBytes))
			require.ErrorIs(t, err, apperror.ErrValidation)
		})
	})
}
