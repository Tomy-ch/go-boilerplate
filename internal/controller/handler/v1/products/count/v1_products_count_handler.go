//go:generate oapi-codegen --include-tags=v1/products/count --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/products/count --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package productcount は、/v1/products/count エンドポイントに関連するハンドラを提供します。
package productcount

import (
	"context"

	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/handler/v1/products/count/gen"
	"go-boilerplate/internal/observability"
	productuc "go-boilerplate/internal/usecase/product"

	"github.com/labstack/echo/v5"
)

type server struct {
	tracer observability.LayerTracer
	uc     productuc.Usecase
}

// BindHandler は、商品検索の一致件数ハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc productuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{tracer: tf.Controller(), uc: uc}, nil))
}

// GetProductsCount は、検索条件に一致する公開商品の件数を取得します。
func (s *server) GetProductsCount(
	ctx context.Context, request gen.GetProductsCountRequestObject,
) (gen.GetProductsCountResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	view, err := s.uc.CountProducts(ctx, productuc.CountProductsParams{
		CategoryID:  conv.UUIDPtr(request.Params.CategoryId),
		StatusID:    conv.UUIDPtr(request.Params.StatusId),
		Keyword:     request.Params.Keyword,
		MinPrice:    request.Params.MinPrice,
		MaxPrice:    request.Params.MaxPrice,
		MinQuantity: request.Params.MinQuantity,
		MaxQuantity: request.Params.MaxQuantity,
	})
	if err != nil {
		return nil, err
	}

	return gen.GetProductsCount200JSONResponse(gen.ProductCountResponse{Count: view.Count}), nil
}
