# ctxhelper

[English](README.md) | 日本語

ctxhelperは「contextの利用を制御する境界レイヤ」です。

このパッケージは、context.ContextとEchoフレームワークのコンテキストを操作するためのヘルパー関数を提供します。

## 実装方法

このパッケージのコードは手動で実装せず、コード生成によって作成します。

生成機構の詳細については、以下を参照してください：

- `scripts/genctxkey/README.ja.md`

## 使用方法

ctxkeyを追加する場合は、以下のように `generate.go` に定義を追加します。

```go
//go:generate go run ../../../scripts/genctxkey --name UserID --type string --out .
```

外部型を使用する場合：

```go
//go:generate go run ../../../scripts/genctxkey --name Authn --type "auth.Authn" --import github.com/your/project/internal/domain/auth --out .
```

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
--import github.com/your/project/internal/domain/auth
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

例外として、依存解決のための軽微な修正（import調整など）は許可されますが、
恒久対応は generator 側で行うことを推奨します。
