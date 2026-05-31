# clock

[English](README.md) | 日本語

現在時刻を取得するための `Clock` インターフェースを提供します。

```go
type Clock interface {
    Now() time.Time
}
```

## なぜ抽象化するのか

- 時刻依存ロジック（TTL、有効期限、スケジューリング）のテスト性を確保
- Domain / Usecase が `time.Now()` に直接依存しないようにする
- テストでモック差し替えにより決定論的な挙動を実現

## 実装

`internal/infrastructure/system/` に `time.Now()` を呼ぶ具体実装が配置されています。
