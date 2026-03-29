# genctxkey

English | [日本語](README.ja.md)

`genctxkey` は、`context.Context` および `echo.Context` に対する値の受け渡しを型安全に行うためのコードを生成するツールです。

## 概要

context に対する値の格納・取得を、以下のような問題を避けながら統一的に扱うことを目的としています。

- stringキーによる衝突
- 型安全性の欠如
- 実装のばらつき

本ツールはこれらを解決するためのヘルパー関数群を自動生成します。

## 生成されるコード

以下のようなコードが生成されます。

- context.Context 用
  - `SetXxx`
  - `GetXxx`
- echo.Context 用
  - `SetXxxToEcho`
  - `GetXxxFromEcho`

## 使用方法

### 1. generate.go に定義

```go
package ctxhelper

//go:generate go run ../../../scripts/genctxkey --name authn --type github.com/your/project/internal/domain/auth.Authn --out .
```

### 2. コード生成

```sh
make gen-go-code
```

## type の指定方法

### 基本型 / 同一パッケージ型

```sh
--type string
--type UserID
```

- import は不要
- そのまま型として扱われます

### 外部パッケージの型（推奨）

```sh
--type github.com/your/project/internal/domain/auth.Authn
```

#### フォーマット

```text
<import-path>.<Type>
```

### 生成時の挙動

```text
github.com/your/project/internal/domain/auth.Authn
```

↓

```text
import auth "github.com/your/project/internal/domain/auth"
```

↓

```text
auth.Authn
```

として扱われます。

### 注意

以下は無効です：

```sh
--type github.com/foo/bar
```

必ず型名まで含めて指定してください。

## 出力仕様

- ファイル名はすべて小文字
  - 例: `authn_ctx.gen.go`
- テストファイルも自動生成
  - 例: `authn_ctx_test.go`

## 設計方針

### 1. generator に責務を集中

- import 解決
- alias 決定
- 型の整形

すべて generator 側で行います。

### 2. template は最小責務

- ロジックは持たない
- 表示のみ

### 3. deterministic（再現可能）

- 推測処理なし（goimports未使用）
- 常に同一結果を生成

## 編集について

生成される `.gen.go` ファイルは自動生成コードです。

- 原則として手動編集は禁止
- 変更は generator を通して行う

### 例外

依存解決の都合で以下は許可されます：

- import の調整
- alias の変更

ただし、

- 再生成時に上書きされる可能性あり
- 恒久対応は generator 側で行うこと

## CIとの関係

- `ctxhelper` の generate は CIでは実行されません
- 生成はローカルで実施し、結果をコミットする前提です

## 補足

本ツールは以下の思想に基づいています：

- context は「制御された境界」で使用する
- 直接操作は禁止し、必ずラッパーを通す
- 生成によって一貫性を担保する

## まとめ

|項目|方針|
|------|------|
|型指定|フルパス推奨|
|import|generatorで解決|
|生成|再現可能|
|編集|原則禁止（例外あり）|

本ツールは、context利用の統一と安全性を保証するための基盤コンポーネントです。
