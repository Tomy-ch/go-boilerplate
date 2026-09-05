package inquiry

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/inquiry"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// AppendMessage は、問い合わせの確保から採番・追加・更新までを 1 つの tx に収めます。
// 問い合わせが無ければ CreateIfAbsent が単一文で確保するので、並行しても衝突しません。
//
// この本体に外部副作用を足してはなりません（tx.Manager.Do の冪等性契約、
// ADR-0035 (transaction-retry-idempotent-callers)）。
func (u *usecase) AppendMessage(ctx context.Context, params AppendMessageParams) (MessageView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	return tx.DoWithResult(ctx, u.txm, func(ctx context.Context) (MessageView, error) {
		return u.appendForUser(ctx, params)
	})
}

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
