//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package realtime

import "context"

const (
	// KindWakeup は「stream を自分の cursor 以降で読み直せ」の合図です。状態は運びません（ADR-0073）。
	KindWakeup NotificationKind = "wakeup"
	// KindRevocation は、subject の destination への権利が取り下げられたことの通知です（ADR-0074）。
	KindRevocation NotificationKind = "revocation"
)

// NotificationKind は、instance の受信先に届く通知の種別です。
type NotificationKind string

// Wakeup は、EventLog に event が 1 件増えたことを全 instance へ伝える通知です。
// 重複して届いても同じ読み直しに畳まれ、欠落は periodic catch-up が覆うので、受け取る側は冪等でなければなりません。
type Wakeup struct {
	// EventID は、増えた event の識別子です（outbox の message_id と同じ）。
	EventID string
	// StreamID は、読み直す stream です。
	StreamID StreamID
	// Sequence は、増えた event の位置です。受け取る側はこれ以下の cursor を持つ接続だけを起こせます。
	Sequence Sequence
}

// Revocation は、subject × destination の失効通知です。受け取った instance は該当する接続を閉じます。
type Revocation struct {
	Subject     string
	Destination StreamID
}

// Notification は、instance の受信先から取り出した 1 件です。Kind に応じて Wakeup か Revocation のどちらかが埋まります。
// 種別が読めない通知は Kind が空のまま返り、受け取る側が Delete で取り除きます（残すと再配送され続けるため）。
// Receipt は受信先固有の削除鍵で、機構は中身を解釈しません。
type Notification struct {
	Kind       NotificationKind
	Wakeup     Wakeup
	Revocation Revocation
	Receipt    string
}

// InstanceSubscription は、serve instance 固有の受信先の境界です。受信先の作成から削除までを 1 つの instance の
// 生存期間に閉じ、fan-out の substrate（topic / queue / subscription）の語彙は外に出しません。
// 失敗は apperror sentinel で返します。
type InstanceSubscription interface {
	// Provision は、id の instance 専用の受信先を作り、fan-out 元へ登録します。同じ id で繰り返し呼んでも 1 組に収束します。
	Provision(ctx context.Context, id InstanceID) error
	// Receive は、届いた通知を最大 limit 件返します。通知が無ければ有限時間待ってから空を返し、ctx 完了でも返ります。
	Receive(ctx context.Context, limit int) ([]Notification, error)
	// Delete は、処理済みの 1 件を受信先から取り除きます。取り除かなかった通知は再配送されます。
	Delete(ctx context.Context, n Notification) error
	// Teardown は、登録を解除して受信先を削除します。Provision していなければ何もしません。
	Teardown(ctx context.Context) error
}
