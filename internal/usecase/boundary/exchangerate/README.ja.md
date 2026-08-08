# exchangerate

[English](README.md) | 日本語

外部為替レート取得サービスへの意味的 gateway として振る舞う `Gateway` インターフェースを提供します（DTO モードのサンプル）。

```go
type Gateway interface {
    GetRate(ctx context.Context, base, quote string) (*Rate, error)
}

type Rate struct {
    Base  string
    Quote string
    Value decimal.Decimal
}
```

`Value` は `float64` ではなく正確な `pkg/decimal.Decimal` です。レートはマネー経路の乗数であり、
float は取込時点で値を破壊するためです（[ADR-0034](../../../../docs/adr/0034-two-scale-quantity-model.md)）。

## なぜ抽象化するのか

- 外部レートプロバイダに依存する usecase を、実 HTTP 呼び出しなしでテスト可能にする
- Usecase が `net/http` やベンダー SDK ではなく意味的 port に依存するようにする
- テストでモック差し替えにより決定論的な挙動を実現
- 境界でトランスポート失敗を `apperror` sentinel（`ErrUnavailable` / `ErrNotFound` 等）へ変換する

## 実装

`internal/infrastructure/webapi/exchangerate/` に外部サービスを HTTP で呼び出す具体実装が配置されています。
