# OpenAPI パラメータ

[English](README.md) | 日本語

`openapi/components/parameters/` は、**再利用可能な OpenAPI パラメータ定義**を関心事ごとに整理して格納するディレクトリです。

## ディレクトリ構成

```text
parameters/
├── pagination/         # ページネーションパラメータ（エンドポイント共通）
│   ├── PageParam.yaml
│   └── PerPageParam.yaml
├── search/             # 検索パラメータ（エンドポイント共通）
│   ├── KeywordParam.yaml
│   └── ActiveParam.yaml
└── user/               # ユーザー固有パラメータ（サンプル）
    └── UserIdParam.yaml
```

> `user/` は**サンプル実装**です。サービス構築時には参考にした上で、必要に応じて置き換え・削除してください。

## 利用可能なパラメータ

### pagination

|ファイル|パラメータ|型|説明|
|---|---|---|---|
|`PageParam.yaml`|`page` (query)|integer|ページ番号（1始まり、デフォルト: 1）|
|`PerPageParam.yaml`|`per_page` (query)|integer|1ページあたりの件数（デフォルト: 10、最大: 100）|

### search

|ファイル|パラメータ|型|説明|
|---|---|---|---|
|`KeywordParam.yaml`|`keyword` (query)|string|全文検索キーワード|
|`ActiveParam.yaml`|`active` (query)|boolean|有効状態でフィルタ（true / false / 省略で全件）|

### user（サンプル）

|ファイル|パラメータ|型|説明|
|---|---|---|---|
|`UserIdParam.yaml`|`user_id` (path)|string (uuid)|ユーザー UUID|

## 使い方

エンドポイント定義で `$ref` を使ってパラメータを参照します：

```yaml
parameters:
  - $ref: '../components/parameters/pagination/PageParam.yaml'
  - $ref: '../components/parameters/pagination/PerPageParam.yaml'
  - $ref: '../components/parameters/search/KeywordParam.yaml'
```

## 命名規則

|要素|規則|例|
|---|---|---|
|ディレクトリ|小文字・関心事別|`pagination/`, `search/`, `user/`|
|ファイル名|PascalCase + `Param`|`PageParam.yaml`, `UserIdParam.yaml`|

## ルール

- `in: path` パラメータは必ず `required: true` を設定する
- すべてのパラメータに `description` と `example` を記述し、ドキュメント品質を確保する
- クエリパラメータには適宜 `minimum`, `maximum`, `default` を設定する
- 再利用性を最大化するため、1ファイル = 1パラメータとする
