# ctxhelper

[English](README.md) | 日本語

ctxhelperは「contextの利用を制御する境界レイヤ」です。

このパッケージは、リクエストスコープの値を context.Context で受け渡すためのヘルパー関数を提供します。

## 実装方法

値を格納するだけの単純なキーはコード生成で作成します。生成機構の詳細については、以下を参照してください：

- `scripts/genctxkey/README.ja.md`

`Authn` のヘルパー（`authn.go`）は手書きです。OpenAPI の `AuthenticationFunc` は context を前方伝播できないため、認証前にミドルウェアが仕込む可変スロットで受け渡します。

## 提供するヘルパー

手書き（`authn.go`）— `Authn` スロット:

- `WithAuthn(ctx) context.Context` — 空の `Authn` スロットを仕込む（認証前に呼ぶ）
- `SetAuthn(ctx, authn) bool` — スロットへ書き込む。スロットが無ければ `false`
- `GetAuthn(ctx) (auth.Authn, bool)` — スロットから読む。未設定なら `ok=false`

生成（`genctxkey`、`generate.go` に定義）— リクエストスコープの真偽値フラグ。各名前は `context.Context` 用と `*echo.Context` 用のペアを提供します:

- `ErrorHandled` — `SetErrorHandled` / `GetErrorHandled`、`SetErrorHandledToEcho` / `GetErrorHandledFromEcho`
- `Recovered` — `SetRecovered` / `GetRecovered`、`SetRecoveredToEcho` / `GetRecoveredFromEcho`

## 使用方法

ctxkeyを追加する場合は、以下のように `generate.go` に定義を追加します。

```go
//go:generate go run ../../../scripts/genctxkey --name UserID --type string --out .
```

外部型を使用する場合：

```go
//go:generate go run ../../../scripts/genctxkey --name Actor --type "auth.Authn" --import go-boilerplate/internal/usecase/boundary/auth --out .
```

上記は外部型指定の構文例にすぎません。本パッケージの実際の `Authn` スロットは手書き（`authn.go`）であり、このコマンドでは生成**しません**。

その後、以下を実行します。

```bash
make gen-go-code
```

## type の指定方法

### 基本型 / 同一パッケージ型

```bash
--type string
--type UserID
```

- import は不要
- そのまま型として扱われます

### 外部パッケージの型

```bash
--type "auth.Authn"
--import go-boilerplate/internal/usecase/boundary/auth
```

- `--type` は Go の型式で指定
- `--import` でパッケージを明示
- `--alias` は任意

### 複雑な型

```bash
--type "*[]auth.Authn"
--type "map[string]auth.Authn"
```

- pointer / slice / map / generic に対応

## 注意

- import path のみ（例: `github.com/foo/bar`）は指定不可
- 外部型は必ず `--type` と `--import` をセットで指定

## 編集について

本ディレクトリ内の `.gen.go` ファイルは自動生成されたコードです。

- 原則として手動編集は禁止
- 変更は `scripts/genctxkey` を通して行ってください

手書きのヘルパー（`authn.go` など）は直接編集します。
