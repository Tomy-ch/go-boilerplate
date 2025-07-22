# OpenAPI パラメータ定義ガイド

このドキュメントでは、OpenAPI における `$ref` の書き方、および `parameters` セクションのベストプラクティスを整理しています。

## パラメータ定義の `$ref` は必ずフラグメント付きで

OpenAPI 仕様では `$ref` は **ファイルパス + JSON Pointer（# 以降）** で構成される必要があります。

```yaml
$ref: './components/parameters/common.yaml#/parameters/PageParam'
```

## ファイル構造は以下を推奨

### 共通系パラメータ（全エンドポイントで使い回すもの）

- 例: `page`, `per_page`, `offset`, `limit` など
- ファイル: `components/parameters/common.yaml`

### ドメイン別パラメータ（user_id など）

- 例: `user_id`, `project_id`
- ファイル: `components/parameters/user.yaml`, `project.yaml` などに分離して管理

## 統合ファイルの例（pagination.yaml）

ファイルの分割は、その目的とドメインに基づいて行うべきです。
下記は、ページネーションに関するパラメータをまとめた、`pagination.yaml` です。

```yaml
parameters:
  PageParam:
    name: page
    in: query
    description: ページ番号
    required: false
    schema:
      type: integer
      minimum: 1
      example: 1

  PerPageParam:
    name: per_page
    in: query
    description: 1ページあたりの取得件数
    required: false
    schema:
      type: integer
      minimum: 1
      maximum: 100
      example: 10
```

## `$ref` の統一例

```yaml
parameters:
  - $ref: '../../components/parameters/pagination.yaml#/parameters/PageParam'
  - $ref: '../../components/parameters/pagination.yaml#/parameters/PerPageParam'
  - $ref: '../../components/parameters/user.yaml#/parameters/UserIdParam'
```

## paths 参照時の `$ref` とエスケープルール

OpenAPI の `paths` を他ファイルに分割する場合、パスに含まれる特殊文字（特にスラッシュ `/` や波括弧 `{}`）は `$ref` で参照する際にエスケープが必要です。

### エスケープルール（JSON Pointer準拠）

| 文字 | エスケープ後 |
|------|--------------|
| `/`  | `~1`         |
| `~`  | `~0`         |
| `{`  | そのままでOK |
| `}`  | そのままでOK |

### 例：`/v1/users/{user_id}` を参照する場合

```yaml
paths:
  /v1/users/{user_id}:
    $ref: './paths/v1/users.yaml#/paths/~1v1~1users~1{user_id}'
```

- `/` を `~1` に置き換える必要があるため、`/v1/users/{user_id}` は `~1v1~1users~1{user_id}` に変換されます。
- `{}` の中身はそのままでOK。

### 注意

- Swagger UI やバリデーションツールで正しく動作させるためには、**エスケープされた形式が必須**です。
- ファイルを分割して path-level `$ref` を使う場合は、常に **`#/paths/...` 形式**での参照にしてください。

## 命名規則まとめ

| 用途       | 命名規則       | 例                        |
|------------|----------------|---------------------------|
| パス名     | スラッグ形式    | `/v1/users/{user_id}`     |
| ディレクトリ | snake_case     | `components/parameters/`  |
| ファイル名 | PascalCase     | `UserIdParam.yaml`        |
| 定義名     | PascalCase     | `UserIdParam`             |
| `$ref`     | PascalCase + fragment | `#/parameters/UserIdParam` |
| フラグメントにエスケープ文字がある場合 | `~1`, `~0` など | `#/paths/~1v1~1users~1{user_id}` |

## 注意点

- `$ref: './xxx.yaml'` のように **フラグメントなし**の参照は非推奨です（複数定義に拡張したとき破綻するため）
- 将来的にドメイン追加や GraphQL API 併存などが起きた際にも、定義の一貫性を保つためこの構造が有効です
