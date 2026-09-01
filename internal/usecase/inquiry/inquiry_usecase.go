//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package inquiry は、問い合わせのユースケースを提供します。利用者の投稿・履歴・購読と、
// 運営の一覧・履歴・回答・購読を扱います。
//
// Realtime Delivery への変換（realtime adapter）はこの package の内側に置きます。機構側は
// 問い合わせの語彙を知りません（ADR-0071 (realtime-delivery-driving-mechanism)）。
package inquiry

import (
	"context"

	"go-boilerplate/internal/domain/inquiry"
	"go-boilerplate/internal/domain/inquirymessage"
	"go-boilerplate/internal/observability"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	"go-boilerplate/internal/usecase/boundary/clock"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/internal/usecase/outbox"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
)

// feedStreamID は、運営の一覧画面が購読する組織 feed の stream です。
// 単一組織を前提とした固定値で、組織が複数になれば一覧と feed ticket に組織の軸が入ります
// （docs/spec/inquiry/usecase.md の placeholder）。環境で変わる値ではないため config には置きません。
const feedStreamID rt.StreamID = "inquiry-feed"

// Usecase は、問い合わせのユースケースです。
type Usecase interface {
	// AppendMessage は、利用者の投稿を追加します。
	// active な問い合わせが無ければ作成し、以降の投稿は同じ問い合わせへ追加します。
	AppendMessage(ctx context.Context, params AppendMessageParams) (MessageView, error)

	// GetHistory は、利用者自身の履歴を位置の昇順で 1 ページ返します。
	// 問い合わせを持たない利用者には空の履歴を返します。
	GetHistory(ctx context.Context, params HistoryParams) (*HistoryView, error)

	// IssueStreamTicket は、利用者の会話 stream を購読するための ticket を発行します。
	IssueStreamTicket(ctx context.Context, params IssueStreamTicketParams) (TicketView, error)

	// ListInquiries は、運営向けに問い合わせを更新日時の新しい順で 1 ページ返します。
	ListInquiries(ctx context.Context, authn *authbd.Authn, params ListInquiriesParams) (*InquiryListView, error)

	// GetInquiryHistory は、運営向けに任意の問い合わせの履歴を 1 ページ返します。
	GetInquiryHistory(ctx context.Context, authn *authbd.Authn, params OperatorHistoryParams) (*HistoryView, error)

	// Reply は、運営の回答を追加します。
	Reply(ctx context.Context, authn *authbd.Authn, params ReplyParams) (MessageView, error)

	// IssueFeedTicket は、運営の feed stream を購読するための ticket を発行します。
	IssueFeedTicket(ctx context.Context, authn *authbd.Authn) (TicketView, error)
}

type usecase struct {
	txm        tx.Manager
	clock      clock.Clock
	repo       inquiry.Repository
	msgRepo    inquirymessage.Repository
	sequences  rt.SequenceAllocator
	emit       outbox.EmitUsecase
	tickets    ucrealtime.TicketIssuer
	authorizer authz.Authorizer
	tracer     observability.LayerTracer
}

// New は、問い合わせユースケースを生成します。
// 依存はすべて異なる型なので、他のユースケースと同じく個別の引数で受けます
// （構造体で束ねると DI コンテナが「解決すべき 1 つの型」として扱い、結線が解けません）。
func New(
	txm tx.Manager,
	clk clock.Clock,
	repo inquiry.Repository,
	msgRepo inquirymessage.Repository,
	sequences rt.SequenceAllocator,
	emit outbox.EmitUsecase,
	tickets ucrealtime.TicketIssuer,
	authorizer authz.Authorizer,
	tf observability.TracerFactory,
) Usecase {
	return &usecase{
		txm:        txm,
		clock:      clk,
		repo:       repo,
		msgRepo:    msgRepo,
		sequences:  sequences,
		emit:       emit,
		tickets:    tickets,
		authorizer: authorizer,
		tracer:     tf.Usecase(),
	}
}

// toMessageView は、メッセージ集約を出力 DTO へ写します。
func toMessageView(m *inquirymessage.Message) MessageView {
	return MessageView{
		ID:         m.ID(),
		InquiryID:  m.InquiryID(),
		AuthorKind: m.Author().Kind().String(),
		Body:       m.Body(),
		Sequence:   m.Sequence(),
		CreatedAt:  m.CreatedAt(),
	}
}
