//go:generate oapi-codegen --include-tags=v1/purchases/detail/ship --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/purchases/detail/ship --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package ship は、PATCH /v1/purchases/{purchaseId}/ship エンドポイントに関連するハンドラを提供します。
package ship

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/purchases/detail/ship/gen"
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

// BindHandler は、購入発送のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc purchaseuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// PatchPurchasesShip は、購入を発送済みへ遷移させます。認証必須かつ admin のみ実行でき、非 admin は 403、
// 不存在は 404、二重発送・不正遷移（未払い相当・完了・キャンセル済み・配達済み）は 409 を返します。
func (s *server) PatchPurchasesShip(
	ctx context.Context,
	request gen.PatchPurchasesShipRequestObject,
) (gen.PatchPurchasesShipResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, ok := ctxhelper.GetAuthn(ctx)
	if !ok {
		return nil, ErrUnauthenticatedUser
	}

	view, err := s.uc.ShipPurchase(ctx, &authn, conv.UUID(request.PurchaseId))
	if err != nil {
		return nil, err
	}

	return gen.PatchPurchasesShip200JSONResponse(toShipResponse(view)), nil
}

// toShipResponse は、発送後の購入 DTO を HTTP レスポンスへ変換します。
func toShipResponse(v purchaseuc.ShipPurchaseView) gen.PurchaseShipResponse {
	details := make([]gen.PurchaseDetailResponse, len(v.Details))
	for i, d := range v.Details {
		details[i] = gen.PurchaseDetailResponse{
			ProductId: d.ProductID.ToPrimitive(),
			Quantity:  toInt32(d.Quantity),
			UnitPrice: d.UnitPrice.String(),
		}
	}

	return gen.PurchaseShipResponse{
		Id:     v.ID.ToPrimitive(),
		Code:   v.Code,
		UserId: v.UserID.ToPrimitive(),
		Status: gen.PurchaseStatusRef{
			Id:   v.StatusID.ToPrimitive(),
			Name: v.StatusName,
		},
		SubtotalAmount: int64(v.SubtotalAmount),
		TaxAmount:      int64(v.TaxAmount),
		ShippingFee:    int64(v.ShippingFee),
		TotalAmount:    int64(v.TotalAmount),
		Details:        details,
		OrderedAt:      v.OrderedAt,
		ShippedAt:      shippedAt(v.ShippedAt),
	}
}

// shippedAt は、発送日時（*time.Time）をレスポンスの time.Time へ変換します。
// 発送成功時は常に非 nil ですが、防御的に nil はゼロ値へ倒します。
func shippedAt(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// toInt32 は、ユースケースの DTO の int をレスポンスの int32 へ変換します。
// 値は 32bit 整数幅で永続化される購入数量由来のため範囲に収まります。
func toInt32(v int) int32 {
	//nolint:gosec // G115: 値は int32 の DB 列由来でありオーバーフローしません
	return int32(v)
}
