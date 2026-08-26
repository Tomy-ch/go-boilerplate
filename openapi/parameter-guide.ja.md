# OpenAPI パラメータ クイックリファレンス

[English](parameter-guide.md) | 日本語

## $ref の例

<!-- sample-api:replace-begin -->
```yaml
# openapi/paths/v1/users.yaml から参照する場合
parameters:
  - $ref: '../../components/parameters/pagination/PageParam.yaml'
  - $ref: '../../components/parameters/pagination/PerPageParam.yaml'
  - $ref: '../../components/parameters/search/KeywordParam.yaml'
```
<!-- sample-api:replace-with -->
<!-- = ```yaml -->
<!-- = # openapi/paths/v1/<resource>.yaml から参照する場合 -->
<!-- = parameters: -->
<!-- =   - $ref: '../../components/parameters/pagination/PageParam.yaml' -->
<!-- =   - $ref: '../../components/parameters/pagination/PerPageParam.yaml' -->
<!-- = ``` -->
<!-- sample-api:replace-end -->

`../` の数は参照元ファイルの深さに従います。`paths/v1/*.yaml` なら 2 段、
`paths/v1/<group>/*.yaml` なら 3 段、といった具合です。

## 命名規則

<!-- sample-api:replace-begin -->
|要素|規則|例|
|---|---|---|
|パス名|スラッグ形式|`/v1/users/{userId}`|
|ディレクトリ|小文字・関心事別|`pagination/`, `search/`, `user/`|
|ファイル名|PascalCase + Param|`PageParam.yaml`, `UserIdParam.yaml`|
<!-- sample-api:replace-with -->
<!-- = |要素|規則|例| -->
<!-- = |---|---|---| -->
<!-- = |パス名|スラッグ形式|`/v1/<resource>/{<resource>Id}`| -->
<!-- = |ディレクトリ|小文字・関心事別|`pagination/`, `idempotency/`| -->
<!-- = |ファイル名|PascalCase + Param|`PageParam.yaml`, `IdempotencyKeyParam.yaml`| -->
<!-- sample-api:replace-end -->

## 詳細ガイド

設計ポリシーと利用可能なパラメータの全容は [components/parameters/README.ja.md](components/parameters/README.ja.md) を参照してください。
