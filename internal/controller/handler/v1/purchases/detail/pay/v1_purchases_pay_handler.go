//go:generate oapi-codegen --include-tags=v1/purchases/detail/pay --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/purchases/detail/pay --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package pay は、PATCH /v1/purchases/{purchaseId}/pay エンドポイントに関連するハンドラを提供します。
package pay

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/purchases/detail/pay/gen"
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

// BindHandler は、購入支払いのハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc purchaseuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// PatchPurchasesPay は、本人の購入を支払い済みへ遷移させます。認証必須で、他人の購入・不存在は 404 で
// 存在を秘匿し、二重支払い・不正遷移（キャンセル済み・完了・発送済み・配達済み）は 409 を返します。
func (s *server) PatchPurchasesPay(
	ctx context.Context,
	request gen.PatchPurchasesPayRequestObject,
) (gen.PatchPurchasesPayResponseObject, error) {
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

	view, err := s.uc.PayPurchase(ctx, purchaseuc.PayPurchaseParams{
		PurchaseID: conv.UUID(request.PurchaseId),
		UserID:     userID,
	})
	if err != nil {
		return nil, err
	}

	return gen.PatchPurchasesPay200JSONResponse(toPayResponse(view)), nil
}

// toPayResponse は、支払い後の購入 DTO を HTTP レスポンスへ変換します。
func toPayResponse(v purchaseuc.PayPurchaseView) gen.PurchasePayResponse {
	details := make([]gen.PurchaseDetailResponse, len(v.Details))
	for i, d := range v.Details {
		details[i] = gen.PurchaseDetailResponse{
			ProductId: d.ProductID.ToPrimitive(),
			Quantity:  toInt32(d.Quantity),
			UnitPrice: d.UnitPrice.String(),
		}
	}

	return gen.PurchasePayResponse{
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
		PaidAt:         paidAt(v.PaidAt),
	}
}

// paidAt は、支払い日時（*time.Time）をレスポンスの time.Time へ変換します。
// 支払い成功時は常に非 nil ですが、防御的に nil はゼロ値へ倒します。
func paidAt(t *time.Time) time.Time {
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
