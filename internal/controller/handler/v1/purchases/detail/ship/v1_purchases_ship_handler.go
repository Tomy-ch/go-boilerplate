//go:generate oapi-codegen --include-tags=v1/purchases/detail/ship --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/purchases/detail/ship --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package ship は、PATCH /v1/purchases/{purchaseCode}/ship エンドポイントに関連するハンドラを提供します。
package ship

import (
	"context"
	"time"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/purchases/detail/ship/gen"
	"go-boilerplate/internal/observability"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
)

type server struct {
	tracer observability.LayerTracer
	uc     purchaseuc.Usecase
}

// BindHandler は、購入発送のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc purchaseuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// PatchPurchasesShip は、購入を発送済みへ遷移させます。認証必須・admin 限定（非 admin は 403）。404: 不存在。409: 不正遷移
// （admin 限定の理由・存在を秘匿しない理由は docs/spec/purchase/usecase.md § PATCH 発送 を参照）。
func (s *server) PatchPurchasesShip(
	ctx context.Context,
	request gen.PatchPurchasesShipRequestObject,
) (gen.PatchPurchasesShipResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	view, err := s.uc.ShipPurchase(ctx, &authn, request.PurchaseCode)
	if err != nil {
		return nil, err
	}

	res, err := toShipResponse(view)
	if err != nil {
		return nil, err
	}
	return gen.PatchPurchasesShip200JSONResponse(res), nil
}

// toShipResponse は、発送後の購入 DTO を HTTP レスポンスへ変換します。
// 数量が int32 に収まらない場合はエラーを返します。
func toShipResponse(v purchaseuc.ShipPurchaseView) (gen.PurchaseShipResponse, error) {
	details := make([]gen.PurchaseDetailResponse, len(v.Details))
	for i, d := range v.Details {
		quantity, err := safecast.IntToInt32(d.Quantity)
		if err != nil {
			return gen.PurchaseShipResponse{}, xerrors.Wrap(err, "invalid purchase detail quantity")
		}
		details[i] = gen.PurchaseDetailResponse{
			ProductId: d.ProductID.ToPrimitive(),
			Quantity:  quantity,
			UnitPrice: d.UnitPrice.String(),
		}
	}

	return gen.PurchaseShipResponse{
		Code:   v.Code,
		UserId: v.UserID.ToPrimitive(),
		Status: gen.PurchaseStatusRef{
			Id:   v.StatusID.ToPrimitive(),
			Code: int64(v.StatusCode),
			Name: v.StatusName,
		},
		SubtotalAmount: int64(v.SubtotalAmount),
		TaxAmount:      int64(v.TaxAmount),
		ShippingFee:    int64(v.ShippingFee),
		TotalAmount:    int64(v.TotalAmount),
		Details:        details,
		OrderedAt:      v.OrderedAt,
		ShippedAt:      ptr.Deref(v.ShippedAt, time.Time{}),
	}, nil
}
