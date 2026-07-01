# OpenAPI パラメータ

[English](README.md) | 日本語

`openapi/components/parameters/` は、**再利用可能な OpenAPI パラメータ定義**を関心事ごとに整理して格納するディレクトリです。

## ディレクトリ構成

```text
parameters/
├── pagination/         # ページネーションパラメータ（エンドポイント共通）
│   ├── PageParam.yaml        # offset 方式
│   ├── PerPageParam.yaml     # offset 方式
│   ├── CursorFirstParam.yaml # cursor 方式
│   └── CursorAfterParam.yaml # cursor 方式
├── search/             # 検索パラメータ（エンドポイント共通）
│   ├── KeywordParam.yaml
│   └── ActiveParam.yaml
└── user/               # ユーザー固有パラメータ（サンプル）
    └── UserIdParam.yaml
```

> `user/` は**サンプル実装**です。サービス構築時には参考にした上で、必要に応じて置き換え・削除してください。

## 利用可能なパラメータ

### pagination

本プロジェクトは**2つのページネーション方式**を提供し、それぞれが各方式で慣用的なパラメータ名を**あえて統一せず**保持しています：

- **オフセット**（`page` / `per_page`）— REST 流のページ移動。件数が有限でページ指定できる一覧向け。
- **カーソル / keyset**（`first` / `after`）— Relay 流の前方走査。offset が高コスト・不安定になる大規模・無限フィード向け。

カーソル側を `per_page` 等にリネームするのは非慣用的（カーソル走査に「ページ」概念はない）かつ API 破壊変更になるため、この分離は意図的です。

|ファイル|パラメータ|型|方式|説明|
|---|---|---|---|---|
|`PageParam.yaml`|`page` (query)|integer|offset|ページ番号（1始まり、デフォルト: 1）|
|`PerPageParam.yaml`|`per_page` (query)|integer|offset|1ページあたりの件数（デフォルト: 10、最大: 100）|
|`CursorFirstParam.yaml`|`first` (query)|integer|cursor|取得件数の上限（デフォルト: 50、最大: 200）|
|`CursorAfterParam.yaml`|`after` (query)|string|cursor|次ページ用の不透明カーソル。先頭ページは省略|

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
