//go:generate oapi-codegen --include-tags=v1/purchases --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/purchases --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package purchases は、/v1/purchases エンドポイントに関連するハンドラを提供します。
package purchases

import (
	"context"
	"net/http"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/purchases/gen"
	idempotencymw "go-boilerplate/internal/controller/httpstack/idempotency"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/idempotency"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v4"
)

// ErrUnauthenticatedUser は、認証ユーザー情報が取得できない場合のエラーです。
var ErrUnauthenticatedUser = xerrors.Wrap(apperror.ErrUnauthenticated, "requires authenticated user")

type server struct {
	tracer observability.LayerTracer
	uc     purchaseuc.Usecase
	idem   idempotency.Deps
}

// BindHandler は、購入作成のハンドラを Echo に登録します。冪等ミドルウェアを併用します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc purchaseuc.Usecase, idem idempotency.Deps) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
		idem:   idem,
	}, []gen.StrictMiddlewareFunc{idempotencymw.StrictMiddleware[gen.StrictHandlerFunc]()}))
}

// PostPurchases は、明細から購入を作成します。認証必須で、idempotency.Run を通して最外トランザクションを開始します。
func (s *server) PostPurchases(ctx context.Context, request gen.PostPurchasesRequestObject) (gen.PostPurchasesResponseObject, error) {
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

	details := make([]purchaseuc.DetailParam, len(request.Body.Details))
	for i, d := range request.Body.Details {
		details[i] = purchaseuc.DetailParam{
			ProductID: conv.UUID(d.ProductId),
			Quantity:  int(d.Quantity),
		}
	}

	var displayCurrency *string
	if request.Params.DisplayCurrency != nil {
		s := string(*request.Params.DisplayCurrency)
		displayCurrency = &s
	}

	view, _, err := idempotency.Run(ctx, s.idem, http.StatusCreated, func(ctx context.Context) (purchaseuc.PurchaseView, error) {
		return s.uc.CreatePurchase(ctx, purchaseuc.CreatePurchaseParams{
			UserID:          userID,
			Details:         details,
			DisplayCurrency: displayCurrency,
		})
	})
	if err != nil {
		return nil, err
	}

	return gen.PostPurchases201JSONResponse(toPurchaseResponse(view)), nil
}

// toPurchaseResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toPurchaseResponse(v purchaseuc.PurchaseView) gen.PurchaseResponse {
	details := make([]gen.PurchaseDetailResponse, len(v.Details))
	for i, d := range v.Details {
		details[i] = gen.PurchaseDetailResponse{
			ProductId: d.ProductID.ToPrimitive(),
			Quantity:  toInt32(d.Quantity),
			UnitPrice: toInt32(d.UnitPrice),
		}
	}

	return gen.PurchaseResponse{
		Id:              v.ID.ToPrimitive(),
		Code:            v.Code,
		UserId:          v.UserID.ToPrimitive(),
		StatusId:        v.StatusID.ToPrimitive(),
		SubtotalAmount:  toInt32(v.SubtotalAmount),
		TaxAmount:       toInt32(v.TaxAmount),
		ShippingFee:     toInt32(v.ShippingFee),
		TotalAmount:     toInt32(v.TotalAmount),
		Details:         details,
		OrderedAt:       v.OrderedAt,
		ReferenceAmount: toReferenceAmount(v.ReferenceAmount),
	}
}

// toReferenceAmount は、参考換算額の DTO を HTTP レスポンスへ変換します（nil はそのまま nil）。
func toReferenceAmount(r *purchaseuc.ReferenceAmountView) *gen.ReferenceAmount {
	if r == nil {
		return nil
	}
	return &gen.ReferenceAmount{
		Currency: r.Currency,
		Amount:   r.Amount,
		Rate:     r.Rate,
		RateDate: r.RateDate,
	}
}

// toInt32 は、ドメイン DTO の int をレスポンスの int32 へ変換します。
// 値は int32 の DB 列（*_amount / quantity / unit_price）由来のため範囲に収まります。
func toInt32(v int) int32 {
	//nolint:gosec // G115: 値は int32 の DB 列由来でありオーバーフローしません
	return int32(v)
}
