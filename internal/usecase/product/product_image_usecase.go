package product

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	"go-boilerplate/internal/usecase/boundary/objectstorage"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// imageKeyPrefix は、商品画像オブジェクトキーの接頭辞です。
const imageKeyPrefix = "products/"

// allowedImageContentTypes は、許可する画像 Content-Type から拡張子への対応表です。
// Content-Type は controller 側で実バイトから判定（sniff）した値を受け取ります。
var allowedImageContentTypes = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpg",
	"image/webp": "webp",
}

// UploadProductImageParams は、商品画像アップロードの入力です。
type UploadProductImageParams struct {
	// ContentType は、実バイトから判定した画像の MIME タイプです。
	ContentType string
	// Data は、画像のバイト列です。
	Data []byte
}

// ProductImageView は、商品画像アップロード結果の出力 DTO です。
type ProductImageView struct {
	// Path は、格納されたオブジェクトのパス（キー）です。表示 URL は上位が組み立てます。
	Path string
}

func (u *usecase) UploadProductImage(ctx context.Context, authn *auth.Authn, params UploadProductImageParams) (ProductImageView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if authn == nil {
		return ProductImageView{}, apperror.ErrUnauthenticated
	}
	if err := u.authorizer.Authorize(ctx, authn, authz.ActionProductImageUpload, authz.NewResource("product", nil)); err != nil {
		return ProductImageView{}, err
	}

	if len(params.Data) == 0 {
		return ProductImageView{}, xerrors.Wrap(apperror.ErrValidation, "image must not be empty")
	}
	ext, ok := allowedImageContentTypes[params.ContentType]
	if !ok {
		return ProductImageView{}, xerrors.Wrap(apperror.ErrUnsupportedMediaType, "unsupported image content type: "+params.ContentType)
	}
	if int64(len(params.Data)) > u.maxUploadBytes {
		return ProductImageView{}, xerrors.Wrap(apperror.ErrPayloadTooLarge, "image exceeds the maximum upload size")
	}

	id, err := uuid.New()
	if err != nil {
		return ProductImageView{}, xerrors.Wrap(err, "failed to generate object key")
	}
	key := imageKeyPrefix + id.String() + "." + ext

	path, err := u.storage.Put(ctx, objectstorage.PutObject{
		Key:         key,
		Body:        params.Data,
		ContentType: params.ContentType,
	})
	if err != nil {
		return ProductImageView{}, err
	}

	return ProductImageView{Path: string(path)}, nil
}
