# envutil

[English](README.md) | 日本語

環境変数を扱うための小さなユーティリティを提供します。

## 使い方

```go
restore, err := envutil.Override("SOME_KEY", "value")
if err != nil {
    return err
}
defer restore()
// ... SOME_KEY が "value" の間に設定を読み込む ...
```

## 注意点

- 設定読み込みの間だけ特定の環境変数を差し替える用途に有用で、グローバル状態の残留を防ぎ冪等性を保ちます。
- `pkg/` は `internal/` や他の `pkg/` に依存できません（例外として `pkg/xerrors` のみ許可。depguard で強制）。本パッケージは `os` と `pkg/xerrors` を使用します。
