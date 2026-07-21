//go:generate oapi-codegen --include-tags=v1/product-categories --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/product-categories --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package productcategories は、/v1/product-categories エンドポイントに関連するハンドラを提供します。
package productcategories

import (
	"context"

	"go-boilerplate/internal/controller/handler/v1/product-categories/gen"
	"go-boilerplate/internal/observability"
	productcategoryuc "go-boilerplate/internal/usecase/product_category"

	"github.com/labstack/echo/v4"
)

type server struct {
	tracer observability.LayerTracer
	uc     productcategoryuc.Usecase
}

// BindHandler は、商品カテゴリマスタ一覧のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc productcategoryuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetProductCategories は、商品カテゴリマスタの全件を sortKey 昇順で返します。
func (s *server) GetProductCategories(
	ctx context.Context, _ gen.GetProductCategoriesRequestObject,
) (gen.GetProductCategoriesResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	list, err := s.uc.ListProductCategories(ctx)
	if err != nil {
		return nil, err
	}

	categories := make([]gen.ProductCategoryResponse, len(list))
	for i, dto := range list {
		categories[i] = toProductCategoryResponse(dto)
	}

	return gen.GetProductCategories200JSONResponse(categories), nil
}

// toProductCategoryResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toProductCategoryResponse(dto productcategoryuc.ProductCategoryDTO) gen.ProductCategoryResponse {
	return gen.ProductCategoryResponse{
		Id:      dto.ID.ToPrimitive(),
		Code:    dto.Code,
		Name:    dto.Name,
		SortKey: dto.SortKey,
	}
}
