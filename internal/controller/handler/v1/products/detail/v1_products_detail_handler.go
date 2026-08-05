//go:generate oapi-codegen --include-tags=v1/products/detail --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/products/detail --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package detail は、/v1/products/{productId} エンドポイントに関連するハンドラを提供します。
package detail

import (
	"context"

	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/products/detail/gen"
	"go-boilerplate/internal/observability"
	productuc "go-boilerplate/internal/usecase/product"
	"go-boilerplate/pkg/patch"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
	"github.com/oapi-codegen/nullable"
)

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

	res, err := toProductResponse(dto)
	if err != nil {
		return nil, err
	}

	return gen.GetProductsDetail200JSONResponse(res), nil
}

// PatchProductsDetail は、指定された UUID に該当する商品の属性を部分更新します。
// 送信されたフィールドのみを更新し、null 指定はクリアとして扱います。
func (s *server) PatchProductsDetail(
	ctx context.Context,
	request gen.PatchProductsDetailRequestObject,
) (gen.PatchProductsDetailResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	id := conv.UUID(request.ProductId)

	params := productuc.UpdateProductParams{
		Version:               int(request.Body.Version),
		Name:                  request.Body.Name,
		Price:                 request.Body.Price,
		Quantity:              ptr.Map(request.Body.Quantity, func(v int32) int { return int(v) }),
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

	res, err := toProductResponse(dto)
	if err != nil {
		return nil, err
	}

	return gen.PatchProductsDetail200JSONResponse(res), nil
}

// PatchProductsStock は、指定された UUID に該当する商品の在庫を増減します。
// delta は正で補充、負で差し引きを表し、増減後の在庫が負になる要求はユースケースが 422 で拒否します。
func (s *server) PatchProductsStock(
	ctx context.Context,
	request gen.PatchProductsStockRequestObject,
) (gen.PatchProductsStockResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	id := conv.UUID(request.ProductId)

	params := productuc.UpdateProductStockParams{Delta: int(request.Body.Delta)}

	dto, err := s.uc.UpdateProductStock(ctx, &authn, id, params)
	if err != nil {
		return nil, err
	}

	res, err := toProductResponse(dto)
	if err != nil {
		return nil, err
	}

	return gen.PatchProductsStock200JSONResponse(res), nil
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

// toProductResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
// 在庫数・在庫警告閾値・バージョンが int32 に収まらない場合はエラーを返します。
func toProductResponse(dto productuc.ProductView) (gen.ProductResponse, error) {
	quantity, err := safecast.IntToInt32(dto.Quantity)
	if err != nil {
		return gen.ProductResponse{}, xerrors.Wrap(err, "invalid product quantity")
	}

	threshold, err := safecast.IntPtrToInt32Ptr(dto.StockWarningThreshold)
	if err != nil {
		return gen.ProductResponse{}, xerrors.Wrap(err, "invalid product stock warning threshold")
	}

	version, err := safecast.IntToInt32(dto.Version)
	if err != nil {
		return gen.ProductResponse{}, xerrors.Wrap(err, "invalid product version")
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
		Version:     version,
	}, nil
}
