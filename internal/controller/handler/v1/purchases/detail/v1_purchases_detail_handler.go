//go:generate oapi-codegen --include-tags=v1/purchases/detail --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/purchases/detail --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package detail は、GET /v1/purchases/{purchaseId} エンドポイントに関連するハンドラを提供します。
package detail

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/purchases/detail/gen"
	"go-boilerplate/internal/observability"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v4"
)

// ErrUnauthenticatedUser は、認証ユーザー情報が取得できない場合のエラーです。
var ErrUnauthenticatedUser = xerrors.Wrap(apperror.ErrUnauthenticated, "requires authenticated user")

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

	authn, ok := ctxhelper.GetAuthn(ctx)
	if !ok {
		return nil, ErrUnauthenticatedUser
	}

	view, err := s.uc.GetPurchaseDetail(ctx, &authn, conv.UUID(request.PurchaseId))
	if err != nil {
		return nil, err
	}

	return gen.GetPurchasesDetail200JSONResponse(toPurchaseGetDetailResponse(view)), nil
}

// toPurchaseGetDetailResponse は、購入詳細の取得 DTO を HTTP レスポンスへ変換します。
func toPurchaseGetDetailResponse(v purchaseuc.PurchaseGetDetailView) gen.PurchaseGetDetailResponse {
	details := make([]gen.PurchaseDetailItemResponse, len(v.Details))
	for i, d := range v.Details {
		details[i] = gen.PurchaseDetailItemResponse{
			ProductId:   d.ProductID.ToPrimitive(),
			ProductName: d.ProductName,
			Quantity:    toInt32(d.Quantity),
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
	}
}

// toInt32 は、ドメイン DTO の int をレスポンスの int32 へ変換します。
// 値は int32 の DB 列（quantity）由来のため範囲に収まります。
func toInt32(v int) int32 {
	//nolint:gosec // G115: 値は int32 の DB 列由来でありオーバーフローしません
	return int32(v)
}
