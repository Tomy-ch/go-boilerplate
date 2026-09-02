//go:generate oapi-codegen --include-tags=v1/inquiries/feed/stream-ticket --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/inquiries/feed/stream-ticket --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package streamticket は、/v1/inquiries/feed/stream-ticket エンドポイントに関連するハンドラを提供します。
package streamticket

import (
	"context"

	"github.com/labstack/echo/v5"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/inquiries/feed/streamticket/gen"
	"go-boilerplate/internal/observability"
	inquiryuc "go-boilerplate/internal/usecase/inquiry"
)

type server struct {
	tracer observability.LayerTracer
	uc     inquiryuc.Usecase
}

// BindHandler は、問い合わせ更新フィードを購読する ticket 発行のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc inquiryuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{tracer: tf.Controller(), uc: uc}, nil))
}

// PostInquiriesFeedStreamTicket は、問い合わせ更新フィードを購読する ticket を発行します。
func (s *server) PostInquiriesFeedStreamTicket(
	ctx context.Context, _ gen.PostInquiriesFeedStreamTicketRequestObject,
) (gen.PostInquiriesFeedStreamTicketResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	view, err := s.uc.IssueFeedTicket(ctx, &authn)
	if err != nil {
		return nil, err
	}

	return gen.PostInquiriesFeedStreamTicket201JSONResponse(gen.InquiryStreamTicketResponse{
		Ticket:    view.Ticket,
		StreamId:  view.StreamID,
		ExpiresAt: view.ExpiresAt,
	}), nil
}
