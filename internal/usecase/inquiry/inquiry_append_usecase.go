package inquiry

import (
	"context"

	"go-boilerplate/internal/domain/inquiry"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// appendMessage は、投稿と回答が共有する 1 トランザクション分の追加処理です。
//
// 呼び出しの順序を入れ替えてはなりません。同一 stream への並行追加は先頭の採番で直列化されます
// （docs/spec/inquiry/usecase.md の AppendMessage）。
func (u *usecase) appendMessage(
	ctx context.Context,
	i *inquiry.Inquiry,
	author inquiry.Author,
	body string,
) (MessageView, error) {
	sequence, err := u.sequences.Allocate(ctx, conversationStreamID(i.ID().String()))
	if err != nil {
		return MessageView{}, err
	}

	messageID, err := uuid.New()
	if err != nil {
		return MessageView{}, xerrors.Wrap(err, "failed to generate inquiry message id")
	}

	m, err := i.AppendMessage(messageID, inquiry.MessageAttributes{
		Author:   author,
		Body:     body,
		Sequence: int64(sequence),
	}, u.clock.Now())
	if err != nil {
		return MessageView{}, err
	}
	if cerr := u.repo.CreateMessage(ctx, i.ID(), m); cerr != nil {
		return MessageView{}, cerr
	}
	if uerr := u.repo.Update(ctx, i); uerr != nil {
		return MessageView{}, uerr
	}

	feedSequence, err := u.sequences.Allocate(ctx, feedStreamID)
	if err != nil {
		return MessageView{}, err
	}

	// 作成日時は DB の既定値が刻むため、書き込んだ行を読み直してから event と応答を組み立てます。
	// 読み直しは書き込みの検証も兼ねます（internal/usecase/README.md の
	// Verifying infrastructure against the domain）。
	stored, err := u.readBackMessage(ctx, i.ID(), int64(sequence))
	if err != nil {
		return MessageView{}, err
	}

	if eerr := u.emitDelivery(ctx, i, stored, int64(feedSequence)); eerr != nil {
		return MessageView{}, eerr
	}

	return toMessageView(i.ID(), stored), nil
}

// readBackMessage は、いま追加した 1 通を読み直します。
func (u *usecase) readBackMessage(
	ctx context.Context,
	inquiryID uuid.UUID,
	sequence int64,
) (*inquiry.Message, error) {
	previous := sequence - 1
	messages, err := u.repo.ListMessages(ctx, inquiryID, inquiry.HistoryParams{
		AfterSequence: &previous,
		UpToSequence:  sequence,
		Limit:         1,
	})
	if err != nil {
		return nil, err
	}
	if len(messages) != 1 {
		return nil, xerrors.Wrap(errMessageNotStored, "the appended message was not readable back")
	}
	return messages[0], nil
}
