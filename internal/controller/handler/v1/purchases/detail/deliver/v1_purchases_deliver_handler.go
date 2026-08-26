//go:generate oapi-codegen --include-tags=v1/purchases/detail/deliver --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/purchases/detail/deliver --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package deliver は、PATCH /v1/purchases/{purchaseCode}/deliver エンドポイントに関連するハンドラを提供します。
package deliver

import (
	"context"
	"time"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/purchases/detail/deliver/gen"
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

// BindHandler は、購入配達完了のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc purchaseuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// PatchPurchasesDeliver は、購入を配達済みへ遷移させます。認証必須・admin 限定（非 admin は 403）。404: 不存在。409: 不正遷移
// （詳細は docs/spec/purchase/usecase.md § PATCH 配達完了 を参照）。
func (s *server) PatchPurchasesDeliver(
	ctx context.Context,
	request gen.PatchPurchasesDeliverRequestObject,
) (gen.PatchPurchasesDeliverResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	view, err := s.uc.DeliverPurchase(ctx, &authn, request.PurchaseCode)
	if err != nil {
		return nil, err
	}

	res, err := toDeliverResponse(view)
	if err != nil {
		return nil, err
	}
	return gen.PatchPurchasesDeliver200JSONResponse(res), nil
}

// toDeliverResponse は、配達完了後の購入 DTO を HTTP レスポンスへ変換します。
// 数量が int32 に収まらない場合はエラーを返します。
func toDeliverResponse(v purchaseuc.DeliverPurchaseView) (gen.PurchaseDeliverResponse, error) {
	details := make([]gen.PurchaseDetailResponse, len(v.Details))
	for i, d := range v.Details {
		quantity, err := safecast.IntToInt32(d.Quantity)
		if err != nil {
			return gen.PurchaseDeliverResponse{}, xerrors.Wrap(err, "invalid purchase detail quantity")
		}
		details[i] = gen.PurchaseDetailResponse{
			ProductId: d.ProductID.ToPrimitive(),
			Quantity:  quantity,
			UnitPrice: d.UnitPrice.String(),
		}
	}

	return gen.PurchaseDeliverResponse{
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
		DeliveredAt:    ptr.Deref(v.DeliveredAt, time.Time{}),
	}, nil
}
