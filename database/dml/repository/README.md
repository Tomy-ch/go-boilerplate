# RepositorySQLについて

## 概要

database/dml/repository/ 配下は、ドメイン層のエンティティ操作に必要な SQL を管理します。
sqlc により型安全な Go コードを自動生成し、リポジトリ層で利用します。

## 目的

- エンティティ永続化のための DML 集約
  INSERT / UPDATE / DELETE / SELECT など、ドメインモデルを永続化・取得するためのクエリを集約します。
- 責務の明確化
- Domain: ビジネスロジック・整合性維持
- このディレクトリ: DB とのやり取りのみ
  → ビジネスロジックを含めないシンプルな SQL。
- 型安全なリポジトリ実装
  SQLC の生成コードにより、パラメータや戻り値の型をコンパイル時に保証します。

## ディレクトリ構成

```text
database/dml/repository/
  ├── user/
  │    ├── insert_user.sql
  │    └── select_user_by_id.sql
  ├── product/
  │    ├── insert_product.sql
  │    └── ...
  └── ...
```

## 運用ルール

- 命名規則
  動詞＋対象名＋条件（例: select_user_by_id.sql）。
- SQLの記述
- -- name: コメントで生成関数名を明示。
- 複雑な条件や結合は最小限に抑える（集計はQS側で行う）。
- 生成方法
  gen-query-repo コマンドで対象カテゴリの SQLC コードを生成。
