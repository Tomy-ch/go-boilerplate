package products

import (
	"context"

	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/products/gen"
	productuc "go-boilerplate/internal/usecase/product"
)

// PostProducts は、admin が商品を作成し、作成した商品を返します。認証必須です。
// リクエストの詰め替えのみを行い、価格・在庫・マスタ整合性などの検証はユースケースが担います。
func (s *server) PostProducts(ctx context.Context, request gen.PostProductsRequestObject) (gen.PostProductsResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, ok := ctxhelper.GetAuthn(ctx)
	if !ok {
		return nil, ErrUnauthenticatedUser
	}

	view, err := s.uc.CreateProduct(ctx, &authn, productuc.CreateProductParams{
		Name:                  request.Body.Name,
		Description:           request.Body.Description,
		Price:                 request.Body.Price,
		Quantity:              int(request.Body.Quantity),
		StockWarningThreshold: int32PtrToIntPtr(request.Body.StockWarningThreshold),
		CategoryID:            conv.UUID(request.Body.CategoryId),
		StatusID:              conv.UUID(request.Body.StatusId),
		PublishedAt:           request.Body.PublishedAt,
		ImagePath:             request.Body.ImagePath,
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

// int32PtrToIntPtr は、リクエストの *int32 をユースケース DTO の *int へ変換します（nil はそのまま nil）。
func int32PtrToIntPtr(v *int32) *int {
	if v == nil {
		return nil
	}
	i := int(*v)
	return &i
}
