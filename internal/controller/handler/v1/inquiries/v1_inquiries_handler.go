//go:generate oapi-codegen --include-tags=v1/inquiries --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/inquiries --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package inquiries は、/v1/inquiries エンドポイントに関連するハンドラを提供します。
package inquiries

import (
	"context"

	"github.com/labstack/echo/v5"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/inquiries/gen"
	"go-boilerplate/internal/observability"
	inquiryuc "go-boilerplate/internal/usecase/inquiry"
	"go-boilerplate/internal/usecase/tools/paging"
)

type server struct {
	tracer observability.LayerTracer
	uc     inquiryuc.Usecase
}

// BindHandler は、問い合わせ一覧のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc inquiryuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{tracer: tf.Controller(), uc: uc}, nil))
}

// GetInquiries は、問い合わせを更新日時の新しい順に 1 ページ返します。
func (s *server) GetInquiries(
	ctx context.Context, request gen.GetInquiriesRequestObject,
) (gen.GetInquiriesResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	cursor, err := paging.NewCursor(request.Params.After, request.Params.First)
	if err != nil {
		return nil, err
	}

	view, err := s.uc.ListInquiries(ctx, &authn, inquiryuc.ListInquiriesParams{Cursor: cursor})
	if err != nil {
		return nil, err
	}

	return gen.GetInquiries200JSONResponse(toInquiryListResponse(view)), nil
}

// toInquiryListResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toInquiryListResponse(view *inquiryuc.InquiryListView) gen.InquiryListResponse {
	items := make([]gen.InquirySummary, 0, len(view.Items))
	for _, item := range view.Items {
		items = append(items, gen.InquirySummary{
			Id:        item.ID.ToPrimitive(),
			UserId:    item.UserID.ToPrimitive(),
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return gen.InquiryListResponse{Items: items, NextCursor: view.NextCursor}
}
