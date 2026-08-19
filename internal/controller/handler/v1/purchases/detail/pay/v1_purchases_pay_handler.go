//go:generate oapi-codegen --include-tags=v1/purchases/detail/pay --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/purchases/detail/pay --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package pay は、PATCH /v1/purchases/{purchaseCode}/pay エンドポイントに関連するハンドラを提供します。
package pay

import (
	"context"
	"time"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/purchases/detail/pay/gen"
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

// BindHandler は、購入支払いのハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc purchaseuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// PatchPurchasesPay は、本人の購入を支払い済みへ遷移させます。認証必須。404: 不存在 / 他人の購入。409: 不正遷移
// （詳細は docs/spec/purchase/usecase.md § PATCH 支払い を参照）。
func (s *server) PatchPurchasesPay(
	ctx context.Context,
	request gen.PatchPurchasesPayRequestObject,
) (gen.PatchPurchasesPayResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	userID, err := ctxhelper.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}

	view, err := s.uc.PayPurchase(ctx, purchaseuc.PayPurchaseParams{
		PurchaseCode: request.PurchaseCode,
		UserID:       userID,
	})
	if err != nil {
		return nil, err
	}

	res, err := toPayResponse(view)
	if err != nil {
		return nil, err
	}
	return gen.PatchPurchasesPay200JSONResponse(res), nil
}

// toPayResponse は、支払い後の購入 DTO を HTTP レスポンスへ変換します。
// 数量が int32 に収まらない場合はエラーを返します。
func toPayResponse(v purchaseuc.PayPurchaseView) (gen.PurchasePayResponse, error) {
	details := make([]gen.PurchaseDetailResponse, len(v.Details))
	for i, d := range v.Details {
		quantity, err := safecast.IntToInt32(d.Quantity)
		if err != nil {
			return gen.PurchasePayResponse{}, xerrors.Wrap(err, "invalid purchase detail quantity")
		}
		details[i] = gen.PurchaseDetailResponse{
			ProductId: d.ProductID.ToPrimitive(),
			Quantity:  quantity,
			UnitPrice: d.UnitPrice.String(),
		}
	}

	return gen.PurchasePayResponse{
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
		PaidAt:         ptr.Deref(v.PaidAt, time.Time{}),
	}, nil
}
