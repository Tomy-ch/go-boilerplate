//go:generate oapi-codegen --include-tags=v1/products/ranking --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/products/ranking --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package productranking は、/v1/products/ranking エンドポイントに関連するハンドラを提供します。
package productranking

import (
	"context"

	"go-boilerplate/internal/controller/handler/v1/products/ranking/gen"
	"go-boilerplate/internal/observability"
	rankinguc "go-boilerplate/internal/usecase/product/ranking"
	"go-boilerplate/internal/usecase/tools/timewindow"

	"github.com/labstack/echo/v5"
)

type server struct {
	tracer observability.LayerTracer
	uc     rankinguc.Usecase
}

// BindHandler は、商品売上ランキングのハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc rankinguc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetProductsRankingQuantity は、購入明細を集計した商品ランキングを販売数量の降順で返します。認証不要の公開エンドポイントです。
func (s *server) GetProductsRankingQuantity(
	ctx context.Context, request gen.GetProductsRankingQuantityRequestObject,
) (gen.GetProductsRankingQuantityResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	window, err := timewindow.New(timewindow.Bounds{
		After:  request.Params.OrderedAfter,
		Before: request.Params.OrderedBefore,
	})
	if err != nil {
		return nil, err
	}

	view, err := s.uc.GetQuantityRanking(ctx, rankinguc.GetRankingParams{
		Window: window,
		Limit:  limitParam(request.Params.Limit),
	})
	if err != nil {
		return nil, err
	}

	return gen.GetProductsRankingQuantity200JSONResponse(toQuantityRankingResponse(view)), nil
}

// GetProductsRankingAmount は、購入明細を集計した商品ランキングを売上金額の降順で返します。認証不要の公開エンドポイントです。
func (s *server) GetProductsRankingAmount(
	ctx context.Context, request gen.GetProductsRankingAmountRequestObject,
) (gen.GetProductsRankingAmountResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	window, err := timewindow.New(timewindow.Bounds{
		After:  request.Params.OrderedAfter,
		Before: request.Params.OrderedBefore,
	})
	if err != nil {
		return nil, err
	}

	view, err := s.uc.GetAmountRanking(ctx, rankinguc.GetRankingParams{
		Window: window,
		Limit:  limitParam(request.Params.Limit),
	})
	if err != nil {
		return nil, err
	}

	return gen.GetProductsRankingAmount200JSONResponse(toAmountRankingResponse(view)), nil
}

// limitParam は、任意指定の limit クエリパラメータを usecase 入力の件数へ変換します。未指定は 0（既定件数）として扱います。
func limitParam(limit *int) int {
	if limit == nil {
		return 0
	}
	return *limit
}

// toQuantityRankingResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toQuantityRankingResponse(view rankinguc.QuantityRankingView) gen.ProductQuantityRankingResponse {
	items := make([]gen.ProductQuantityRankingItem, len(view.Rankings))
	for i, r := range view.Rankings {
		items[i] = gen.ProductQuantityRankingItem{
			ProductId:    r.ProductID.ToPrimitive(),
			Name:         r.Name,
			Price:        r.Price.String(),
			SoldQuantity: r.SoldQuantity,
		}
	}
	return gen.ProductQuantityRankingResponse{Rankings: items}
}

// toAmountRankingResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toAmountRankingResponse(view rankinguc.AmountRankingView) gen.ProductAmountRankingResponse {
	items := make([]gen.ProductAmountRankingItem, len(view.Rankings))
	for i, r := range view.Rankings {
		items[i] = gen.ProductAmountRankingItem{
			ProductId:   r.ProductID.ToPrimitive(),
			Name:        r.Name,
			Price:       r.Price.String(),
			SalesAmount: r.SalesAmount.String(),
		}
	}
	return gen.ProductAmountRankingResponse{Rankings: items}
}
