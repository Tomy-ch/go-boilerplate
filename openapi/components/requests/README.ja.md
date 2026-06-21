# OpenAPI リクエストスキーマ

[English](README.md) | 日本語

`openapi/components/requests/` は**エンドポイントのリクエストボディスキーマ**を格納します。これらは OpenAPI の素の **schema** であり（`requestBodies` の component object 型ではない。[`schemas/README.ja.md`](../schemas/README.ja.md) 参照）、パスからは `requestBody.content.<media>.schema.$ref` で参照します。

## `schemas/` との役割境界

- `schemas/` — 小さく再利用できる部品（例：`UserBaseInputRequest.yaml`）。
- `requests/` — **エンドポイント固有**の形。多くは `allOf` で基底部品を合成し、操作固有のフィールドを足す。

```yaml
# requests/users/UsersPostRequest.yaml
allOf:
  - $ref: '../../schemas/UserBaseInputRequest.yaml'
  - properties:
      password:        # 「ユーザー作成」固有のフィールド
        type: string
```

## ディレクトリ内容

```text
requests/
└── users/
    ├── UsersPostRequest.yaml      # ユーザー作成（基底 ＋ password）
    ├── UserPutRequest.yaml        # 全項目更新（全フィールド必須）
    ├── UserPatchRequest.yaml      # 部分更新（基底・フィールド任意）
    └── UserPasswordPutRequest.yaml # パスワード変更（現＋新）
```

> `users/` は**サンプル実装**です。独自リソースでも同じ構造に倣ってください。

## 命名規則

|要素|規則|例|
|---|---|---|
|ディレクトリ|リソース別に小文字|`users/`|
|ファイル名|PascalCase ＋ `Request`|`UsersPostRequest.yaml`|

## ルール

- フィールドを複製せず、`schemas/` の再利用基底を `allOf` で合成する。
- 不明フィールドを拒否するため `additionalProperties: false` を維持する（直接 or ラッパーに）。
- すべてのプロパティに `description` と `example` を記述する。
- 境界値（`maxLength` 等）は **ワイヤー契約**であり domain ルールではない — [入力境界値のオーナーシップ](../../boundary-ownership.ja.md) を参照。
