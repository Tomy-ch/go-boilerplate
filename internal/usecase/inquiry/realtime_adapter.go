package inquiry

import (
	"context"
	"encoding/json"

	"go-boilerplate/internal/domain/inquiry"
	"go-boilerplate/internal/usecase/boundary/outbox"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/internal/usecase/inquiry/event"
	ucoutbox "go-boilerplate/internal/usecase/outbox"
	"go-boilerplate/pkg/xerrors"
)

// aggregateType は、outbox 行に載せる集約種別（観測・調査用）です。
const aggregateType = "inquiry"

// conversationStreamID は、1 つの問い合わせに対応する会話 stream の識別子を返します。
func conversationStreamID(inquiryID string) rt.StreamID { return rt.StreamID(inquiryID) }

// emitDelivery は、1 通の追加を会話 stream と feed stream の 2 つの event として出します。
//
// 2 つを 1 つの event にまとめてはなりません（docs/spec/usecase/inquiry.md の 2 つの destination）。
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

	conversation := conversationStreamID(i.ID().String())

	messageEnvelope, err := envelope(rt.DeliveryEvent{
		StreamID:      conversation,
		Sequence:      rt.Sequence(m.Sequence()),
		Type:          event.TypeMessageCreated,
		OccurredAt:    m.CreatedAt(),
		SchemaVersion: event.SchemaVersionMessageCreated,
		Payload:       messagePayload,
	})
	if err != nil {
		return err
	}

	if _, eerr := u.emit.Emit(ctx, ucoutbox.EmitInput{
		AggregateType:    aggregateType,
		AggregateID:      i.ID().String(),
		EventType:        event.TypeMessageCreated,
		Payload:          messageEnvelope,
		Channel:          outbox.ChannelRealtime,
		OrderingKey:      conversation.String(),
		OrderingSequence: m.Sequence(),
	}); eerr != nil {
		return eerr
	}

	threadPayload, err := event.BuildThreadUpdated(i, m.Sequence())
	if err != nil {
		return err
	}

	// feed 側の位置は会話 stream と別に採番されます。payload に載る sequence は会話側の位置で、
	// 封筒の Sequence は feed の位置です。同じ値に見えても別の stream の位置なので混ぜません。
	threadEnvelope, err := envelope(rt.DeliveryEvent{
		StreamID:      feedStreamID,
		Sequence:      rt.Sequence(feedSequence),
		Type:          event.TypeThreadUpdated,
		OccurredAt:    i.UpdatedAt(),
		SchemaVersion: event.SchemaVersionThreadUpdated,
		Payload:       threadPayload,
	})
	if err != nil {
		return err
	}

	if _, eerr := u.emit.Emit(ctx, ucoutbox.EmitInput{
		AggregateType:    aggregateType,
		AggregateID:      i.ID().String(),
		EventType:        event.TypeThreadUpdated,
		Payload:          threadEnvelope,
		Channel:          outbox.ChannelRealtime,
		OrderingKey:      feedStreamID.String(),
		OrderingSequence: feedSequence,
	}); eerr != nil {
		return eerr
	}

	return nil
}

// envelope は、feature の event を配送封筒へ翻訳して直列化します。outbox の Payload に載せるのはこの
// 封筒であり、feature の payload そのものではありません（翻訳をこの package が担う理由は
// docs/spec/usecase/inquiry.md Notes「realtime adapter の置き場所」）。
//
// EventID は空のままにします。値は outbox が採番する message_id と同じでなければならず、
// それが決まるのは Emit の後だからです。publish 側が message_id で埋めてから検証します。
func envelope(e rt.DeliveryEvent) ([]byte, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to encode realtime delivery envelope")
	}

	return b, nil
}
