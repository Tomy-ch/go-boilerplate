//go:generate oapi-codegen --include-tags=v1/inquiries/me/stream-ticket --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/inquiries/me/stream-ticket --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package streamticket は、/v1/inquiries/me/stream-ticket エンドポイントに関連するハンドラを提供します。
package streamticket

import (
	"context"

	"github.com/labstack/echo/v5"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/inquiries/me/streamticket/gen"
	"go-boilerplate/internal/observability"
	inquiryuc "go-boilerplate/internal/usecase/inquiry"
)

type server struct {
	tracer observability.LayerTracer
	uc     inquiryuc.Usecase
}

// BindHandler は、自分の問い合わせを購読する ticket 発行のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc inquiryuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{tracer: tf.Controller(), uc: uc}, nil))
}

// PostInquiriesMeStreamTicket は、呼び出し主体自身の問い合わせを購読する ticket を発行します。
func (s *server) PostInquiriesMeStreamTicket(
	ctx context.Context, _ gen.PostInquiriesMeStreamTicketRequestObject,
) (gen.PostInquiriesMeStreamTicketResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}
	userID, err := authn.UserID()
	if err != nil {
		return nil, err
	}

	view, err := s.uc.IssueStreamTicket(ctx, inquiryuc.IssueStreamTicketParams{
		UserID:  userID,
		Subject: authn.Subject(),
	})
	if err != nil {
		return nil, err
	}

	return gen.PostInquiriesMeStreamTicket201JSONResponse(gen.InquiryStreamTicketResponse{
		Ticket:    view.Ticket,
		StreamId:  view.StreamID,
		ExpiresAt: view.ExpiresAt,
	}), nil
}
