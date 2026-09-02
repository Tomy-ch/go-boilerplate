//go:generate oapi-codegen --include-tags=v1/inquiries/detail/messages --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/inquiries/detail/messages --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package messages は、/v1/inquiries/{inquiryId}/messages エンドポイントに関連するハンドラを提供します。
package messages

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v5"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/inquiries/detail/messages/gen"
	idempotencymw "go-boilerplate/internal/controller/httpstack/idempotency"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/idempotency"
	inquiryuc "go-boilerplate/internal/usecase/inquiry"
	"go-boilerplate/pkg/uuid"
)

type server struct {
	tracer observability.LayerTracer
	uc     inquiryuc.Usecase
	idem   idempotency.Deps
}

// BindHandler は、任意の問い合わせの履歴と回答のハンドラを Echo に登録します。冪等ミドルウェアを併用します。
func BindHandler(
	e *echo.Echo,
	tf observability.TracerFactory,
	uc inquiryuc.Usecase,
	idem idempotency.Deps,
) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
		idem:   idem,
	}, []gen.StrictMiddlewareFunc{idempotencymw.StrictMiddleware[gen.StrictHandlerFunc]()}))
}

// GetInquiriesDetailMessages は、指定した問い合わせの履歴を 1 ページ返します。
func (s *server) GetInquiriesDetailMessages(
	ctx context.Context, request gen.GetInquiriesDetailMessagesRequestObject,
) (gen.GetInquiriesDetailMessagesResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	view, err := s.uc.GetInquiryHistory(ctx, &authn, inquiryuc.OperatorHistoryParams{
		InquiryID:     uuid.FromPrimitive(request.InquiryId),
		AfterSequence: request.Params.AfterSequence,
		First:         request.Params.First,
	})
	if err != nil {
		return nil, err
	}

	return gen.GetInquiriesDetailMessages200JSONResponse(toHistoryResponse(view)), nil
}

// PostInquiriesDetailMessages は、指定した問い合わせへ回答を 1 通追加します。
func (s *server) PostInquiriesDetailMessages(
	ctx context.Context, request gen.PostInquiriesDetailMessagesRequestObject,
) (gen.PostInquiriesDetailMessagesResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}
	operatorID, err := authn.UserID()
	if err != nil {
		return nil, err
	}

	result, _, err := idempotency.Run(ctx, s.idem, http.StatusCreated,
		func(ctx context.Context) (inquiryuc.MessageView, error) {
			return s.uc.Reply(ctx, &authn, inquiryuc.ReplyParams{
				InquiryID:  uuid.FromPrimitive(request.InquiryId),
				OperatorID: operatorID,
				Body:       request.Body.Body,
			})
		})
	if err != nil {
		return nil, err
	}

	return gen.PostInquiriesDetailMessages201JSONResponse(toMessageResponse(result)), nil
}

// toHistoryResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toHistoryResponse(view *inquiryuc.HistoryView) gen.InquiryHistoryResponse {
	messages := make([]gen.InquiryMessage, 0, len(view.Messages))
	for _, m := range view.Messages {
		messages = append(messages, toInquiryMessage(m))
	}

	response := gen.InquiryHistoryResponse{
		Messages:          messages,
		NextAfterSequence: view.NextAfterSequence,
		StreamCursor:      view.StreamCursor,
	}
	if !view.InquiryID.IsNil() {
		id := view.InquiryID.ToPrimitive()
		response.InquiryId = &id
	}
	return response
}

// toMessageResponse は、追加したメッセージを HTTP レスポンスへ変換します。
func toMessageResponse(view inquiryuc.MessageView) gen.InquiryMessageResponse {
	return gen.InquiryMessageResponse{Message: toInquiryMessage(view)}
}

// toInquiryMessage は、メッセージ 1 通を応答の型へ変換します。
func toInquiryMessage(view inquiryuc.MessageView) gen.InquiryMessage {
	return gen.InquiryMessage{
		Id:         view.ID.ToPrimitive(),
		InquiryId:  view.InquiryID.ToPrimitive(),
		AuthorKind: gen.InquiryMessageAuthorKind(view.AuthorKind),
		Body:       view.Body,
		Sequence:   view.Sequence,
		CreatedAt:  view.CreatedAt,
	}
}
