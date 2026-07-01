# publisher

[English](README.md) | 日本語

ドメインイベントの outbound publish 境界 `Publisher` と、publish 先非依存のメッセージ封筒を定義します。
relay engine（controller 層）と publish adapter（infrastructure 層）の双方がこの境界に依存します。

```go
type Publisher interface {
    Publish(ctx context.Context, m Message) error
}

type Message struct {
    MessageID uuid.UUID
    EventType string
    Payload   []byte
    Headers   map[string]string
}
```

## なぜ抽象化するのか

- relay ロジックを、実メッセージを送信せずにテスト可能にする
- relay engine が `net/http` や broker SDK ではなく中立な port に依存するようにする
- テストでモック差し替えにより決定論的な挙動を実現
- 中立な封筒により outbox 行の表現を publish 先の substrate から分離する（at-least-once: `Publish` 失敗は次 poll で再送）

## 実装

`internal/infrastructure/publisher/` にメッセージを HTTP で publish する具体実装が配置されています。
