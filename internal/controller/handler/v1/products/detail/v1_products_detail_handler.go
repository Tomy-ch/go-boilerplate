//go:generate oapi-codegen --include-tags=v1/products/detail --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/products/detail --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package detail は、/v1/products/{productId} エンドポイントに関連するハンドラを提供します。
package detail

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/products/detail/gen"
	"go-boilerplate/internal/observability"
	productuc "go-boilerplate/internal/usecase/product"
	"go-boilerplate/pkg/patch"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/nullable"
)

// ErrUnauthenticatedUser は、認証ユーザー情報が取得できない場合のエラーです。
var ErrUnauthenticatedUser = xerrors.Wrap(apperror.ErrUnauthenticated, "requires authenticated user")

type server struct {
	tracer observability.LayerTracer
	uc     productuc.Usecase
}

// BindHandler は、商品詳細のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc productuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetProductsDetail は、指定された UUID に該当する公開済み商品の詳細情報を取得します。
// 未存在・非公開はいずれもユースケースが NotFound を返し、404 で存在を秘匿します。
func (s *server) GetProductsDetail(ctx context.Context, request gen.GetProductsDetailRequestObject) (gen.GetProductsDetailResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	id := conv.UUID(request.ProductId)

	dto, err := s.uc.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}

	return gen.GetProductsDetail200JSONResponse(toProductResponse(dto)), nil
}

// PatchProductsDetail は、指定された UUID に該当する商品の属性を部分更新します。
// 送信されたフィールドのみを更新し、null 指定はクリアとして扱います。
func (s *server) PatchProductsDetail(
	ctx context.Context,
	request gen.PatchProductsDetailRequestObject,
) (gen.PatchProductsDetailResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, ok := ctxhelper.GetAuthn(ctx)
	if !ok {
		return nil, ErrUnauthenticatedUser
	}

	id := conv.UUID(request.ProductId)

	params := productuc.UpdateProductParams{
		Version:               int(request.Body.Version),
		Name:                  request.Body.Name,
		Price:                 request.Body.Price,
		Quantity:              int32PtrToIntPtr(request.Body.Quantity),
		CategoryID:            conv.UUIDPtr(request.Body.CategoryId),
		StatusID:              conv.UUIDPtr(request.Body.StatusId),
		Description:           toPatchField(request.Body.Description),
		StockWarningThreshold: toPatchFieldInt(request.Body.StockWarningThreshold),
		PublishedAt:           toPatchField(request.Body.PublishedAt),
		ImagePath:             toPatchField(request.Body.ImagePath),
	}

	dto, err := s.uc.UpdateProduct(ctx, &authn, id, params)
	if err != nil {
		return nil, err
	}

	return gen.PatchProductsDetail200JSONResponse(toProductResponse(dto)), nil
}

// toPatchField は、リクエストの 3 状態フィールドを部分更新の指定状態へ変換します。
func toPatchField[T any](v nullable.Nullable[T]) patch.Field[T] {
	if !v.IsSpecified() {
		return patch.Unspecified[T]()
	}
	if v.IsNull() {
		return patch.Null[T]()
	}
	return patch.Value(v.MustGet())
}

// toPatchFieldInt は、リクエストの 3 状態 int32 フィールドをユースケースの DTO の int へ変換します。
func toPatchFieldInt(v nullable.Nullable[int32]) patch.Field[int] {
	if !v.IsSpecified() {
		return patch.Unspecified[int]()
	}
	if v.IsNull() {
		return patch.Null[int]()
	}
	return patch.Value(int(v.MustGet()))
}

func toProductResponse(dto productuc.ProductView) gen.ProductResponse {
	return gen.ProductResponse{
		Id:                    dto.ID.ToPrimitive(),
		Name:                  dto.Name,
		Description:           dto.Description,
		Price:                 dto.Price.String(),
		Quantity:              toInt32(dto.Quantity),
		StockWarningThreshold: intPtrToInt32Ptr(dto.StockWarningThreshold),
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
		Version:     toInt32(dto.Version),
	}
}

// int32PtrToIntPtr は、リクエストの *int32 をユースケース DTO の *int へ変換します（nil はそのまま nil）。
func int32PtrToIntPtr(v *int32) *int {
	if v == nil {
		return nil
	}
	i := int(*v)
	return &i
}

// toInt32 は、ユースケースの DTO の int をレスポンスの int32 へ変換します。
// 値は 32bit 整数幅で永続化される在庫数・在庫警告閾値・バージョン由来のため範囲に収まります。
func toInt32(v int) int32 {
	//nolint:gosec // G115: 値は 32bit 整数幅で永続化される値でありオーバーフローしません
	return int32(v)
}

// intPtrToInt32Ptr は、ユースケースの DTO の *int をレスポンスの *int32 へ変換します（nil はそのまま nil）。
func intPtrToInt32Ptr(v *int) *int32 {
	if v == nil {
		return nil
	}
	i := toInt32(*v)
	return &i
}
