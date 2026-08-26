package products

import (
	"context"

	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/products/gen"
	productuc "go-boilerplate/internal/usecase/product"
	"go-boilerplate/pkg/ptr"
)

// PostProducts は、admin が商品を作成し、作成した商品を返します。認証必須です。
func (s *server) PostProducts(ctx context.Context, request gen.PostProductsRequestObject) (gen.PostProductsResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	view, err := s.uc.CreateProduct(ctx, &authn, productuc.CreateProductParams{
		Name:                  request.Body.Name,
		Description:           request.Body.Description,
		Price:                 request.Body.Price,
		Quantity:              int(request.Body.Quantity),
		StockWarningThreshold: ptr.Map(request.Body.StockWarningThreshold, func(v int32) int { return int(v) }),
		CategoryID:            conv.UUID(request.Body.CategoryId),
		StatusID:              conv.UUID(request.Body.StatusId),
		PublishedAt:           request.Body.PublishedAt,
		Images:                toProductImageParams(ptr.Deref(request.Body.Images, nil)),
	})
	if err != nil {
		return nil, err
	}

	res, err := toProductResponse(view)
	if err != nil {
		return nil, err
	}

	return gen.PostProducts201JSONResponse(res), nil
}
