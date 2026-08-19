//go:generate oapi-codegen --include-tags=v1/purchases --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/purchases --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package purchases は、/v1/purchases エンドポイントに関連するハンドラを提供します。
package purchases

import (
	"context"
	"net/http"

	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/purchases/gen"
	idempotencymw "go-boilerplate/internal/controller/httpstack/idempotency"
	"go-boilerplate/internal/observability"
	checkoutuc "go-boilerplate/internal/usecase/checkout"
	"go-boilerplate/internal/usecase/idempotency"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	"go-boilerplate/internal/usecase/purchase/period"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
)

type server struct {
	tracer   observability.LayerTracer
	uc       purchaseuc.Usecase
	checkout checkoutuc.Usecase
	idem     idempotency.Deps
}

// BindHandler は、購入作成のハンドラを Echo に登録します。冪等ミドルウェアを併用します。
func BindHandler(
	e *echo.Echo,
	tf observability.TracerFactory,
	uc purchaseuc.Usecase,
	checkout checkoutuc.Usecase,
	idem idempotency.Deps,
) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer:   tf.Controller(),
		uc:       uc,
		checkout: checkout,
		idem:     idem,
	}, []gen.StrictMiddlewareFunc{idempotencymw.StrictMiddleware[gen.StrictHandlerFunc]()}))
}

// GetPurchases は、認証主体（自分）の購入履歴を注文日時降順（cursor ページネーション）で取得します。認証必須です。
func (s *server) GetPurchases(ctx context.Context, request gen.GetPurchasesRequestObject) (gen.GetPurchasesResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	userID, err := ctxhelper.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}

	cursor, err := paging.NewCursor(request.Params.After, request.Params.First)
	if err != nil {
		return nil, err
	}

	list, err := s.uc.GetPurchases(ctx, userID, cursor, toPeriodSpec(request.Params))
	if err != nil {
		return nil, err
	}

	items := make([]gen.PurchaseSummaryResponse, len(list.Items))
	for i, v := range list.Items {
		items[i] = toPurchaseSummaryResponse(v)
	}

	return gen.GetPurchases200JSONResponse(gen.PurchaseListResponse{
		Items:      items,
		NextCursor: list.NextCursor,
		HasNext:    list.NextCursor != nil,
	}), nil
}

// toPeriodSpec は、期間絞り込みのクエリパラメータをユースケースの期間指定へ変換します。
// 区分名の妥当性は OpenAPI の enum が、区分ごとの必須指定の有無はユースケースが判定します。
func toPeriodSpec(params gen.GetPurchasesParams) period.Spec {
	spec := period.Spec{
		From:  conv.DatePtr(params.From),
		To:    conv.DatePtr(params.To),
		Month: params.Month,
		Days:  ptr.Map(params.Days, func(v int32) int { return int(v) }),
	}
	if params.Period != nil {
		spec.Kind = period.Kind(*params.Period)
	}
	return spec
}

// toPurchaseSummaryResponse は、購入履歴一覧のユースケース DTO を HTTP レスポンスへ変換します。
func toPurchaseSummaryResponse(v purchaseuc.PurchaseSummaryView) gen.PurchaseSummaryResponse {
	return gen.PurchaseSummaryResponse{
		Code:        v.Code,
		TotalAmount: int64(v.TotalAmount),
		Status: gen.PurchaseStatusRef{
			Id:   v.StatusID.ToPrimitive(),
			Code: int64(v.StatusCode),
			Name: v.StatusName,
		},
		FirstItemName: v.FirstItemName,
		ItemCount:     int64(v.ItemCount),
		OrderedAt:     v.OrderedAt,
	}
}

// PostPurchases は、明細から購入を作成します。認証必須で、同一 Idempotency-Key の再送には
// 最初の結果を再生して返します。
func (s *server) PostPurchases(ctx context.Context, request gen.PostPurchasesRequestObject) (gen.PostPurchasesResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	userID, err := ctxhelper.RequireUserID(ctx)
	if err != nil {
		return nil, err
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

	view, _, err := idempotency.Run(ctx, s.idem, http.StatusCreated, func(ctx context.Context) (checkoutuc.PurchaseView, error) {
		return s.checkout.CreatePurchase(ctx, checkoutuc.CreatePurchaseParams{
			UserID:          userID,
			Details:         details,
			DisplayCurrency: displayCurrency,
		})
	})
	if err != nil {
		return nil, err
	}

	res, err := toPurchaseResponse(view.Purchase, view.ReferenceAmount)
	if err != nil {
		return nil, err
	}
	return gen.PostPurchases201JSONResponse(res), nil
}

// toPurchaseResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
// 数量が int32 に収まらない場合はエラーを返します。
func toPurchaseResponse(v purchaseuc.PurchaseView, ref *checkoutuc.ReferenceAmountView) (gen.PurchaseResponse, error) {
	details := make([]gen.PurchaseDetailResponse, len(v.Details))
	for i, d := range v.Details {
		quantity, err := safecast.IntToInt32(d.Quantity)
		if err != nil {
			return gen.PurchaseResponse{}, xerrors.Wrap(err, "invalid purchase detail quantity")
		}
		details[i] = gen.PurchaseDetailResponse{
			ProductId: d.ProductID.ToPrimitive(),
			Quantity:  quantity,
			UnitPrice: d.UnitPrice.String(),
		}
	}

	return gen.PurchaseResponse{
		Code:            v.Code,
		UserId:          v.UserID.ToPrimitive(),
		StatusId:        v.StatusID.ToPrimitive(),
		SubtotalAmount:  int64(v.SubtotalAmount),
		TaxAmount:       int64(v.TaxAmount),
		ShippingFee:     int64(v.ShippingFee),
		TotalAmount:     int64(v.TotalAmount),
		Details:         details,
		OrderedAt:       v.OrderedAt,
		ReferenceAmount: toReferenceAmount(ref),
	}, nil
}

// toReferenceAmount は、参考換算額の DTO を HTTP レスポンスへ変換します（nil はそのまま nil）。
func toReferenceAmount(r *checkoutuc.ReferenceAmountView) *gen.ReferenceAmount {
	if r == nil {
		return nil
	}
	return &gen.ReferenceAmount{
		Currency: r.Currency,
		Amount:   r.Amount,
		Rate:     r.Rate.String(),
		RateDate: r.RateDate,
	}
}
