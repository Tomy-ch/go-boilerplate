package inquiry

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/inquiry"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// AppendMessage は、利用者の投稿を追加します。
// active な問い合わせが無ければ作成します。
//
// この本体に外部副作用を足してはなりません。tx がやり直されると二重に実行されます
// （ADR-0035 (transaction-retry-idempotent-callers)）。
func (u *usecase) AppendMessage(ctx context.Context, params AppendMessageParams) (MessageView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	return tx.DoWithResult(ctx, u.txm, func(ctx context.Context) (MessageView, error) {
		return u.appendForUser(ctx, params)
	})
}

// appendForUser は、1 トランザクション分の投稿処理です。
func (u *usecase) appendForUser(ctx context.Context, params AppendMessageParams) (MessageView, error) {
	i, err := u.resolveOrCreateInquiry(ctx, params.UserID)
	if err != nil {
		return MessageView{}, err
	}

	author, err := inquiry.NewAuthor(inquiry.AuthorKindUser, params.UserID)
	if err != nil {
		return MessageView{}, err
	}

	return u.appendMessage(ctx, i, author, params.Body)
}

// resolveOrCreateInquiry は、利用者の問い合わせを取得し、無ければ作成します。
func (u *usecase) resolveOrCreateInquiry(
	ctx context.Context,
	userID uuid.UUID,
) (*inquiry.Inquiry, error) {
	found, err := u.repo.FindActiveByUserID(ctx, userID)
	if err == nil {
		return found, nil
	}
	if !xerrors.Is(err, apperror.ErrNotFound) {
		return nil, err
	}

	inquiryID, err := uuid.New()
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to generate inquiry id")
	}
	candidate, err := inquiry.New(inquiryID, inquiry.Attributes{UserID: userID})
	if err != nil {
		return nil, err
	}
	return u.repo.CreateIfAbsent(ctx, candidate)
}
