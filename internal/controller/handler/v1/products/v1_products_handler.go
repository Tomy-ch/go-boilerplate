//go:generate oapi-codegen --include-tags=v1/products --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/products --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package products は、/v1/products エンドポイントに関連するハンドラを提供します。
package products

import (
	"context"

	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/handler/v1/products/gen"
	"go-boilerplate/internal/observability"
	productuc "go-boilerplate/internal/usecase/product"
	"go-boilerplate/internal/usecase/tools/paging"

	"github.com/labstack/echo/v4"
)

// sortPublishedAtAsc は、公開日時の昇順を表す sort パラメータ値です。これ以外（未指定を含む）は降順として扱います。
const sortPublishedAtAsc = "publishedAt"

type server struct {
	tracer observability.LayerTracer
	uc     productuc.Usecase
}

// BindHandler は、商品一覧のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc productuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetProducts は、公開済み商品を公開日時順（cursor ページネーション）で取得します。認証不要の公開エンドポイントです。
func (s *server) GetProducts(ctx context.Context, request gen.GetProductsRequestObject) (gen.GetProductsResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	cursor, err := paging.NewCursor(request.Params.After, request.Params.First)
	if err != nil {
		return nil, err
	}

	list, err := s.uc.ListProducts(ctx, productuc.ListProductsParams{
		Cursor:     cursor,
		CategoryID: conv.UUIDPtr(request.Params.CategoryId),
		StatusID:   conv.UUIDPtr(request.Params.StatusId),
		Keyword:    request.Params.Keyword,
		Ascending:  isAscending(request.Params.Sort),
	})
	if err != nil {
		return nil, err
	}

	products := make([]gen.ProductResponse, len(list.Items))
	for i, dto := range list.Items {
		products[i] = toProductResponse(dto)
	}

	return gen.GetProducts200JSONResponse(gen.ProductListResponse{
		Products:   products,
		NextCursor: list.NextCursor,
		HasNext:    list.NextCursor != nil,
	}), nil
}

// isAscending は、sort パラメータを昇順フラグへ変換します。publishedAt のみ昇順、それ以外（既定の -publishedAt / 未指定）は降順です。
func isAscending(sort *gen.GetProductsParamsSort) bool {
	return sort != nil && string(*sort) == sortPublishedAtAsc
}

// toProductResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toProductResponse(dto productuc.ProductView) gen.ProductResponse {
	return gen.ProductResponse{
		Id:                    dto.ID.ToPrimitive(),
		Name:                  dto.Name,
		Description:           dto.Description,
		Price:                 toInt32(dto.Price),
		Quantity:              toInt32(dto.Quantity),
		StockWarningThreshold: intPtrToInt32Ptr(dto.StockWarningThreshold),
		StatusId:              dto.StatusID.ToPrimitive(),
		CategoryId:            dto.CategoryID.ToPrimitive(),
		PublishedAt:           dto.PublishedAt,
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
