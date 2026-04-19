# OpenAPI パラメータ クイックリファレンス

[English](parameter-guide.md) | 日本語

## $ref の例

```yaml
parameters:
  - $ref: '../components/parameters/pagination/PageParam.yaml'
  - $ref: '../components/parameters/pagination/PerPageParam.yaml'
  - $ref: '../components/parameters/search/KeywordParam.yaml'
  - $ref: '../components/parameters/user/UserIdParam.yaml'
```

## 命名規則

|要素|規則|例|
|---|---|---|
|パス名|スラッグ形式|`/v1/users/{user_id}`|
|ディレクトリ|小文字・関心事別|`pagination/`, `search/`, `user/`|
|ファイル名|PascalCase + Param|`PageParam.yaml`, `UserIdParam.yaml`|

## 詳細ガイド

設計ポリシーと利用可能なパラメータの全容は [components/parameters/README.ja.md](components/parameters/README.ja.md) を参照してください。
