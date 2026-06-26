# clock

[English](README.md) | 日本語

現在時刻を取得する `Clock` インターフェースと、待機のための `Sleeper` インターフェースを提供します。

```go
type Clock interface {
    Now() time.Time
}

type Sleeper interface {
    Sleep(ctx context.Context, d time.Duration) error
}
```

## なぜ抽象化するのか

- 時刻依存ロジック（TTL、有効期限、スケジューリング）のテスト性を確保
- Domain / Usecase が `time.Now()` に直接依存しないようにする
- テストでモック差し替えにより決定論的な挙動を実現
- `Sleeper` により backoff の待機を注入可能にし、実時間 sleep なしでリトライをテストできる。利用者: レジリエント HTTP クライアント（`internal/infrastructure/httpclient`）とトランザクションマネージャのリトライ（`internal/infrastructure/rdb/driver`, H1）。

## 実装

`internal/infrastructure/system/` に `time.Now()` / `time.After`（ctx 対応）を呼ぶ具体実装が配置されています。
