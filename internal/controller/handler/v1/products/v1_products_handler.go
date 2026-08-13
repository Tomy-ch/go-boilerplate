//go:generate oapi-codegen --include-tags=v1/products --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/products --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package products は、/v1/products エンドポイントに関連するハンドラを提供します。
package products

import (
	"context"

	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/handler/v1/products/gen"
	"go-boilerplate/internal/observability"
	productuc "go-boilerplate/internal/usecase/product"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
)

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
		Cursor:      cursor,
		CategoryID:  conv.UUIDPtr(request.Params.CategoryId),
		StatusID:    conv.UUIDPtr(request.Params.StatusId),
		Keyword:     request.Params.Keyword,
		MinPrice:    request.Params.MinPrice,
		MaxPrice:    request.Params.MaxPrice,
		MinQuantity: request.Params.MinQuantity,
		MaxQuantity: request.Params.MaxQuantity,
		Ascending:   isAscending(request.Params.Sort),
	})
	if err != nil {
		return nil, err
	}

	products := make([]gen.ProductResponse, len(list.Items))
	for i, dto := range list.Items {
		res, err := toProductResponse(dto)
		if err != nil {
			return nil, err
		}
		products[i] = res
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
