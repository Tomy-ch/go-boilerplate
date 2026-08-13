# OpenAPI パラメータ

[English](README.md) | 日本語

`openapi/components/parameters/` は、**再利用可能な OpenAPI パラメータ定義**を関心事ごとに整理して格納するディレクトリです。

## ディレクトリ構成

パラメータ群ごとに 1 つのディレクトリを置く。エンドポイントをまたいで共有するものは関心事の名前
（`pagination/` / `search/`）を、単一リソースが所有するものはそのリソース名を付ける。

## ページネーション戦略

本プロジェクトは**2つのページネーション方式**を提供し、それぞれが各方式で慣用的なパラメータ名を**あえて統一せず**保持しています：

- **オフセット**（`page` / `perPage`）— REST 流のページ移動。件数が有限でページ指定できる一覧向け。
- **カーソル / keyset**（`first` / `after`）— Relay 流の前方走査。offset が高コスト・不安定になる大規模・無限フィード向け。

カーソル側を `perPage` 等にリネームするのは非慣用的（カーソル走査に「ページ」概念はない）かつ API 破壊変更になるため、この分離は意図的です。

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
