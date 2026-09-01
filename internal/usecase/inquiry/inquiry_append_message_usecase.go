package inquiry

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/inquiry"
	"go-boilerplate/internal/domain/inquirymessage"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// maxAppendAttempts は、問い合わせの作成が競合したときにトランザクションごとやり直す上限です
// （docs/spec/inquiry/usecase.md の AppendMessage）。
const maxAppendAttempts = 2

// AppendMessage は、利用者の投稿を追加します。
// active な問い合わせが無ければ作成し、作成が競合した場合はトランザクションごとやり直します。
//
// この本体に外部副作用を足してはなりません。やり直しで二重に実行されます
// （ADR-0035 (transaction-retry-idempotent-callers)）。
func (u *usecase) AppendMessage(ctx context.Context, params AppendMessageParams) (MessageView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	var raced error
	for attempt := range maxAppendAttempts {
		view, err := tx.DoWithResult(ctx, u.txm, func(ctx context.Context) (MessageView, error) {
			return u.appendForUser(ctx, params, attempt == 0)
		})
		if !xerrors.Is(err, errInquiryCreationRace) {
			return view, err
		}
		raced = err
	}

	return MessageView{}, xerrors.Wrap(raced, "inquiry creation kept losing the race")
}

// appendForUser は、1 トランザクション分の投稿処理です。
// mayCreate が false のやり直しでは作成を試みず、先に作られた問い合わせを読むだけにします。
func (u *usecase) appendForUser(
	ctx context.Context,
	params AppendMessageParams,
	mayCreate bool,
) (MessageView, error) {
	i, err := u.resolveOrCreateInquiry(ctx, params.UserID, mayCreate)
	if err != nil {
		return MessageView{}, err
	}

	author, err := inquirymessage.NewAuthor(inquirymessage.AuthorKindUser, params.UserID)
	if err != nil {
		return MessageView{}, err
	}

	return u.appendMessage(ctx, i, author, params.Body)
}

// resolveOrCreateInquiry は、利用者の問い合わせを取得し、無ければ作成します。
//
// 作成が一意制約に当たった場合はやり直しを求めます。一意制約違反はトランザクション自体を中断させ、
// 同じトランザクションの中では読み直せないためです（docs/spec/inquiry/usecase.md の AppendMessage）。
func (u *usecase) resolveOrCreateInquiry(
	ctx context.Context,
	userID uuid.UUID,
	mayCreate bool,
) (*inquiry.Inquiry, error) {
	found, err := u.repo.FindActiveByUserID(ctx, userID)
	if err == nil {
		return found, nil
	}
	if !xerrors.Is(err, apperror.ErrNotFound) {
		return nil, err
	}
	if !mayCreate {
		return nil, errInquiryCreationRace
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
			return nil, errInquiryCreationRace
		}
		return nil, cerr
	}
	return created, nil
}
