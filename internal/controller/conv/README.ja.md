# conv

[English](README.md) | 日本語

OpenAPI 生成型をドメイン型へ変換する境界ヘルパー。**controller 層のみ**が利用します。

## 目的

OpenAPI 生成型（`github.com/oapi-codegen/runtime/types`）を controller より下層へ漏らさないため、変換を本パッケージへ集約します。これにより生成型の import が境界に限定され、`usecase` / `domain` は生成型に依存しません。

## 注意点

- `UUID` は**エラーを返しません**。`openapi_types.UUID` は検証済みの 16 バイト配列という値型のため、`uuid.FromPrimitive` によるドメイン `pkg/uuid.UUID` への変換は無条件に成功し失敗しません（エラー分岐も panic もありません）。これによりハンドラに死んだエラー分岐を作りません。
- `Email` は `string` を、`EmailPtr` は `*string` を返します（`nil` 入力は `nil` を返す）。
