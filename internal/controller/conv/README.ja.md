# conv

[English](README.md) | 日本語

OpenAPI 生成型をドメイン型へ変換する境界ヘルパー。**controller 層のみ**が利用します。

## 目的

OpenAPI 生成型（`github.com/oapi-codegen/runtime/types`）を controller より下層へ漏らさないため、変換を本パッケージへ集約します。これにより生成型の import が境界に限定され、`usecase` / `domain` は生成型に依存しません。

## 公開 API

|関数|説明|
|---|---|
|`UUID(p openapi_types.UUID) uuid.UUID`|生成 UUID（path / query パラメータ）をドメイン `uuid.UUID` へ変換|

## 注意点

- `UUID` は**エラーを返しません**。値は echo のバインド時に UUID 形式が検証済みのため必ず変換できます。万一変換できない場合は到達してはならない不変条件違反（バグ）なので、エラーを返さず **panic** します（ハンドラに死んだエラー分岐を作らないため）。
- この変換を省くために `pkg/uuid` へ検証バイパスのコンストラクタを足さないこと。invalid 時 panic のアサーションは意図的で、テストでは文字列入力ヘルパー経由で網羅しています。
