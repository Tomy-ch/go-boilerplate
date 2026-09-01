package inquiry

import (
	"context"

	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
)

const (
	// conversationScope は、会話 stream に対して認可した権限の範囲です。機構は解釈しません。
	conversationScope = "inquiry:read"
	// feedScope は、運営 feed に対して認可した権限の範囲です。
	feedScope = "inquiry-feed:read"
)

// IssueStreamTicket は、利用者の会話 stream を購読するための ticket を発行します。
// 購読する問い合わせが無い場合は NotFound を返します。
func (u *usecase) IssueStreamTicket(
	ctx context.Context,
	params IssueStreamTicketParams,
) (TicketView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	i, err := u.repo.FindActiveByUserID(ctx, params.UserID)
	if err != nil {
		return TicketView{}, err
	}

	return u.issueTicket(ctx, params.Subject, conversationStreamID(i.ID().String()), conversationScope)
}

// IssueFeedTicket は、運営 feed を購読するための ticket を発行します。
func (u *usecase) IssueFeedTicket(ctx context.Context, authn *authbd.Authn) (TicketView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if err := u.authorize(ctx, authn, authz.ActionInquiryFeedSubscribe); err != nil {
		return TicketView{}, err
	}

	return u.issueTicket(ctx, authn.Subject(), feedStreamID, feedScope)
}

// issueTicket は、認可済みの subject × stream に対して ticket を発行します。
//
// 開始位置には stream の現在位置を渡します。cursor を持たずに接続した client へ、履歴 API が担う
// 過去分を購読側から再生し直さないためです。これは認可の下限ではないので、client は replay の
// 保持範囲であればより前の位置から再開できます（docs/design/realtime-delivery.md §2.3）。
func (u *usecase) issueTicket(
	ctx context.Context,
	subject string,
	destination rt.StreamID,
	scope string,
) (TicketView, error) {
	cursor, ok, err := u.sequences.Current(ctx, destination)
	if err != nil {
		return TicketView{}, err
	}
	if !ok {
		cursor = 0
	}

	issued, err := u.tickets.Issue(ctx, ucrealtime.IssueTicketInput{
		Subject:       subject,
		Destination:   destination,
		Scope:         scope,
		InitialCursor: cursor,
	})
	if err != nil {
		return TicketView{}, err
	}

	return TicketView{
		Ticket:    issued.Value,
		StreamID:  destination.String(),
		ExpiresAt: issued.ExpiresAt,
	}, nil
}
