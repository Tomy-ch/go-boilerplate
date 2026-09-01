package inquiry

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/inquiry"
	"go-boilerplate/internal/domain/inquirymessage"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/xerrors"
)

const (
	// historyDefaultLimit は、件数の指定が無いときに返す履歴の件数です。
	historyDefaultLimit = 50
	// historyMaxLimit は、履歴 1 ページの上限件数です。OpenAPI の first と同じ値を保ちます。
	historyMaxLimit = 200
)

// historyLimitPolicy は、履歴の取得件数を正規化する規約です。
var historyLimitPolicy = paging.LimitPolicy{Default: historyDefaultLimit, Max: historyMaxLimit}

// GetHistory は、利用者自身の履歴を 1 ページ返します。
// 問い合わせを持たない利用者には空の履歴を返します（購読する会話がまだ無いだけで、誤りではありません）。
func (u *usecase) GetHistory(ctx context.Context, params HistoryParams) (*HistoryView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	return tx.DoWithResult(ctx, u.txm, func(ctx context.Context) (*HistoryView, error) {
		i, err := u.repo.FindActiveByUserID(ctx, params.UserID)
		if err != nil {
			if xerrors.Is(err, apperror.ErrNotFound) {
				return &HistoryView{Messages: []MessageView{}}, nil
			}
			return nil, err
		}
		return u.historyOf(ctx, i, params.AfterSequence, params.First)
	})
}

// historyOf は、現在位置を先に読み、その位置までのメッセージを 1 ページ読み出します。
//
// この順序は入れ替えてはなりません。現在位置を先に読み、それを上限にすることで
// 「現在位置とメッセージを同じ snapshot で読んだ」のと等価になります
// （docs/spec/inquiry/usecase.md の streamCursor と snapshot）。
func (u *usecase) historyOf(
	ctx context.Context,
	i *inquiry.Inquiry,
	afterSequence *int64,
	first *int,
) (*HistoryView, error) {
	limit := paging.NewLimit(first, historyLimitPolicy).Value()

	cursor, ok, err := u.sequences.Current(ctx, conversationStreamID(i.ID().String()))
	if err != nil {
		return nil, err
	}
	if !ok {
		return &HistoryView{InquiryID: i.ID(), Messages: []MessageView{}}, nil
	}

	messages, err := u.msgRepo.ListByInquiry(ctx, i.ID(), inquirymessage.HistoryParams{
		AfterSequence: afterSequence,
		UpToSequence:  int64(cursor),
		Limit:         limit + 1,
	})
	if err != nil {
		return nil, err
	}

	views, next := pageOf(messages, limit)
	return &HistoryView{
		InquiryID:         i.ID(),
		Messages:          views,
		NextAfterSequence: next,
		StreamCursor:      int64(cursor),
	}, nil
}

// pageOf は、1 件多く読んだ結果を 1 ページ分の DTO と次ページの開始位置へ分けます。
func pageOf(messages []*inquirymessage.Message, limit int) ([]MessageView, *int64) {
	var next *int64
	if len(messages) > limit {
		messages = messages[:limit]
		last := messages[len(messages)-1].Sequence()
		next = &last
	}

	views := make([]MessageView, 0, len(messages))
	for _, m := range messages {
		views = append(views, toMessageView(m))
	}
	return views, next
}
