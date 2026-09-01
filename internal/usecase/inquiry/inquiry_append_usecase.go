package inquiry

import (
	"context"

	"go-boilerplate/internal/domain/inquiry"
	"go-boilerplate/internal/domain/inquirymessage"
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
	author inquirymessage.Author,
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

	m, err := inquirymessage.New(messageID, inquirymessage.Attributes{
		InquiryID: i.ID(),
		Author:    author,
		Body:      body,
		Sequence:  int64(sequence),
	})
	if err != nil {
		return MessageView{}, err
	}
	if cerr := u.msgRepo.Create(ctx, m); cerr != nil {
		return MessageView{}, cerr
	}

	now := u.clock.Now()
	if terr := i.Touch(now); terr != nil {
		return MessageView{}, terr
	}
	if terr := u.repo.Touch(ctx, i.ID(), now); terr != nil {
		return MessageView{}, terr
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

	return toMessageView(stored), nil
}

// readBackMessage は、いま追加した 1 通を読み直します。
func (u *usecase) readBackMessage(
	ctx context.Context,
	inquiryID uuid.UUID,
	sequence int64,
) (*inquirymessage.Message, error) {
	previous := sequence - 1
	messages, err := u.msgRepo.ListByInquiry(ctx, inquiryID, inquirymessage.HistoryParams{
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
