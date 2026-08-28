// Package realtime は、Realtime Delivery（docs/design/realtime-delivery.md）の seam を定義します。
// feature 中立な封筒 DeliveryEvent と、それを保存・配送する側が実装する境界
// （EventLogStore / StreamTicketStore / InstanceLeaseStore / SecretGenerator、および #1410 の
// SequenceAllocator）を置きます。DynamoDB / AWS の語彙（table / partition / TTL 等）はこの
// package に現れません。feature の語彙（会話・メッセージ等）も現れません。
package realtime

import (
	"encoding/json"
	"strconv"
	"time"

	"go-boilerplate/pkg/xerrors"
)

// MaxSerializedBytes は、DeliveryEvent を JSON に直列化したときの上限（64 KiB）です。
// payload 単体ではなく封筒全体で判定します（SSE の 1 event として送る大きさがこれだからです）。
const MaxSerializedBytes = 64 * 1024

// StreamID は、event が属する stream（= 配送先 destination）の識別子です。
type StreamID string

// Sequence は、stream 内の位置です。1 から始まり gap 無く増え、0 値は「位置を持たない」ことを
// 意味しません（未採番かどうかは SequenceAllocator.Current の ok で表します）。
type Sequence uint64

// String は、sequence の 10 進表記（ゼロ埋めなし）を返します。SSE の id と cursor はこの形です。
func (s Sequence) String() string {
	return strconv.FormatUint(uint64(s), 10)
}

// DeliveryEvent は、feature 中立な配送封筒です。EventID は outbox の message_id と同じ値で、
// 同じ (StreamID, Sequence) への再 append が冪等かどうかを判定する基準になります。
type DeliveryEvent struct {
	// EventID は、event の一意な識別子（= outbox message_id）です。
	EventID string
	// StreamID は、event が属する stream です。
	StreamID StreamID
	// Sequence は、stream 内の位置です。
	Sequence Sequence
	// Type は、event 種別（`<feature>.<noun>.<verb>.vN`）です。機構は解釈しません。
	Type string
	// OccurredAt は、event が起きた時刻（UTC）です。保持期間の判定に使います。
	OccurredAt time.Time
	// SchemaVersion は、Payload の schema 版です。
	SchemaVersion int
	// Payload は、feature が定義する JSON です。機構は中身を解釈しません。
	Payload json.RawMessage
}

// wireEvent は、DeliveryEvent の直列化形（client に届く JSON）です。
type wireEvent struct {
	EventID       string          `json:"eventId"`
	StreamID      StreamID        `json:"streamId"`
	Sequence      string          `json:"sequence"`
	Type          string          `json:"type"`
	OccurredAt    time.Time       `json:"occurredAt"`
	SchemaVersion int             `json:"schemaVersion"`
	Payload       json.RawMessage `json:"payload"`
}

// MarshalJSON は、client に届く形（sequence は 10 進文字列、時刻は RFC 3339）で直列化します。
func (e DeliveryEvent) MarshalJSON() ([]byte, error) {
	payload := e.Payload
	if len(payload) == 0 {
		payload = json.RawMessage("null")
	}

	b, err := json.Marshal(wireEvent{
		EventID:       e.EventID,
		StreamID:      e.StreamID,
		Sequence:      e.Sequence.String(),
		Type:          e.Type,
		OccurredAt:    e.OccurredAt.UTC(),
		SchemaVersion: e.SchemaVersion,
		Payload:       payload,
	})
	if err != nil {
		return nil, xerrors.Wrap(ErrInvalidEvent, err.Error())
	}

	return b, nil
}

// Validate は、封筒が保存・配送できる形かを判定します。emit する側は outbox へ書く前に呼び、
// store の実装は Append 前に呼びます。必須項目の欠落は ErrInvalidEvent、直列化後の大きさが
// MaxSerializedBytes を超えるものは ErrPayloadTooLarge を返します。
func (e DeliveryEvent) Validate() error {
	switch {
	case e.EventID == "":
		return xerrors.Wrap(ErrInvalidEvent, "event id is empty")
	case e.StreamID == "":
		return xerrors.Wrap(ErrInvalidEvent, "stream id is empty")
	case e.Sequence == 0:
		return xerrors.Wrap(ErrInvalidEvent, "sequence is zero")
	case e.Type == "":
		return xerrors.Wrap(ErrInvalidEvent, "type is empty")
	case e.OccurredAt.IsZero():
		return xerrors.Wrap(ErrInvalidEvent, "occurred at is zero")
	case len(e.Payload) > 0 && !json.Valid(e.Payload):
		return xerrors.Wrap(ErrInvalidEvent, "payload is not valid json")
	}

	b, err := e.MarshalJSON()
	if err != nil {
		return err
	}

	if len(b) > MaxSerializedBytes {
		return xerrors.Wrap(ErrPayloadTooLarge, "serialized event is "+strconv.Itoa(len(b))+" bytes")
	}

	return nil
}
