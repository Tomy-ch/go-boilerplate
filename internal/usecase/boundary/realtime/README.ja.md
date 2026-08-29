# realtime

Realtime Delivery（[`docs/design/realtime-delivery.ja.md`](../../../../docs/design/realtime-delivery.ja.md)）の seam です。
feature 中立な封筒 `DeliveryEvent` と、それを保存・配送する側が実装する境界を置きます。usecase 側
（`internal/usecase/realtime/`、feature の realtime adapter）はこの package にだけ依存し、DynamoDB の
store・PostgreSQL の sequence allocator・SNS / SQS の fan-out adapter（infrastructure）が実装します。vendor の語彙（table /
partition / TTL）も feature の語彙（会話 / メッセージ / operator）もここには現れません。

```go
// 封筒。EventID は outbox の message_id と同じで、冪等性の判定基準になる。
type DeliveryEvent struct {
    EventID       string
    StreamID      StreamID
    Sequence      Sequence        // int64。String() が 10 進の wire 形
    Type          string
    OccurredAt    time.Time
    SchemaVersion int
    Payload       json.RawMessage
}

func (e DeliveryEvent) Validate() error          // ErrInvalidEvent / ErrPayloadTooLarge（直列化後 64 KiB）
func (e DeliveryEvent) MarshalJSON() ([]byte, error)

type EventLogStore interface {
    Append(ctx context.Context, event DeliveryEvent) error                        // EventID で冪等、違えば ErrSequenceConflict
    ReadAfter(ctx context.Context, q ReadAfterQuery) (ReadAfterResult, error)     // 強い一貫性、昇順、HasMore
    Latest(ctx context.Context, streamID StreamID) (DeliveryEvent, bool, error)
    Find(ctx context.Context, streamID StreamID, seq Sequence) (DeliveryEvent, bool, error)
}

type StreamTicketStore interface {
    Save(ctx context.Context, ticket StreamTicket) error
    Find(ctx context.Context, hash TicketHash, asOf time.Time) (StreamTicket, bool, error)  // 期限切れは ok=false
    Invalidate(ctx context.Context, subject string, destination StreamID) error
}

type InstanceLeaseStore interface {
    Heartbeat(ctx context.Context, lease InstanceLease) error
    Delete(ctx context.Context, id InstanceID) error
    ListExpired(ctx context.Context, asOf time.Time) ([]InstanceLease, error)
    AcquireCleanup(ctx context.Context, claim CleanupClaim) (bool, error)         // 条件付き。false = 他者が引き受け済み
}

type SecretGenerator interface {
    Generate() (string, error)   // 256 bit の不透明な ticket 生値
}

// fan-out 上の instance 自身の queue（serve instance ごとに 1 つ、起動時に作り停止時に消す）。
type InstanceSubscription interface {
    Provision(ctx context.Context, id InstanceID) error            // id ごとに冪等
    Receive(ctx context.Context, limit int) ([]Notification, error) // 有界の待ち。ctx で返る
    Delete(ctx context.Context, n Notification) error               // 消さなかった通知は再配送される
    Teardown(ctx context.Context) error                             // 登録解除してから削除。provision していなければ no-op
}

type Notification struct {            // Kind がどちらに値が入っているかを決める
    Kind       NotificationKind      // KindWakeup | KindRevocation | KindUnknown（""。読めないので削除する）
    Wakeup     Wakeup                // EventID / StreamID / Sequence — 「cursor の先から stream を読み直せ」
    Revocation Revocation            // Subject / Destination — 「その subject の接続を閉じろ」
    Receipt    string                // substrate 固有の削除キー。ここでは不透明
}
```

## 境界が担う不変条件

| 不変条件 | 強制される場所 |
| --- | --- |
| 直列化した event は `MaxSerializedBytes`（64 KiB）以下 — payload 単体でなく封筒全体 | `DeliveryEvent.Validate`。emit する adapter が outbox に書く前に呼び、`EventLogStore.Append` の実装も保存前に呼ぶ |
| `Sequence` は wire 上 10 進、stream 内で gap 無し、0 値に意味は無い — 「未採番」は `SequenceAllocator.Current` の `ok` であって番兵値ではない | `Sequence.String`、allocator と ADR-0072 |
| 同じ位置への同じ `EventID` の再 append は成功、異なる `EventID` は `ErrSequenceConflict` | `EventLogStore.Append`（outbox relay は特別扱い無しで retry できる） |
| cursor は ticket 自身の `Destination` に対してだけ意味を持つ | `StreamTicket.Destination`。stream handler が比較する |
| 期限の判定は呼び出し側の時計（`asOf`）で行い、store の掃除を正本にしない | `StreamTicketStore.Find`、`InstanceLeaseStore.ListExpired` / `AcquireCleanup` |

`EventLogRetention`（7 日）は store（item の掃除）と usecase（replay floor の導出）の両方が読むためここに
定義します。ticket TTL、lease の heartbeat / expiry / cleanup margin は `internal/usecase/realtime/` が持ち、
store はその結果の時刻だけを受け取り、数値は知りません。

## `SecretGenerator` を別に持つ理由

`boundary/token` は cart のセッション追跡のために存在し、sample feature と一緒に削除されます
（`scripts/setup/remove-sample-api/sample-manifest.ts`）。Realtime Delivery はその削除後も compile /
test できなければならないので、乱数の seam を自前で持ちます。実装は
`internal/infrastructure/realtimesecret/` です。

## 実装

| 境界 | 実装 |
| --- | --- |
| `EventLogStore` | `internal/infrastructure/eventlog/dynamodb/` |
| `StreamTicketStore` | `internal/infrastructure/streamticket/dynamodb/` |
| `InstanceLeaseStore` | `internal/infrastructure/instancelease/dynamodb/` |
| `SecretGenerator` | `internal/infrastructure/realtimesecret/` |
| `SequenceAllocator`（`sequence.go`） | `internal/infrastructure/rdb/system_cqrs/realtime/` |
| `RevocationNotifier`（`revocation.go`） | `internal/infrastructure/realtime/aws/`（wakeup と同じ topic へ revocation を publish する）。`usecase/realtime.AccessRevoker` が ticket を無効にした後に呼ぶ |
| `InstanceSubscription`（`fanout.go`） | `internal/infrastructure/realtime/aws/`（instance の SQS queue とその SNS subscription）。`internal/controller/realtime/` の consumer エンジンが駆動する |

mock はファイルごとに `mock/` へ生成します（`go generate ./...`）。
