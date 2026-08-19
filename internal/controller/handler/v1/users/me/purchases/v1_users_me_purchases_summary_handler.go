//go:generate oapi-codegen --include-tags=v1/users/me/purchases --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/users/me/purchases --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package usersmepurchases は、/v1/users/me/purchases 配下のエンドポイントに関連するハンドラを提供します。
package usersmepurchases

import (
	"context"

	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/users/me/purchases/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/purchase/period"
	summaryuc "go-boilerplate/internal/usecase/purchase/summary"
	"go-boilerplate/pkg/ptr"

	"github.com/labstack/echo/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type server struct {
	tracer observability.LayerTracer
	uc     summaryuc.Usecase
}

// BindHandler は、認証ユーザー自身の購入集計のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc summaryuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetUsersMePurchasesSummary は、認証コンテキストの内部 UserID に該当するユーザー自身の購入集計を取得します。
func (s *server) GetUsersMePurchasesSummary(
	ctx context.Context, request gen.GetUsersMePurchasesSummaryRequestObject,
) (gen.GetUsersMePurchasesSummaryResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	view, err := s.uc.GetPurchaseSummary(ctx, &authn, summaryuc.GetSummaryParams{
		Period:  toPeriodSpec(request.Params),
		GroupBy: toGroupKinds(request.Params.GroupBy),
	})
	if err != nil {
		return nil, err
	}

	return gen.GetUsersMePurchasesSummary200JSONResponse(toPurchaseAggregateResponse(view)), nil
}

// toPeriodSpec は、期間指定のクエリパラメータをユースケースの期間指定へ変換します。
// 区分名の妥当性は OpenAPI の enum が、区分ごとの必須指定の有無はユースケースが判定します。
func toPeriodSpec(params gen.GetUsersMePurchasesSummaryParams) period.Spec {
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

// toGroupKinds は、グループ化単位のクエリパラメータをユースケースの指定へ変換します。
// 指定順がネストの階層順を決めるため、並びはそのまま保ちます。
func toGroupKinds(groupBy *gen.PurchaseGroupByParam) []summaryuc.GroupKind {
	if groupBy == nil {
		return nil
	}
	kinds := make([]summaryuc.GroupKind, len(*groupBy))
	for i, g := range *groupBy {
		kinds[i] = summaryuc.GroupKind(g)
	}
	return kinds
}

// toPurchaseAggregateResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toPurchaseAggregateResponse(view summaryuc.SummaryView) gen.PurchaseAggregateResponse {
	breakdown := make([]gen.PurchaseStatusBreakdownResponse, len(view.StatusBreakdown))
	for i, b := range view.StatusBreakdown {
		breakdown[i] = gen.PurchaseStatusBreakdownResponse{
			Status: gen.PurchaseStatusRef{
				Id:   b.StatusID.ToPrimitive(),
				Code: int64(b.StatusCode),
				Name: b.StatusName,
			},
			Count:       b.Count,
			TotalAmount: b.TotalAmount,
		}
	}

	return gen.PurchaseAggregateResponse{
		Period:          toPurchasePeriodResponse(view.Period),
		TotalCount:      view.TotalCount,
		TotalAmount:     view.TotalAmount,
		ItemsTotal:      view.ItemsTotal.String(),
		StatusBreakdown: breakdown,
		Groups:          toGroupsResponse(view.Groups),
	}
}

// toPurchasePeriodResponse は、集計に用いた対象期間を HTTP レスポンスへ変換します。
// 期間で絞り込まなかった場合は、開始日・終了日をいずれも null として返します。
func toPurchasePeriodResponse(window period.Window) gen.PurchasePeriodResponse {
	if !window.Filtered() {
		return gen.PurchasePeriodResponse{}
	}
	from := openapi_types.Date{Time: window.From()}
	to := openapi_types.Date{Time: window.To()}
	return gen.PurchasePeriodResponse{From: &from, To: &to}
}

// toGroupsResponse は、グループ化した集計の最上位ノードを HTTP レスポンスへ変換します。
// グループ化していない場合（nil）は nil のまま返し、レスポンスに groups を含めません。
func toGroupsResponse(groups map[string]summaryuc.GroupNodeView) *map[string]gen.PurchaseGroupResponse {
	if groups == nil {
		return nil
	}
	converted := make(map[string]gen.PurchaseGroupResponse, len(groups))
	for key, node := range groups {
		converted[key] = gen.PurchaseGroupResponse{
			Name:       node.Name,
			ItemsTotal: node.ItemsTotal.String(),
			Groups:     toSubGroupsResponse(node.Groups),
		}
	}
	return &converted
}

// toSubGroupsResponse は、グループ化した集計の下位ノードを HTTP レスポンスへ変換します。
// グループ化単位は 2 つまでのため、下位ノードはこれ以上の階層を持ちません。
func toSubGroupsResponse(groups map[string]summaryuc.GroupNodeView) *map[string]gen.PurchaseSubGroupResponse {
	if groups == nil {
		return nil
	}
	converted := make(map[string]gen.PurchaseSubGroupResponse, len(groups))
	for key, node := range groups {
		converted[key] = gen.PurchaseSubGroupResponse{
			Name:       node.Name,
			ItemsTotal: node.ItemsTotal.String(),
		}
	}
	return &converted
}
