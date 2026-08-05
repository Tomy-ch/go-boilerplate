//go:generate oapi-codegen --include-tags=v1/purchases/detail --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/purchases/detail --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package detail は、GET /v1/purchases/{purchaseId} エンドポイントに関連するハンドラを提供します。
package detail

import (
	"context"

	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/purchases/detail/gen"
	"go-boilerplate/internal/observability"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
)

type server struct {
	tracer observability.LayerTracer
	uc     purchaseuc.Usecase
}

// BindHandler は、購入詳細のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc purchaseuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetPurchasesDetail は、本人の購入 1 件を明細（商品名込み）とともに取得します。認証必須で、他人の購入・不存在は
// いずれも 404 で存在を秘匿します。
func (s *server) GetPurchasesDetail(
	ctx context.Context,
	request gen.GetPurchasesDetailRequestObject,
) (gen.GetPurchasesDetailResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	view, err := s.uc.GetPurchaseDetail(ctx, &authn, conv.UUID(request.PurchaseId))
	if err != nil {
		return nil, err
	}

	res, err := toPurchaseGetDetailResponse(view)
	if err != nil {
		return nil, err
	}
	return gen.GetPurchasesDetail200JSONResponse(res), nil
}

// toPurchaseGetDetailResponse は、購入詳細の取得 DTO を HTTP レスポンスへ変換します。
// 数量が int32 に収まらない場合はエラーを返します。
func toPurchaseGetDetailResponse(v purchaseuc.PurchaseGetDetailView) (gen.PurchaseGetDetailResponse, error) {
	details := make([]gen.PurchaseDetailItemResponse, len(v.Details))
	for i, d := range v.Details {
		quantity, err := safecast.IntToInt32(d.Quantity)
		if err != nil {
			return gen.PurchaseGetDetailResponse{}, xerrors.Wrap(err, "invalid purchase detail quantity")
		}
		details[i] = gen.PurchaseDetailItemResponse{
			ProductId:   d.ProductID.ToPrimitive(),
			ProductName: d.ProductName,
			Quantity:    quantity,
			UnitPrice:   d.UnitPrice.String(),
		}
	}

	return gen.PurchaseGetDetailResponse{
		Id:     v.ID.ToPrimitive(),
		Code:   v.Code,
		UserId: v.UserID.ToPrimitive(),
		Status: gen.PurchaseStatusRef{
			Id:   v.StatusID.ToPrimitive(),
			Name: v.StatusName,
		},
		SubtotalAmount: v.SubtotalAmount,
		TaxAmount:      v.TaxAmount,
		ShippingFee:    v.ShippingFee,
		TotalAmount:    v.TotalAmount,
		Details:        details,
		OrderedAt:      v.OrderedAt,
		PaidAt:         v.PaidAt,
		CanceledAt:     v.CanceledAt,
	}, nil
}
