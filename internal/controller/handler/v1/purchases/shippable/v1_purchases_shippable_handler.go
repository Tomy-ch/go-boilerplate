//go:generate oapi-codegen --include-tags=v1/purchases/shippable --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/purchases/shippable --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package shippable は、GET /v1/purchases/shippable エンドポイントに関連するハンドラを提供します。
package shippable

import (
	"context"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/purchases/shippable/gen"
	"go-boilerplate/internal/observability"
	purchaseuc "go-boilerplate/internal/usecase/purchase"

	"github.com/labstack/echo/v5"
)

type server struct {
	tracer observability.LayerTracer
	uc     purchaseuc.Usecase
}

// BindHandler は、発送待ち購入のまとめ発送一覧のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc purchaseuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetPurchasesShippable は、発送可能な購入を、まとめて発送してよい組に分けて取得します。
func (s *server) GetPurchasesShippable(
	ctx context.Context, request gen.GetPurchasesShippableRequestObject,
) (gen.GetPurchasesShippableResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	view, err := s.uc.ListShippablePurchases(ctx, &authn, purchaseuc.ListShippablePurchasesParams{
		Limit: limitParam(request.Params.Limit),
	})
	if err != nil {
		return nil, err
	}

	groups := make([]gen.PurchaseDispatchGroupResponse, len(view.Groups))
	for i, dto := range view.Groups {
		groups[i] = toDispatchGroupResponse(dto)
	}

	return gen.GetPurchasesShippable200JSONResponse(gen.PurchaseShippableResponse{Groups: groups}), nil
}

// limitParam は、任意指定の limit クエリパラメータを usecase 入力の件数へ変換します。未指定は 0（既定件数）として扱います。
func limitParam(limit *int) int {
	if limit == nil {
		return 0
	}
	return *limit
}

// toDispatchGroupResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toDispatchGroupResponse(dto purchaseuc.DispatchGroupView) gen.PurchaseDispatchGroupResponse {
	purchases := make([]gen.PurchaseShippableItemResponse, len(dto.Purchases))
	for i, p := range dto.Purchases {
		purchases[i] = gen.PurchaseShippableItemResponse{
			Code:        p.Code,
			TotalAmount: int64(p.TotalAmount),
			OrderedAt:   p.OrderedAt,
		}
	}

	return gen.PurchaseDispatchGroupResponse{
		UserId:    dto.UserID.ToPrimitive(),
		Purchases: purchases,
	}
}
