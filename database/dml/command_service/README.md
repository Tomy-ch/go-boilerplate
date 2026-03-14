# CommandService について

## 概要

CommandService は、アプリケーションの **書き込み処理（状態変更）を集約する層**です。

`database/dml/command_service/<category>` 配下に配置された SQL ファイルをもとに、sqlc により型安全な Go コードを自動生成します。

CommandService はドメイン層の集約ルールと不変条件を尊重しながら、データベースへの **INSERT / UPDATE / DELETE** を実行します。  
読み取り最適化を目的とする QueryService と異なり、**状態変更の整合性とトランザクション管理を重視する層**です。

## 役割と非役割

役割：状態変更を伴うユースケース。INSERT・UPDATE・DELETE 等の DML を実行し、トランザクション整合性を担保する。

非役割：参照最適化のための JOIN / 集計 / 非正規化 / キャッシュ / 検索。これらは QueryService の責務。

## 目的

- 状態変更の明確化  
書き込み処理を CommandService に集約することで、状態変更の責務を明確にします。

- トランザクション整合性の維持  
複数テーブルの更新や整合性が必要な処理をトランザクション単位で管理します。

- 型安全なアクセス  
SQLC の生成物により、プレースホルダやスキャンの型ミスをコンパイル時に防止します。

## ディレクトリ構成

```txt
database/dml/command_service/
  ├── user/
  │    ├── insert_user.sql
  │    ├── update_user_email.sql
  │    └── ...
  ├── product/
  │    ├── publish_product.sql
  │    └── ...
  └── ...
```

## 運用ルール

- ファイル名
  - クエリの意図が分かる **動詞＋対象名** で命名します。
  - 意味で命名：
    - OK: CreateUser
    - OK: UpdateUserEmail
    - NG：InsertIntoUsers（Infra先行の命名）

- SQLの記述
  - `-- name:` コメントで関数名を明示します。
  - 必要に応じてパラメータや返却カラムの型をコメントに記載します。

- 生成
  - `gen-query-cmd` コマンドで対象カテゴリの SQLC コードを生成します。

- 整合性
  - CommandService は **トランザクション境界内で実行**します。
  - 更新対象が複数テーブルにまたがる場合は、必ず整合性を確認します。

- パフォーマンス
  - 不要な SELECT を伴う更新は禁止します。
  - UPDATE / DELETE は必ずインデックスを考慮した条件で実行します。
  - 大量更新の場合はバッチ処理を検討します。

- 安全性
  - UPDATE / DELETE は **WHERE 条件必須**とします。
  - フルテーブル更新が必要な場合は ADR に理由を記載します。
