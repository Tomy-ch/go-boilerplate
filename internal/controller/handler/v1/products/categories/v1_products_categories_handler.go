//go:generate oapi-codegen --include-tags=v1/products/categories --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/products/categories --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package productcategories は、/v1/products/categories エンドポイントに関連するハンドラを提供します。
package productcategories

import (
	"context"

	"go-boilerplate/internal/controller/handler/v1/products/categories/gen"
	"go-boilerplate/internal/observability"
	categoryuc "go-boilerplate/internal/usecase/product/category"

	"github.com/labstack/echo/v5"
)

type server struct {
	tracer observability.LayerTracer
	uc     categoryuc.Usecase
}

// BindHandler は、商品カテゴリマスタ一覧のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc categoryuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetProductCategories は、商品カテゴリマスタの全件をマスタの表示順で返します。表示順の値は応答に含めません。
func (s *server) GetProductCategories(
	ctx context.Context, _ gen.GetProductCategoriesRequestObject,
) (gen.GetProductCategoriesResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	list, err := s.uc.ListCategories(ctx)
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
func toProductCategoryResponse(dto categoryuc.CategoryDTO) gen.ProductCategoryResponse {
	return gen.ProductCategoryResponse{
		Id:   dto.ID.ToPrimitive(),
		Code: dto.Code,
		Name: dto.Name,
	}
}
