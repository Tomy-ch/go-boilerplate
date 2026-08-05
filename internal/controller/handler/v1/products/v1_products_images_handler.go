package products

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/products/gen"
	productuc "go-boilerplate/internal/usecase/product"
	"go-boilerplate/pkg/xerrors"
)

const formFieldImage = "image"

// PostProductsImages は、admin が商品画像をアップロードし、格納先パスを返します。認証必須です。
// Content-Type は実バイトから判定（sniff）してユースケースへ渡し、形式・サイズ検証はユースケースが行います。
func (s *server) PostProductsImages(ctx context.Context, request gen.PostProductsImagesRequestObject) (gen.PostProductsImagesResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	data, err := readImagePart(request.Body)
	if err != nil {
		return nil, err
	}

	view, err := s.uc.UploadProductImage(ctx, &authn, productuc.UploadProductImageParams{
		ContentType: http.DetectContentType(data),
		Data:        data,
	})
	if err != nil {
		return nil, err
	}

	return gen.PostProductsImages201JSONResponse(gen.ProductImageResponse{
		ImagePath: view.Path,
	}), nil
}

// readImagePart は、multipart リクエストから "image" フィールドのバイト列を読み出します。
// フィールドが存在しない場合は 422 を返します。
func readImagePart(reader *multipart.Reader) ([]byte, error) {
	if reader == nil {
		return nil, xerrors.Wrap(apperror.ErrValidation, "multipart body is required")
	}
	for {
		part, err := reader.NextPart()
		if xerrors.Is(err, io.EOF) {
			return nil, xerrors.Wrap(apperror.ErrValidation, "image field is required")
		}
		if err != nil {
			return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "failed to read multipart body: "+err.Error())
		}
		if part.FormName() != formFieldImage {
			_ = part.Close()
			continue
		}
		data, rerr := io.ReadAll(part)
		_ = part.Close()
		if rerr != nil {
			return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "failed to read image field: "+rerr.Error())
		}
		return data, nil
	}
}
