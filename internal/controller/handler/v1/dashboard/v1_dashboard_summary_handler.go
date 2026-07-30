//go:generate oapi-codegen --include-tags=v1/dashboard --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/dashboard --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package dashboard は、/v1/dashboard 配下のエンドポイントに関連するハンドラを提供します。
package dashboard

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/dashboard/gen"
	"go-boilerplate/internal/observability"
	dashboarduc "go-boilerplate/internal/usecase/dashboard"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
)

// ErrUnauthenticatedUser は、認証ユーザー情報が取得できない場合のエラーです。
var ErrUnauthenticatedUser = xerrors.Wrap(apperror.ErrUnauthenticated, "requires authenticated user")

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

	authn, ok := ctxhelper.GetAuthn(ctx)
	if !ok {
		return nil, ErrUnauthenticatedUser
	}

	view, err := s.uc.GetDashboardSummary(ctx, &authn, dashboarduc.GetSummaryParams{
		Period: periodParam(request.Params.Period),
		From:   dateParam(request.Params.From),
		To:     dateParam(request.Params.To),
	})
	if err != nil {
		return nil, err
	}

	return gen.GetDashboardSummary200JSONResponse(toDashboardSummaryResponse(view)), nil
}

// periodParam は、任意指定の period クエリパラメータを usecase 入力の文字列へ変換します。未指定は空文字として扱います。
func periodParam(period *gen.GetDashboardSummaryParamsPeriod) string {
	if period == nil {
		return ""
	}
	return string(*period)
}

// dateParam は、任意指定の日付クエリパラメータを usecase 入力へ変換します。未指定は nil のまま渡します。
func dateParam(date *gen.DashboardFromParam) *time.Time {
	if date == nil {
		return nil
	}
	return &date.Time
}

// toDashboardSummaryResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toDashboardSummaryResponse(view dashboarduc.SummaryView) gen.DashboardSummaryResponse {
	statusCounts := make([]gen.DashboardPurchaseStatusCountResponse, len(view.PurchaseStatusCounts))
	for i, c := range view.PurchaseStatusCounts {
		statusCounts[i] = gen.DashboardPurchaseStatusCountResponse{
			Status: gen.PurchaseStatusRef{
				Id:   c.StatusID.ToPrimitive(),
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
