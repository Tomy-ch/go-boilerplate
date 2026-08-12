# OpenAPI パラメータ クイックリファレンス

[English](parameter-guide.md) | 日本語

## $ref の例

```yaml
# openapi/paths/v1/users.yaml から参照する場合
parameters:
  - $ref: '../../components/parameters/pagination/PageParam.yaml'
  - $ref: '../../components/parameters/pagination/PerPageParam.yaml'
  - $ref: '../../components/parameters/search/KeywordParam.yaml'
```

`../` の数は参照元ファイルの深さに従います。`paths/v1/*.yaml` なら 2 段、
`paths/v1/<group>/*.yaml` なら 3 段、といった具合です。

## 命名規則

|要素|規則|例|
|---|---|---|
|パス名|スラッグ形式|`/v1/users/{userId}`|
|ディレクトリ|小文字・関心事別|`pagination/`, `search/`, `user/`|
|ファイル名|PascalCase + Param|`PageParam.yaml`, `UserIdParam.yaml`|

## 詳細ガイド

設計ポリシーと利用可能なパラメータの全容は [components/parameters/README.ja.md](components/parameters/README.ja.md) を参照してください。
