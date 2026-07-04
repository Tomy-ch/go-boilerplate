---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [contract, openapi, codegen]
---

# ADR-0011: oapi-codegen の strict-server モードでタグ/ハンドラーごとに生成する

English canonical: [0011-oapi-codegen-strict-server.md](../../adr/0011-oapi-codegen-strict-server.md)

## ステータス

accepted

## 背景

[ADR-0009](0009-openapi-first.ja.md) はサーバーコードを手書きするのではなく OpenAPI 仕様から生成することを要求している。問題はその生成のスコープをどうするかである。単一のグローバル生成は、すべてのハンドラーが一緒に実装しなければならない 1 つの大きなインターフェースを生成し、それによって独立したハンドラーパッケージ間に密な結合が生じる。さらに、デフォルトの oapi-codegen サーバーモードは各ハンドラーメソッドに生の `echo.Context` を渡すため、リクエストのアンマーシャリングとレスポンスのシリアライズはハンドラー実装に委ねられる——これはすべてのハンドラー作成者が一貫して再現しなければならないボイラープレートである。

## 決定

**oapi-codegen を `--generate=echo-server,strict-server` で使用し**、OpenAPI タグごとにスコープを絞る。これによって各ハンドラーパッケージは自身のタグのインターフェースのみを持つ。

各ハンドラーファイルの先頭に 2 つの `go:generate` ディレクティブを記述する：

```go
//go:generate oapi-codegen --include-tags=<tag> --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=<tag> --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml
```

`--include-tags=<tag>` フラグはバンドルされた仕様をこのハンドラーパッケージに属するオペレーションのみにフィルタリングする。生成された `gen/` サブパッケージには以下が含まれる：

- `type.gen.go` — タグのリクエスト/レスポンス型。
- `server.gen.go` — タグの `StrictServerInterface` と、ルートを登録して strict ハンドラーを呼び出す `echo-server` グルーコード。

**strict-server モード**では、生成された `StrictServerInterface` は型付きの `RequestObject`（既にアンマーシャリングされたパラメーターとボディを持つ）を受け取り、型付きの `ResponseObject` を返す。ハンドラー実装は `c.Bind`、`c.JSON` などを直接呼び出さない——strict グルー層がすべてのマーシャリングを処理する。例：

```go
type StrictServerInterface interface {
    GetUsers(ctx context.Context, request GetUsersRequestObject) (GetUsersResponseObject, error)
    PostUsers(ctx context.Context, request PostUsersRequestObject) (PostUsersResponseObject, error)
}
```

`NewStrictHandler(ssi StrictServerInterface, middlewares []StrictMiddlewareFunc)` が strict インターフェースを、Echo のルート登録が期待する通常の `ServerInterface` に適合させる。

## 影響

### ポジティブな影響

- 各ハンドラーパッケージは独立して生成、コンパイル、テストされる。新しいタグを追加しても既存のハンドラーパッケージには影響しない。
- strict-server モードでハンドラーごとのボイラープレートが排除される。アンマーシャリングとシリアライズは生成コードが処理する。
- ハンドラーメソッドは完全に型付けされた Go 構造体を受け取るため、型の不一致はランタイムパニックではなくコンパイルエラーになる。
- 単一パッケージの再生成は高速（`go generate ./internal/controller/handler/<pkg>/`）。

### ネガティブな影響

- すべてのハンドラーパッケージが独自の `//go:generate` ディレクティブを宣言しなければならない。単一のグローバル生成ターゲットは存在しない。
- 生成された strict グルー層は Echo とハンドラー実装の間に間接層を追加する。strict-server モードに不慣れな開発者はコールフローが分かりにくいと感じる場合がある。
- 再生成にはバンドルされた `openapi.gen.yaml` が先に存在する必要がある（[ADR-0010](0010-redocly-modular-spec-pipeline.ja.md) 参照）。

## 検討した代替案

### 単一グローバル生成（全タグを 1 パッケージに）

設定がシンプルになる。却下：すべてのハンドラーパッケージを共有インターフェースに結合する——1 つのエンドポイントを追加・変更するとすべてが再コンパイルされ、責任境界が曖昧になる。

### プレーンな echo-server モード（strict-server なし）

タグごとにスコープを絞るが、各ハンドラーメソッドは生の `echo.Context` を受け取り、自身でバインディングとシリアライズを行う必要がある。却下：すべてのハンドラーで同じボイラープレートを再現することになり、一貫性のないエラー処理の余地が残る。

## 補足

- `//go:generate` ディレクティブは [`internal/controller/handler/`](../../../internal/controller/handler/) 配下の各 `*_handler.go` ファイルの先頭にある。
- 生成ファイルは各ハンドラーディレクトリの `gen/` サブパッケージにあり、手動で編集してはならない。
- ハンドラー層の契約（`operationId` ごとに 1 つの `StrictServerInterface` メソッド、ハンドラー内にビジネスロジックなし）は [`docs/rules.md`](../../rules.md) のアーキテクチャルールで強制される。
- 親の決定: [ADR-0009](0009-openapi-first.ja.md)（OpenAPI ファースト）。
- 仕様バンドルの前提条件: [ADR-0010](0010-redocly-modular-spec-pipeline.ja.md)。
