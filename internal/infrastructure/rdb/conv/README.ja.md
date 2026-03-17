# `conv` パッケージ

[English](README.md) | 日本語

概要: `database/sql` の nullable 型（`sql.NullString`, `sql.NullInt16`, `sql.NullInt64`, `sql.NullBool`, `sql.NullFloat64`, `sql.NullTime`）および `googleUUID.NullUUID` と、Go のポインタ型（`*string`, `*int64` など）を相互変換するユーティリティ関数を提供します。

このパッケージは主に **sqlc で生成されたコードと組み合わせて利用する Infrastructure 層のユーティリティ**です。

## 役割

このパッケージは DB の nullable 値とアプリケーションコードのポインタ値の橋渡しを行います。

主な役割:

- DB から取得した nullable 値をアプリケーションコードで扱いやすいポインタ型に変換する
- アプリケーション内で生成したポインタ値を `sql.NullXxx` 型に変換して DB 書き込みに渡す
- NULL 値の扱いを関数化してロジックの重複を防ぐ
- DB 固有型とアプリケーション型の境界を明確にする

これにより **NULL ハンドリングを一箇所に集約**できます。

## 対象型

このパッケージは次の nullable 型を扱います。

### database/sql

- `sql.NullString`
- `sql.NullInt16`
- `sql.NullInt64`
- `sql.NullBool`
- `sql.NullFloat64`
- `sql.NullTime`

### UUID

- `github.com/google/uuid.NullUUID`

UUID はアプリケーション側の `pkg/uuid.UUID` と相互変換されます。

## 提供する変換パターン

各型について **対称的な変換 API** を提供します。

```txt
NullXxx  → *T
*T       → NullXxx
T        → NullXxx
```

具体例（string）

```txt
StringPtrFromNull
NullStringFromPtr
NewNullString
```

この対称設計により、読み取りと書き込みの両方で同じルールを利用できます。

## 使用方法（振る舞いベース）

### 読み取り側

DB から取得した nullable 値をポインタに変換します。

```go
name := conv.StringPtrFromNull(row.Name)
```

変換ルール

```txt
NULL  → nil
value → *value
```

アプリケーション側では `nil` を使って NULL を判定できます。

### 書き込み側

ポインタを nullable 型に変換します。

```go
row.Name = conv.NullStringFromPtr(namePtr)
```

変換ルール

```txt
nil  → NULL
value → Valid=true
```

### 値から nullable を生成

非 NULL の値を nullable 型に変換します。

```go
row.Name = conv.NewNullString("alice")
```

## UUID 変換

UUID 変換は `googleUUID.NullUUID` とアプリケーションの `uuid.UUID` を相互変換します。

```go
UUIDPtrFromNull
NullUUIDFromPtr
NewNullUUID
```

注意:

- `UUIDPtrFromNull` は UUID の parse を伴うため **error を返す API** になっています。

例

```go
u, err := conv.UUIDPtrFromNull(row.UserID)
if err != nil {
    return err
}
```

## ポインタの安全性

nullable 型から生成されるポインタは、内部値のコピーを参照します。

つまり

```txt
&ns.String
```

のような値であり、DB ドライバの内部メモリを直接参照するわけではありません。

そのため通常のポインタ値として安全に扱えます。

## sqlc との関係

このパッケージは主に **sqlc 生成コードの補助ユーティリティ**として利用されます。

役割

```txt
sqlc generated code
        ↓
conv utilities
        ↓
application code
```

これにより

- DB 型
- アプリケーション型

の責務分離を実現します。

## テスト

各関数に対して単体テストが用意されています（`nullable_test.go`）。

テストでは `testify/require` を使用し、次の内容を検証します。

- `Valid` フラグが正しく設定される
- NULL → nil
- nil → NULL

## 必要度

### 本番運用

必須ではありませんが **強く推奨**されます。

理由

- NULL ハンドリングを集中管理できる
- 重複コードを防げる
- バグの混入を減らせる

### 開発 / テスト

推奨

理由

- NULL / 非 NULL のケースを簡潔に表現できる
- テストコードが読みやすくなる

## 無効化した場合の影響

直接的なランタイム障害は発生しない場合が多いですが

- NULL ハンドリングがコードベースに分散する
- 重複ロジックが増える
- バグの温床になる

可能性があります。

## 注意点

このパッケージは **変換ユーティリティのみを提供**します。

扱わない責務:

- DB クエリ実行
- トランザクション管理
- ドメインロジック

これらは

- Repository
- Usecase

などの別レイヤーで実装してください。

## 拡張ルール

新しい nullable 型を追加する場合は次の命名規則に従ってください。

```txt
XxxPtrFromNull
NullXxxFromPtr
NewNullXxx
```

例

```txt
DecimalPtrFromNull
NullDecimalFromPtr
NewNullDecimal
```

この規則により API の一貫性を保ちます。
