package inquiry

import (
	"context"

	"go-boilerplate/internal/domain/inquiry"
	"go-boilerplate/internal/usecase/boundary/outbox"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/internal/usecase/inquiry/event"
	ucoutbox "go-boilerplate/internal/usecase/outbox"
)

// aggregateType は、outbox 行に載せる集約種別（観測・調査用）です。
const aggregateType = "inquiry"

// conversationStreamID は、1 つの問い合わせに対応する会話 stream の識別子を返します。
func conversationStreamID(inquiryID string) rt.StreamID { return rt.StreamID(inquiryID) }

// emitDelivery は、1 通の追加を会話 stream と feed stream の 2 つの event として出します。
//
// 2 つを 1 つの event にまとめてはなりません（docs/spec/inquiry/usecase.md の 2 つの destination）。
func (u *usecase) emitDelivery(
	ctx context.Context,
	i *inquiry.Inquiry,
	m *inquiry.Message,
	feedSequence int64,
) error {
	messagePayload, err := event.BuildMessageCreated(i, m)
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
