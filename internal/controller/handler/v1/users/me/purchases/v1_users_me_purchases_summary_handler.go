//go:generate oapi-codegen --include-tags=v1/users/me/purchases --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/users/me/purchases --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package usersmepurchases は、/v1/users/me/purchases 配下のエンドポイントに関連するハンドラを提供します。
package usersmepurchases

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/users/me/purchases/gen"
	"go-boilerplate/internal/observability"
	summaryuc "go-boilerplate/internal/usecase/purchase/summary"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
)

// ErrUnauthenticatedUser は、認証ユーザー情報が取得できない場合のエラーです。
var ErrUnauthenticatedUser = xerrors.Wrap(apperror.ErrUnauthenticated, "requires authenticated user")

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
	ctx context.Context, _ gen.GetUsersMePurchasesSummaryRequestObject,
) (gen.GetUsersMePurchasesSummaryResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, ok := ctxhelper.GetAuthn(ctx)
	if !ok {
		return nil, ErrUnauthenticatedUser
	}

	view, err := s.uc.GetPurchaseSummary(ctx, &authn)
	if err != nil {
		return nil, err
	}

	return gen.GetUsersMePurchasesSummary200JSONResponse(toPurchaseAggregateResponse(view)), nil
}

// toPurchaseAggregateResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toPurchaseAggregateResponse(view summaryuc.SummaryView) gen.PurchaseAggregateResponse {
	breakdown := make([]gen.PurchaseStatusBreakdownResponse, len(view.StatusBreakdown))
	for i, b := range view.StatusBreakdown {
		breakdown[i] = gen.PurchaseStatusBreakdownResponse{
			Status: gen.PurchaseStatusRef{
				Id:   b.StatusID.ToPrimitive(),
				Name: b.StatusName,
			},
			Count:       b.Count,
			TotalAmount: b.TotalAmount,
		}
	}

	return gen.PurchaseAggregateResponse{
		TotalCount:      view.TotalCount,
		TotalAmount:     view.TotalAmount,
		StatusBreakdown: breakdown,
	}
}
