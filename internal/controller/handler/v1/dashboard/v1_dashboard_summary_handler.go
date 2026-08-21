//go:generate oapi-codegen --include-tags=v1/dashboard --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/dashboard --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package dashboard は、/v1/dashboard 配下のエンドポイントに関連するハンドラを提供します。
package dashboard

import (
	"context"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/dashboard/gen"
	"go-boilerplate/internal/observability"
	dashboarduc "go-boilerplate/internal/usecase/dashboard"
	"go-boilerplate/internal/usecase/tools/timewindow"

	"github.com/labstack/echo/v5"
)

type server struct {
	tracer observability.LayerTracer
	uc     dashboarduc.Usecase
}

// BindHandler は、admin ダッシュボード横断集計のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc dashboarduc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetDashboardSummary は、購入・商品を横断した admin ダッシュボード向けの集計を取得します。
func (s *server) GetDashboardSummary(
	ctx context.Context, request gen.GetDashboardSummaryRequestObject,
) (gen.GetDashboardSummaryResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	window, err := timewindow.New(timewindow.Bounds{
		After:  request.Params.OrderedAfter,
		Before: request.Params.OrderedBefore,
	})
	if err != nil {
		return nil, err
	}

	view, err := s.uc.GetDashboardSummary(ctx, &authn, window)
	if err != nil {
		return nil, err
	}

	return gen.GetDashboardSummary200JSONResponse(toDashboardSummaryResponse(view)), nil
}

// toDashboardSummaryResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toDashboardSummaryResponse(view dashboarduc.SummaryView) gen.DashboardSummaryResponse {
	statusCounts := make([]gen.DashboardPurchaseStatusCountResponse, len(view.PurchaseStatusCounts))
	for i, c := range view.PurchaseStatusCounts {
		statusCounts[i] = gen.DashboardPurchaseStatusCountResponse{
			Status: gen.PurchaseStatusRef{
				Id:   c.StatusID.ToPrimitive(),
				Code: int64(c.StatusCode),
				Name: c.StatusName,
			},
			Count: c.Count,
		}
	}

	return gen.DashboardSummaryResponse{
		SalesAmount:           view.SalesAmount,
		SalesCount:            view.SalesCount,
		PurchaseStatusCounts:  statusCounts,
		TotalProductCount:     view.TotalProductCount,
		PublishedProductCount: view.PublishedProductCount,
	}
}
