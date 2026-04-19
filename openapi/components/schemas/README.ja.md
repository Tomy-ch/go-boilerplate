# OpenAPI スキーマ

[English](README.md) | 日本語

`openapi/components/schemas/` は、**再利用可能な OpenAPI スキーマ定義**（リクエスト・レスポンス・セキュリティスキームのデータ構造）を格納するディレクトリです。

## ディレクトリ内容

|ファイル|種別|説明|
|---|---|---|
|`ErrorResponse.yaml`|レスポンス|統一エラーレスポンススキーマ（code / message / details / request_id）|
|`PaginationMetadataResponse.yaml`|レスポンス|ページネーションメタデータ（total / limit / offset）|
|`BasicAuth.yaml`|セキュリティ|HTTP Basic 認証スキーム|
|`BearerAuth.yaml`|セキュリティ|HTTP Bearer（JWT）認証スキーム|
|`UserBaseInputRequest.yaml`|リクエスト|ユーザー入力フィールド（サンプル）|
|`UserResponse.yaml`|レスポンス|ユーザーレスポンスフィールド（サンプル）|

> `User*` ファイルは**サンプル実装**です。独自のスキーマを構築する際の命名・構造の参考として使用してください。

## 設計ポリシー

### 1ファイル = 1スキーマ

各ファイルは単一のトップレベルスキーマを定義します。1ファイルに複数のスキーマを含めないでください。

```yaml
# Good: UserResponse.yaml
type: object
required:
  - name
  - email
properties:
  name:
    type: string
  email:
    type: string
    format: email
```

### 命名規則

|要素|規則|例|
|---|---|---|
|ファイル名|PascalCase|`ErrorResponse.yaml`, `UserBaseInputRequest.yaml`|
|リクエストスキーマ|`*Request.yaml`|`UserBaseInputRequest.yaml`|
|レスポンススキーマ|`*Response.yaml`|`UserResponse.yaml`, `ErrorResponse.yaml`|
|セキュリティスキーム|説明的な名前|`BasicAuth.yaml`, `BearerAuth.yaml`|

### リクエスト / レスポンスを schemas に定義する理由

`redocly` のバンドルと `oapi-codegen` の生成の制約により、リクエストボディとレスポンスは `requestBodies/` や `responses/` ではなく **schemas として定義**します。

### $ref の使い方

すべての `$ref` 参照は `redocly bundle` との互換性のため、**相対 YAML パス**を使用します（`#/components/...` フラグメント形式は使用しない）。

```yaml
# パス定義から
schema:
  $ref: '../components/schemas/UserResponse.yaml'
```

### PATCH 対応

PATCH 操作には `allOf` を使って専用のラッパースキーマを作成します：

```yaml
# UserPatchRequest.yaml
allOf:
  - $ref: './UserBaseInputRequest.yaml'
additionalProperties: false
```

入力構造と操作のセマンティクスを分離します。

## コアスキーマ

### ErrorResponse

全エンドポイントで使用される統一エラーレスポンス：

```yaml
type: object
required: [code, message, request_id]
properties:
  code:        # 機械可読なエラーコード（例: BAD_REQUEST）
  message:     # ユーザー向けエラーメッセージ
  details:     # オプションの詳細文字列配列
  request_id:  # リクエスト追跡 ID
```

Go 側の `response.HTTPErrorResponse` にマッピング — 詳細は `internal/controller/error/response/` を参照。

### PaginationMetadataResponse

一覧エンドポイントで返されるページネーションメタデータ：

```yaml
type: object
required: [total, limit, offset]
properties:
  total:   # 全件数
  limit:   # 1ページあたりの件数
  offset:  # 現在のオフセット
```

## ルール

- パス定義内でインラインにスキーマを定義しない — 必ず `schemas/` に切り出す
- リクエストとレスポンスの兼用スキーマは極力避ける
- すべてのプロパティに `description` と `example` を記述する
- `required` で必須フィールドを明示する
- リクエストスキーマには `additionalProperties: false` を設定して不明フィールドを拒否する

## チェックリスト

- [ ] 1ファイル = 1スキーマになっているか
- [ ] ファイル名がスキーマの目的と一致しているか（PascalCase）
- [ ] `$ref` が相対パス形式で統一されているか
- [ ] PATCH には専用ラッパースキーマを作成しているか
- [ ] すべてのプロパティに `description` と `example` があるか
