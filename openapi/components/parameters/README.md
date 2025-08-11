# OpenAPI パラメータ設計ポリシー

このドキュメントは、OpenAPI の `parameters` セクションにおける設計方針を定義します。

クエリパラメータ・パスパラメータを再利用性と読みやすさを両立して定義することを目的としています。

## パラメータの種類と目的

| 用途           | 定義場所           | 例                     |
|----------------|--------------------|------------------------|
| クエリパラメータ | `in: query`        | `?page=1&per_page=10`  |
| パスパラメータ   | `in: path`         | `/users/{user_id}`     |

## ディレクトリと命名ポリシー

### ファイル構造の例

```text
components/
└── parameters/
    ├── pagination.yaml
    ├── user.yaml
```

### 命名規則

| 要素       | 規則            | 例                         |
|------------|------------------|----------------------------|
| ファイル名 | PascalCase       | `Pagination.yaml`, `User.yaml` |
| 定義名     | PascalCase       | `PageParam`, `UserIdParam`     |
| `$ref`       | パス + fragment | `#/components/parameters/UserIdParam`      |

## 統合パラメータファイルの使用

複数のパラメータをまとめて管理したい場合、1ファイルに統合する方法を推奨します。

### 例：PageParam.yaml

```yaml
name: page
in: query
description: ページ番号（1以上）
required: false
schema:
  type: integer
  minimum: 1
  example: 1
```

## 再利用の方法

YAML 内では `$ref` を使用して共通パラメータを再利用します：

```yaml
parameters:
  - $ref: '../../components/parameters/pagination.yaml'
  - $ref: '../../components/parameters/user.yaml'
```

## 注意点

- `in: path` のパラメータは `required: true` を必ず指定してください
- `in: query` のパラメータには必要に応じて `minimum`, `maximum`, `nullable` などを活用してください
- `description` と `example` は必ず記述して可読性・自動ドキュメント性を高めましょう

## 結論

- パラメータ定義は、**責務単位でファイル分割 or 統合しやすい構造で整理**
- `$ref` 参照は **常にフラグメント付きで一貫性を確保**
- 仕様とドキュメントがブレない形でパラメータ設計を行い、**長期的な保守性を高める**
