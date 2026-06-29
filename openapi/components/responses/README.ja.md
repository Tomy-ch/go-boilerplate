# OpenAPI レスポンススキーマ

[English](README.md) | 日本語

`openapi/components/responses/` は**エンドポイントのレスポンスボディスキーマ**を格納します。これらは OpenAPI の素の **schema** であり（`responses` の component object 型ではない。[`schemas/README.ja.md`](../schemas/README.ja.md) 参照）、パスからは `responses.<status>.content.<media>.schema.$ref` で参照します。

## `schemas/` との役割境界

- `schemas/` — 小さく再利用できる部品（`UserResponse.yaml`, `PaginationMetadataResponse.yaml`, `CursorPaginationMetadataResponse.yaml`, `ErrorResponse.yaml`）。
- `responses/` — **エンドポイント固有**の形。多くはそれらを `allOf` で合成する（例：items 配列 ＋ ページネーションメタ）。

```yaml
# responses/users/UsersResponse.yaml
allOf:
  - type: object
    required: [users]
    properties:
      users:
        type: array
        items:
          $ref: '../../schemas/UserResponse.yaml'
  - $ref: '../../schemas/PaginationMetadataResponse.yaml'
```

## ディレクトリ内容

```text
responses/
├── users/
│   ├── UsersResponse.yaml          # ユーザー一覧（UserResponse[] ＋ オフセットページネーション）
│   ├── UsersFeedResponse.yaml      # ユーザーフィード（UserResponse[] ＋ カーソルページネーション）
│   ├── UsersSearchResponse.yaml    # 検索結果一覧
│   └── UsersSearchResponseItem.yaml # 検索結果1件（UserResponse ＋ registeredAt）
├── health-check/
│   ├── HealthResponse.yaml
│   └── ReadyResponse.yaml
└── version/
    └── VersionResponse.yaml
```

> `users/` は**サンプル実装**です。独自リソースでも同じ構造に倣ってください。
>
> 再利用する**エラー**レスポンスオブジェクト（共通の `400/401/403/404/500`）は、`ErrorResponse` と同じ [`schemas/errors/`](../schemas/README.ja.md) に置きます（ここではありません）。

## 命名規則

|要素|規則|例|
|---|---|---|
|ディレクトリ|リソース／関心事別に小文字|`users/`, `health-check/`, `version/`|
|ファイル名|PascalCase ＋ `Response`|`UsersResponse.yaml`, `VersionResponse.yaml`|

## ルール

- フィールドを複製せず、`schemas/` の再利用部品を `allOf` で合成する。
- 一覧レスポンスには対応するページネーションメタを含める（オフセットは `PaginationMetadataResponse`、カーソルは `CursorPaginationMetadataResponse`）。
- レスポンス制約は domain が出しうる値を包含する必要がある（`domain ⊆ レスポンス`） — [入力境界値のオーナーシップ](../../boundary-ownership.ja.md) を参照。
- すべてのプロパティに `description` と `example` を記述する。
