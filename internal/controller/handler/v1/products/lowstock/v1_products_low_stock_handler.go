//go:generate oapi-codegen --include-tags=v1/products/low-stock --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/products/low-stock --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package lowstock は、/v1/products/low-stock エンドポイントに関連するハンドラを提供します。
package lowstock

import (
	"context"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/products/lowstock/gen"
	"go-boilerplate/internal/observability"
	productuc "go-boilerplate/internal/usecase/product"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
)

type server struct {
	tracer observability.LayerTracer
	uc     productuc.Usecase
}

// BindHandler は、在庫僅少商品一覧のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc productuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetProductsLowStock は、在庫が在庫警告閾値以下まで減った商品を在庫の少ない順に取得します。
func (s *server) GetProductsLowStock(
	ctx context.Context, request gen.GetProductsLowStockRequestObject,
) (gen.GetProductsLowStockResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	view, err := s.uc.ListLowStockProducts(ctx, &authn, productuc.ListLowStockProductsParams{
		Limit: limitParam(request.Params.Limit),
	})
	if err != nil {
		return nil, err
	}

	products := make([]gen.ProductResponse, len(view.Items))
	for i, dto := range view.Items {
		res, err := toProductResponse(dto)
		if err != nil {
			return nil, err
		}
		products[i] = res
	}

	return gen.GetProductsLowStock200JSONResponse(gen.ProductLowStockResponse{Products: products}), nil
}

// limitParam は、任意指定の limit クエリパラメータを usecase 入力の件数へ変換します。未指定は 0（既定件数）として扱います。
func limitParam(limit *int) int {
	if limit == nil {
		return 0
	}
	return *limit
}

// toProductImageItems は、商品画像の DTO を HTTP レスポンスへ変換します。
// 表示順が int32 に収まらない場合はエラーを返します。
func toProductImageItems(dtos []productuc.ProductImageItemView) ([]gen.ProductImageItem, error) {
	items := make([]gen.ProductImageItem, len(dtos))
	for i, dto := range dtos {
		sortKey, err := safecast.IntToInt32(dto.SortKey)
		if err != nil {
			return nil, xerrors.Wrap(err, "invalid product image sort key")
		}
		items[i] = gen.ProductImageItem{ImagePath: dto.Path, SortKey: sortKey}
	}
	return items, nil
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

	images, err := toProductImageItems(dto.Images)
	if err != nil {
		return gen.ProductResponse{}, err
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
		Images:      images,
		Version:     version,
	}, nil
}
