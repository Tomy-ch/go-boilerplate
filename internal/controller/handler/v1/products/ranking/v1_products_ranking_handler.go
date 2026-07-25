//go:generate oapi-codegen --include-tags=v1/products/ranking --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/products/ranking --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package productranking は、/v1/products/ranking エンドポイントに関連するハンドラを提供します。
package productranking

import (
	"context"

	"go-boilerplate/internal/controller/handler/v1/products/ranking/gen"
	"go-boilerplate/internal/observability"
	rankinguc "go-boilerplate/internal/usecase/product/ranking"

	"github.com/labstack/echo/v4"
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

// GetProductsRanking は、購入明細を集計した商品売上ランキングを販売数量の降順で返します。認証不要の公開エンドポイントです。
func (s *server) GetProductsRanking(
	ctx context.Context, request gen.GetProductsRankingRequestObject,
) (gen.GetProductsRankingResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	view, err := s.uc.GetProductsRanking(ctx, rankinguc.GetRankingParams{
		Period: periodParam(request.Params.Period),
		Limit:  limitParam(request.Params.Limit),
	})
	if err != nil {
		return nil, err
	}

	return gen.GetProductsRanking200JSONResponse(toProductRankingResponse(view)), nil
}

// periodParam は、任意指定の period クエリパラメータを usecase 入力の文字列へ変換します。未指定は空文字（全期間）として扱います。
func periodParam(period *gen.GetProductsRankingParamsPeriod) string {
	if period == nil {
		return ""
	}
	return string(*period)
}

// limitParam は、任意指定の limit クエリパラメータを usecase 入力の件数へ変換します。未指定は 0（既定件数）として扱います。
func limitParam(limit *int) int {
	if limit == nil {
		return 0
	}
	return *limit
}

// toProductRankingResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toProductRankingResponse(view rankinguc.RankingView) gen.ProductRankingResponse {
	items := make([]gen.ProductRankingItem, len(view.Rankings))
	for i, r := range view.Rankings {
		items[i] = gen.ProductRankingItem{
			ProductId:    r.ProductID.ToPrimitive(),
			Name:         r.Name,
			Price:        r.Price.String(),
			SoldQuantity: r.SoldQuantity,
		}
	}
	return gen.ProductRankingResponse{Rankings: items}
}
