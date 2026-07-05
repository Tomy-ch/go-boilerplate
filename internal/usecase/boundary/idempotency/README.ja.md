# idempotency

[English](README.md) | 日本語

冪等性キーの永続化境界 `Store` を定義します（claim / replay / 409 / 422 判定）。
すべてのメソッドは `scope` 必須です（id 単独 lookup を持たない＝越境防止）。

```go
type Store interface {
    Claim(ctx context.Context, p ClaimParams) (claimed bool, err error)
    Get(ctx context.Context, scope, key string) (*Record, error)
    Complete(ctx context.Context, p CompleteParams) error
    DeleteExpired(ctx context.Context, cutoff time.Time, limit int32) (int64, error)
}
```

## なぜ抽象化するのか

- replay / 競合判定ロジックを、実データベースなしでテスト可能にする
- Usecase が sqlc や SQL の詳細ではなく永続化 port に依存するようにする
- テストでモック差し替えにより決定論的な挙動を実現
- 並行時の失敗を境界 sentinel（`ErrLockTimeout`、usecase 側で 409 へマップ）として表明する

## 実装

`internal/infrastructure/rdb/system_cqrs/idempotency/` に sqlc 生成クエリを用いた RDB 具体実装が配置されています。
