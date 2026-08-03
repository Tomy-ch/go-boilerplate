//go:generate oapi-codegen --include-tags=v1/products/low-stock --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/products/low-stock --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package lowstock は、/v1/products/low-stock エンドポイントに関連するハンドラを提供します。
package lowstock

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/products/lowstock/gen"
	"go-boilerplate/internal/observability"
	productuc "go-boilerplate/internal/usecase/product"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
)

// ErrUnauthenticatedUser は、認証ユーザー情報が取得できない場合のエラーです。
var ErrUnauthenticatedUser = xerrors.Wrap(apperror.ErrUnauthenticated, "requires authenticated user")

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

	authn, ok := ctxhelper.GetAuthn(ctx)
	if !ok {
		return nil, ErrUnauthenticatedUser
	}

	view, err := s.uc.ListLowStockProducts(ctx, &authn, productuc.ListLowStockProductsParams{
		Limit: limitParam(request.Params.Limit),
	})
	if err != nil {
		return nil, err
	}

	products := make([]gen.ProductResponse, len(view.Items))
	for i, dto := range view.Items {
		products[i] = toProductResponse(dto)
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

// toProductResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
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
		ImagePath:   dto.ImagePath,
		Version:     toInt32(dto.Version),
	}
}

// toInt32 は、ユースケースの DTO の int をレスポンスの int32 へ変換します。
// 値は 32bit 整数幅で永続化される在庫数・在庫警告閾値・バージョン由来のため範囲に収まります。
func toInt32(v int) int32 {
	//nolint:gosec // G115: 値は 32bit 整数幅で永続化される値でありオーバーフローしません
	return int32(v)
}

// intPtrToInt32Ptr は、ユースケースの DTO の *int をレスポンスの *int32 へ変換します（nil はそのまま nil）。
func intPtrToInt32Ptr(v *int) *int32 {
	if v == nil {
		return nil
	}
	i := toInt32(*v)
	return &i
}
