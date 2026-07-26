//go:generate oapi-codegen --include-tags=v1/purchases/detail/cancel --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/purchases/detail/cancel --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package cancel は、PATCH /v1/purchases/{purchaseId}/cancel エンドポイントに関連するハンドラを提供します。
package cancel

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/purchases/detail/cancel/gen"
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

// BindHandler は、購入キャンセルのハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc purchaseuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// PatchPurchasesCancel は、本人の購入をキャンセルします。認証必須で、他人の購入・不存在は 404 で
// 存在を秘匿し、不正遷移（完了・キャンセル済み・発送済み・配達済み）は 409 を返します。
func (s *server) PatchPurchasesCancel(
	ctx context.Context,
	request gen.PatchPurchasesCancelRequestObject,
) (gen.PatchPurchasesCancelResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, ok := ctxhelper.GetAuthn(ctx)
	if !ok {
		return nil, ErrUnauthenticatedUser
	}
	userID, err := authn.UserID()
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to get user ID from authenticator")
	}

	view, err := s.uc.CancelPurchase(ctx, purchaseuc.CancelPurchaseParams{
		PurchaseID: conv.UUID(request.PurchaseId),
		UserID:     userID,
	})
	if err != nil {
		return nil, err
	}

	return gen.PatchPurchasesCancel200JSONResponse(toCancelResponse(view)), nil
}

// toCancelResponse は、キャンセル後の購入 DTO を HTTP レスポンスへ変換します。
func toCancelResponse(v purchaseuc.CancelPurchaseView) gen.PurchaseCancelResponse {
	details := make([]gen.PurchaseDetailResponse, len(v.Details))
	for i, d := range v.Details {
		details[i] = gen.PurchaseDetailResponse{
			ProductId: d.ProductID.ToPrimitive(),
			Quantity:  toInt32(d.Quantity),
			UnitPrice: d.UnitPrice.String(),
		}
	}

	return gen.PurchaseCancelResponse{
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
		CanceledAt:     canceledAt(v.CanceledAt),
	}
}

// canceledAt は、キャンセル日時（*time.Time）をレスポンスの time.Time へ変換します。
// キャンセル成功時は常に非 nil ですが、防御的に nil はゼロ値へ倒します。
func canceledAt(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// toInt32 は、ユースケースの DTO の int をレスポンスの int32 へ変換します。
// 値は 32bit 整数幅で永続化される購入数量由来のため範囲に収まります。
func toInt32(v int) int32 {
	//nolint:gosec // G115: 値は 32bit 整数幅で永続化される値でありオーバーフローしません
	return int32(v)
}
