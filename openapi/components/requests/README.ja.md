# OpenAPI リクエストスキーマ

[English](README.md) | 日本語

`openapi/components/requests/` は**エンドポイントのリクエストボディスキーマ**を格納します。これらは OpenAPI の素の **schema** であり（`requestBodies` の component object 型ではない。[`schemas/README.ja.md`](../schemas/README.ja.md) 参照）、パスからは `requestBody.content.<media>.schema.$ref` で参照します。

## `schemas/` との役割境界

- `schemas/` — 小さく再利用できる部品（例：`UserBaseInputRequest.yaml`）。
- `requests/` — **エンドポイント固有**の形。多くは `allOf` で基底部品を合成し、操作固有の制約（例：`required` リスト）を足す。

```yaml
# requests/users/UsersPostRequest.yaml
allOf:
  - $ref: '../../schemas/UserBaseInputRequest.yaml'
  - required:          # 「ユーザー作成」で必須となるフィールド
      - firstName
      - lastName
      - email
```

## ディレクトリ内容

リソースごとに 1 つのディレクトリを置き、リソース名を付ける。そのリソースのリクエストボディを収める。

## 命名規則

|要素|規則|例|
|---|---|---|
|ディレクトリ|リソース別に小文字|`users/`|
|ファイル名|PascalCase ＋ `Request`|`UsersPostRequest.yaml`|

## ルール

- フィールドを複製せず、`schemas/` の再利用基底を `allOf` で合成する。
- 不明フィールドを拒否するため、`additionalProperties: false` は properties を宣言しているスキーマオブジェクト自身に置く。`additionalProperties` が見るのは**同じスキーマオブジェクト**で宣言された properties だけなので、`allOf` で合成する場合の置き場所は properties を持つ基底であり、`required` だけを足すラッパーに置くと properties が見えず全フィールドを拒否してしまう。
- ネストした要素スキーマ（配列の `items`）には自前の宣言が要る。外側のリクエストの宣言は届かない。両者は [`internal/architest`](../../../internal/architest/README.md) の `TestRequestBodyRejectsUnknownFields` が、リクエストボディから到達できる全スキーマについて検査する。
- すべてのプロパティに `description` と `example` を記述する。
- 境界値（`maxLength` 等）は **ワイヤー契約**であり domain ルールではない — [入力境界値のオーナーシップ](../../boundary-ownership.ja.md) を参照。
