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
//
// 作成が競合した場合は、先に作られた問い合わせを読み直して返します。作成は行が既にあれば
// 何もしないため、競合してもトランザクションは中断されません
// （docs/spec/usecase/inquiry.md の AppendMessage）。
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
	created, err := inquiry.New(inquiryID, inquiry.Attributes{UserID: userID})
	if err != nil {
		return nil, err
	}
	if cerr := u.repo.Create(ctx, created); cerr != nil {
		if xerrors.Is(cerr, apperror.ErrConflict) {
			return u.findRaceWinner(ctx, userID)
		}
		return nil, cerr
	}
	return created, nil
}

// findRaceWinner は、作成に先を越されたあとで相手が作った問い合わせを読み直します。
//
// 読み直しは同じトランザクションの中で行うため、そのトランザクションが自分より後の
// コミットを見ない分離レベルで開かれていると、勝者が見えないことがあります。その場合は
// NotFound ではなく競合として返し、利用者が再送すれば解けるようにします。
func (u *usecase) findRaceWinner(ctx context.Context, userID uuid.UUID) (*inquiry.Inquiry, error) {
	found, err := u.repo.FindActiveByUserID(ctx, userID)
	if err == nil {
		return found, nil
	}
	if xerrors.Is(err, apperror.ErrNotFound) {
		return nil, errInquiryCreationRace
	}
	return nil, err
}
