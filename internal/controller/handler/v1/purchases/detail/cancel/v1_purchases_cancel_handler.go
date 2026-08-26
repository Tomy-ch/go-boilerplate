//go:generate oapi-codegen --include-tags=v1/purchases/detail/cancel --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/purchases/detail/cancel --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package cancel は、PATCH /v1/purchases/{purchaseCode}/cancel エンドポイントに関連するハンドラを提供します。
package cancel

import (
	"context"
	"time"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/purchases/detail/cancel/gen"
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

// BindHandler は、購入キャンセルのハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc purchaseuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// PatchPurchasesCancel は、本人の購入をキャンセルします。認証必須。404: 不存在 / 他人の購入。409: 不正遷移
// （詳細は docs/spec/purchase/usecase.md § PATCH キャンセル を参照）。
func (s *server) PatchPurchasesCancel(
	ctx context.Context,
	request gen.PatchPurchasesCancelRequestObject,
) (gen.PatchPurchasesCancelResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	userID, err := ctxhelper.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}

	view, err := s.uc.CancelPurchase(ctx, purchaseuc.CancelPurchaseParams{
		PurchaseCode: request.PurchaseCode,
		UserID:       userID,
	})
	if err != nil {
		return nil, err
	}

	res, err := toCancelResponse(view)
	if err != nil {
		return nil, err
	}
	return gen.PatchPurchasesCancel200JSONResponse(res), nil
}

// toCancelResponse は、キャンセル後の購入 DTO を HTTP レスポンスへ変換します。
// 数量が int32 に収まらない場合はエラーを返します。
func toCancelResponse(v purchaseuc.CancelPurchaseView) (gen.PurchaseCancelResponse, error) {
	details := make([]gen.PurchaseDetailResponse, len(v.Details))
	for i, d := range v.Details {
		quantity, err := safecast.IntToInt32(d.Quantity)
		if err != nil {
			return gen.PurchaseCancelResponse{}, xerrors.Wrap(err, "invalid purchase detail quantity")
		}
		details[i] = gen.PurchaseDetailResponse{
			ProductId: d.ProductID.ToPrimitive(),
			Quantity:  quantity,
			UnitPrice: d.UnitPrice.String(),
		}
	}

	return gen.PurchaseCancelResponse{
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
		CanceledAt:     ptr.Deref(v.CanceledAt, time.Time{}),
	}, nil
}
