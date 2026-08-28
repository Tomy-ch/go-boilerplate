# outbox

トランザクショナル outbox の永続化境界 `Store` を定義します。
emit（usecase 層）と relay engine（controller 層）の双方がこの境界に依存します。

```go
type Store interface {
    Insert(ctx context.Context, p EmitParams) (uuid.UUID, error)
    ClaimPending(ctx context.Context, channel Channel, limit int32) ([]PendingMessage, error)
    MarkPublished(ctx context.Context, id int64) error
    MarkFailed(ctx context.Context, id int64, lastErr string, nextAttemptAt time.Time) error
    MarkDead(ctx context.Context, id int64) error
    ReplayDead(ctx context.Context, messageID *uuid.UUID) (int64, error)
    DeletePublished(ctx context.Context, cutoff time.Time, limit int32) (int64, error)
    OldestPendingCreatedAt(ctx context.Context, channel Channel) (createdAt time.Time, ok bool, err error)
    CountBlockedStreams(ctx context.Context, channel Channel) (int64, error)
}
```

行はちょうど 1 つの**配送チャネル**（`Channel`）に属し、relay プロセスはちょうど 1 チャネルを
claim します。したがって停止したチャネルが別のチャネルを止めることはありません。行はさらに
**順序キー**とその中の位置を持てます。`ClaimPending` は同一キーの先行位置が未 publish の間その行を
選ばないため、ストリームの配送は穴のある集合ではなく連続 prefix になります。

`MarkFailed` は dead 化しません。行が `dead` になるのは `MarkDead` を通る場合だけで、relay は
publisher が失敗を恒久的と分類したときにこれを呼びます
（[ADR-0058](../../../../docs/adr/0058-outbox-dead-on-permanent-error.md)）。

## なぜ抽象化するのか

- emit / relay / GC ロジックを、実データベースなしでテスト可能にする
- Usecase と relay engine が、sqlc や `FOR UPDATE SKIP LOCKED` SQL の詳細ではなく永続化 port に依存するようにする
- テストでモック差し替えにより決定論的な挙動を実現
- 業務 tx 内の `Insert`（usecase）と claim/mark/replay/SLI 呼び出し（relay engine）で 1 つの契約を共有する

## 実装

`internal/infrastructure/rdb/system_cqrs/outbox/` に sqlc 生成クエリを用いた RDB 具体実装が配置されています。
