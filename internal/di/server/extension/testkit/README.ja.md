# testkit

[English](README.md) | 日本語

`testkit` は、di server extension のモジュールテスト（兄弟パッケージである `inbound` / `outbound` / `security` / `instrumentation` / `nonprod`）向けの **共有テストヘルパー**を提供するパッケージです。

使い捨ての `fx` アプリを構築し、モジュールが fx グループに何を provide するかを検証する定型処理を集約することで、各 extension の `module_graph_test.go` を小さく一貫した形に保ちます。

## ヘルパー

|ヘルパー|説明|
|---|---|
|`RequireProvidesOne[T any](t, group, opts...)`|与えたモジュール群が指定 fx `group` に型 `T` の要素をちょうど 1 件 provide することを検証|

## 使い方

```go
testkit.RequireProvidesOne[extension.UseMiddleware](t, "middlewares.use", security.Module())
```

## 注意点

- `T`（provide される要素型）に対するジェネリック。`group` は fx グループタグ（例: `middlewares.use`、`server.configurators`）
- `opts` から `fx.App` を構築し、`Start` / `Stop` を実行して、populate されたスライスの要素数がちょうど 1 件であることを検証する
- `fx.NopLogger` は構成時のログ出力を抑えるだけで、検証結果には影響しない
- テスト専用ヘルパー — **本番の DI 配線で使用してはならない**
