package inquiry

import (
	"context"
	"fmt"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/inquiry"
	"go-boilerplate/internal/domain/inquirymessage"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// inquiryCursorKeyCount は、一覧の keyset 境界を成すキーの個数（更新日時, ID）です。
const inquiryCursorKeyCount = 2

// ListInquiries は、運営向けに問い合わせを更新日時の新しい順で 1 ページ返します。
func (u *usecase) ListInquiries(
	ctx context.Context,
	authn *authbd.Authn,
	params ListInquiriesParams,
) (*InquiryListView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if err := u.authorize(ctx, authn, authz.ActionInquiryList); err != nil {
		return nil, err
	}

	boundary, err := decodeInquiryCursor(params.Cursor)
	if err != nil {
		return nil, err
	}

	// 次ページの有無は 1 件多く読んで判定します。
	limit := params.Cursor.Limit()
	inquiries, err := u.repo.ListForOperator(ctx, inquiry.ListParams{Cursor: boundary, Limit: limit + 1})
	if err != nil {
		return nil, err
	}

	var next *string
	if len(inquiries) > limit {
		inquiries = inquiries[:limit]
		encoded := encodeInquiryCursor(inquiries[len(inquiries)-1])
		next = &encoded
	}

	items := make([]InquirySummaryView, 0, len(inquiries))
	for _, i := range inquiries {
		items = append(items, InquirySummaryView{
			ID:        i.ID(),
			UserID:    i.UserID(),
			CreatedAt: i.CreatedAt(),
			UpdatedAt: i.UpdatedAt(),
		})
	}

	return &InquiryListView{Items: items, NextCursor: next}, nil
}

// GetInquiryHistory は、運営向けに任意の問い合わせの履歴を 1 ページ返します。
// 手順は利用者向けの履歴と同じで、母集団の決め方だけが異なります。
func (u *usecase) GetInquiryHistory(
	ctx context.Context,
	authn *authbd.Authn,
	params OperatorHistoryParams,
) (*HistoryView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if err := u.authorize(ctx, authn, authz.ActionInquiryReadAll); err != nil {
		return nil, err
	}

	return tx.DoWithResult(ctx, u.txm, func(ctx context.Context) (*HistoryView, error) {
		i, err := u.repo.FindByID(ctx, params.InquiryID)
		if err != nil {
			return nil, err
		}
		return u.historyOf(ctx, i, params.AfterSequence, params.First)
	})
}

// Reply は、運営の回答を追加します。
// 追加の手順は利用者の投稿と同じで、送り手の種別と問い合わせの決め方だけが異なります。
func (u *usecase) Reply(
	ctx context.Context,
	authn *authbd.Authn,
	params ReplyParams,
) (MessageView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if err := u.authorize(ctx, authn, authz.ActionInquiryReply); err != nil {
		return MessageView{}, err
	}

	author, err := inquirymessage.NewAuthor(inquirymessage.AuthorKindOperator, params.OperatorID)
	if err != nil {
		return MessageView{}, err
	}

	return tx.DoWithResult(ctx, u.txm, func(ctx context.Context) (MessageView, error) {
		i, ferr := u.repo.FindByID(ctx, params.InquiryID)
		if ferr != nil {
			return MessageView{}, ferr
		}
		return u.appendMessage(ctx, i, author, params.Body)
	})
}

// authorize は、認証主体が運営操作を行ってよいか判定します。
// 所有者を持たないリソースとして問い合わせるため、所有者フォールバックは成立せず admin だけが通ります。
func (u *usecase) authorize(ctx context.Context, authn *authbd.Authn, action authz.Action) error {
	if authn == nil {
		return apperror.ErrUnauthenticated
	}
	return u.authorizer.Authorize(ctx, authn, action, authz.NewResource("inquiry", nil))
}

// decodeInquiryCursor は、不透明キー列を keyset 境界へ解釈します。
// 先頭ページ（カーソル無し）では境界なし（nil）を正常値として返します。
func decodeInquiryCursor(cursor *paging.Cursor) (*inquiry.Cursor, error) {
	if !cursor.HasCursor() {
		return nil, nil //nolint:nilnil // 先頭ページは境界なし（nil）を正常値として返す
	}

	keys := cursor.Keys()
	if len(keys) != inquiryCursorKeyCount {
		return nil, xerrors.Wrap(
			apperror.ErrInvalidArgument,
			fmt.Sprintf("invalid cursor: expected %d keys", inquiryCursorKeyCount),
		)
	}

	updatedAt, err := time.Parse(time.RFC3339Nano, keys[0])
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "invalid cursor: updated_at is not RFC3339Nano")
	}
	id, err := uuid.Parse(keys[1])
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "invalid cursor: id is not a valid UUID")
	}

	return &inquiry.Cursor{UpdatedAt: updatedAt, ID: id}, nil
}

// encodeInquiryCursor は、ページ末尾のソートキー（更新日時, ID）から次ページ用の不透明カーソルを作ります。
func encodeInquiryCursor(last *inquiry.Inquiry) string {
	return paging.EncodeCursor(last.UpdatedAt().Format(time.RFC3339Nano), last.ID().String())
}
