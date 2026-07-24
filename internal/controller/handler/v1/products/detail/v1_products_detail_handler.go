//go:generate oapi-codegen --include-tags=v1/products/detail --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/products/detail --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package detail は、/v1/products/{productId} エンドポイントに関連するハンドラを提供します。
package detail

import (
	"context"

	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/handler/v1/products/detail/gen"
	"go-boilerplate/internal/observability"
	productuc "go-boilerplate/internal/usecase/product"

	"github.com/labstack/echo/v4"
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

	return gen.GetProductsDetail200JSONResponse(toProductResponse(dto)), nil
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
	}
}

// toInt32 は、ドメイン DTO の int をレスポンスの int32 へ変換します。
// 値は int32 の DB 列（products.price / quantity）由来のため範囲に収まります。
func toInt32(v int) int32 {
	//nolint:gosec // G115: 値は int32 の DB 列由来でありオーバーフローしません
	return int32(v)
}

// intPtrToInt32Ptr は、ドメイン DTO の *int をレスポンスの *int32 へ変換します（nil はそのまま nil）。
func intPtrToInt32Ptr(v *int) *int32 {
	if v == nil {
		return nil
	}
	i := toInt32(*v)
	return &i
}
