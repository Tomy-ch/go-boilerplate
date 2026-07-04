---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, cqrs]
---

# ADR-0026: CQRSの分割の外に位置する第4のDMLカテゴリとしてsystem_queryを導入する

English canonical: [0026-system-query-dml-category.md](../../adr/0026-system-query-dml-category.md)

## ステータス

accepted

## 背景

DMLディレクトリはSQLソースファイルをアーキテクチャ上の役割ごとに整理する。

| カテゴリ | 目的 |
| --- | --- |
| `repository/` | 集約CRUD（ドメインレイヤーインターフェース） |
| `query_service/` | ユースケース固有の読み込みクエリ（ユースケースレイヤーインターフェース） |
| `command_service/` | 書き込み側コマンド（ユースケースレイヤー、予約済み） |

これらの3つの役割のいずれにも属さないクエリが存在する。ヘルスチェック・べき等キーの検索・outbox行のポーリングはインフラレベルの操作であり、以下の性質を持つ。

- ユーザー向けのユースケースによって駆動されない
- ドメイン集約に対応しない
- ビジネスフィーチャーに関わらずすべてのデプロイに存在しなければならない

これらのクエリを`repository/`や`query_service/`に強制的に入れることは誤解を招く — 集約のオーナーもユースケースインターフェースも持たない。専用のカテゴリがその区別を明示的にする。

## 決定

`database/dml/system_query/`はシステム運用クエリ（ヘルス検証・べき等性強制・outbox配信）のための**第4のDMLカテゴリ**である。その実装は`internal/infrastructure/rdb/system_query/`に置かれ、`persistenceModule`内の専用`system_query`サブモジュール（`internal/di/module/persistence.go`）に登録される。

`system_query`カテゴリは[ADR-0025](0025-lightweight-cqrs.ja.md)で説明したCQRSの読み込み/書き込み分割の明示的な外側に位置する。アプリケーションのビジネスロジックではなくインフラの関心事に対応する。

`make gen-query`は4つのカテゴリすべてを同じmerge-dmlとsqlcパイプライン（[ADR-0021](0021-merged-dml-schema-as-sqlc-input.ja.md)参照）で処理するため、system_queryは他のカテゴリと同じ型安全なコード生成に参加する。

## 影響

### ポジティブな影響

- システム運用クエリがアプリケーションDMLから分離され、ドメイン永続化とヘルスチェックやべき等性書き込みが混在するリスクがない。
- インフラ実装（ヘルスチェック・べき等性・outbox）がRepositoryやQueryServiceモジュールを汚染することなく専用のDIサブモジュールに明確に登録される。
- 4カテゴリモデルが`persistenceModule`内の4つのサブモジュールに直接対応し、DMLとDIの両方で一貫した構造を提供する。

### ネガティブな影響

- 新しいクエリを配置する際に開発者が4カテゴリモデルを把握しなければならず、2カテゴリまたはフラットモデルと比較して認知オーバーヘッドが増える。
- `system_query`という名称は読み込み専用に聞こえるが、べき等性書き込みとoutboxの挿入を含む。「インフラ運用」という言葉の方が正確だが長い。

## 検討した代替案

### システムクエリをrepository/にマージする

シンプルな構造だが、ドメイン集約の関心事とインフラ運用の関心事を混在させる。ヘルスチェックとべき等キーは集約を持たない。それらを`repository/`に置くことは誤解を招きRepositoryの概念の一貫性を損なう。

### 単一のinfrastructure/カテゴリ

より汎用的 — CQRS以外のすべてのクエリを1か所に置く。4カテゴリモデルが提供するロールごとの明確さが失われ、どのSQLがドメイン関連かインフラ関連かが不明瞭になる。

### 専用のDMLカテゴリなし（GoファイルにインラインSQL）

追加のディレクトリを排除するがシステムクエリのsqlc型安全性を放棄する。sqlcが提供するコンパイル時の保証は追加の構造に見合う価値がある。

## 補足

- Source: [`database/dml/README.md`](../../../database/dml/README.md)の§ "Directory Structure"および§ "Subdirectory Mapping to Onion Architecture"。
- Source: [`database/dml/system_query/README.md`](../../../database/dml/system_query/README.md)。
- DI登録: [`internal/di/module/persistence.go`](../../../internal/di/module/persistence.go)。
- 関連: [ADR-0025](0025-lightweight-cqrs.ja.md)（system_queryが外側に位置するCQRS分割）；[ADR-0021](0021-merged-dml-schema-as-sqlc-input.ja.md)（merge-dmlパイプライン）。
