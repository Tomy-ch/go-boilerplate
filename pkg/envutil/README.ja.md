# envutil

[English](README.md) | 日本語

環境変数を扱うための小さなユーティリティを提供します。

## 公開 API

|関数|説明|
|---|---|
|`Override(key, value string) func()`|環境変数を一時的に設定し、復元関数を返します。復元時は、元値が存在した場合はその値へ、存在しなかった場合は Unset へ戻します。|

## 使い方

```go
restore := envutil.Override("DB_NAME", "test")
defer restore()
// ... DB_NAME が "test" の間に設定を読み込む ...
```

## 注意点

- 設定読み込みの間だけ特定の環境変数（例: `DB_NAME`）を差し替える用途に有用で、グローバル状態の残留を防ぎ冪等性を保ちます。
- `pkg/` は `internal/` や他の `pkg/` に依存できません（depguard で強制）。本パッケージは `os` のみ使用します。
