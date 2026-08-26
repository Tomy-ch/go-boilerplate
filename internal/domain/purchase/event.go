package purchase

import (
	"time"

	"go-boilerplate/pkg/uuid"
)

// 購入に起こりうる事象。
var (
	// EventCreated は、購入が作成されたという事象です。
	EventCreated = EventType{name: "created"}
	// EventPaid は、購入が支払われたという事象です。
	EventPaid = EventType{name: "paid"}
	// EventCanceled は、購入がキャンセルされたという事象です。
	EventCanceled = EventType{name: "canceled"}
	// EventShipped は、購入が発送されたという事象です。
	EventShipped = EventType{name: "shipped"}
	// EventDelivered は、購入が配達されたという事象です。
	EventDelivered = EventType{name: "delivered"}
)

// EventType は、購入に起きた事象の種別です。
//
// 名前は過去形でユビキタス言語の一部であり、ドメインが所有します。外部へ公開するときの
// version 付き文字列や JSON のフィールド名は転送契約（Published Language）であって、
// ドメインではなく詰め替え層が所有します。
type EventType struct {
	name string
}

// Event は、購入に起きた事実です。
//
// 起きたことを知っているのは遷移を起こした集約だけなので、事実の宣言は集約が行います。
// 遷移メソッドは成功したときにだけ Event を返すため、「状態は変わったがイベントが出ていない」
// 「イベントは出たが遷移していない」がコンパイル時に書けなくなります。
//
// 事実は変わらないため、生成後に変更する手段を持ちません。
type Event struct {
	typ        EventType
	purchaseID uuid.UUID
	occurredAt time.Time
}

// Name は、事象の名前を返します。
func (t EventType) Name() string { return t.name }

// newEvent は、購入の事象を生成します。遷移メソッドからのみ呼ばれます。
func newEvent(typ EventType, purchaseID uuid.UUID, occurredAt time.Time) Event {
	return Event{typ: typ, purchaseID: purchaseID, occurredAt: occurredAt}
}

// Type は、起きた事象の種別を返します。
func (e Event) Type() EventType { return e.typ }

// PurchaseID は、事象が起きた購入の ID を返します。
func (e Event) PurchaseID() uuid.UUID { return e.purchaseID }

// OccurredAt は、事象が起きた時刻を返します。遷移に用いた時刻と同一です。
func (e Event) OccurredAt() time.Time { return e.occurredAt }
