package inquiry

import (
	"context"

	"go-boilerplate/internal/domain/inquiry"
	"go-boilerplate/internal/domain/inquirymessage"
	"go-boilerplate/internal/usecase/boundary/outbox"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/internal/usecase/inquiry/event"
	ucoutbox "go-boilerplate/internal/usecase/outbox"
)

// aggregateType は、outbox 行に載せる集約種別（観測・調査用）です。
const aggregateType = "inquiry"

// conversationStreamID は、1 つの問い合わせに対応する会話 stream の識別子を返します。
// 機構は問い合わせを知らないため、feature 側で問い合わせ ID を stream の語彙へ写します。
func conversationStreamID(inquiryID string) rt.StreamID { return rt.StreamID(inquiryID) }

// emitDelivery は、1 通の追加を 2 つの stream へ出します。
//
// 1 event = 1 stream のため、会話画面と一覧画面には別々の event を出します。順序の単位（ordering_key）は
// それぞれの stream で、位置は各 stream の採番結果です。どちらも同じ業務 tx の中で記録されます。
func (u *usecase) emitDelivery(
	ctx context.Context,
	i *inquiry.Inquiry,
	m *inquirymessage.Message,
	feedSequence int64,
) error {
	messagePayload, err := event.BuildMessageCreated(m)
	if err != nil {
		return err
	}
	if _, eerr := u.emit.Emit(ctx, ucoutbox.EmitInput{
		AggregateType:    aggregateType,
		AggregateID:      i.ID().String(),
		EventType:        event.TypeMessageCreated,
		Payload:          messagePayload,
		Channel:          outbox.ChannelRealtime,
		OrderingKey:      conversationStreamID(i.ID().String()).String(),
		OrderingSequence: m.Sequence(),
	}); eerr != nil {
		return eerr
	}

	threadPayload, err := event.BuildThreadUpdated(i, m.Sequence())
	if err != nil {
		return err
	}
	if _, eerr := u.emit.Emit(ctx, ucoutbox.EmitInput{
		AggregateType:    aggregateType,
		AggregateID:      i.ID().String(),
		EventType:        event.TypeThreadUpdated,
		Payload:          threadPayload,
		Channel:          outbox.ChannelRealtime,
		OrderingKey:      feedStreamID.String(),
		OrderingSequence: feedSequence,
	}); eerr != nil {
		return eerr
	}

	return nil
}
