# OpenAPI パラメータ定義ガイド

このドキュメントでは、OpenAPI における `$ref` の書き方、および `parameters` セクションのベストプラクティスを整理しています。

## ファイル構造は以下を推奨

### 共通系パラメータ（全エンドポイントで使い回すもの）

- 例: `page`, `per_page`, `offset`, `limit` など
- ファイル: `components/parameters/common.yaml`

### ドメイン別パラメータ（user_id など）

- 例: `user_id`, `project_id`
- ファイル: `components/parameters/user.yaml`, `project.yaml` などに分離して管理

## `$ref` の例

```yaml
parameters:
  - $ref: './../../../components/parameters/pagination.yaml'
  - $ref: './../../../components/parameters/pagination.yaml'
  - $ref: './../../../components/parameters/user.yaml'
```

## 命名規則まとめ

|用途|命名規則|例|
|------------|----------------|---------------------------|
|パス名|スラッグ形式|`/v1/users/{user_id}`|
|ディレクトリ|snake_case|`components/parameters/`|
|ファイル名|PascalCase|`UserIdParam.yaml`|
|定義名|PascalCase|`UserIdParam`|
