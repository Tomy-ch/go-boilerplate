# インフラ層のRDB（`internal/infrastructure/rdb`）ガイド

[English](README.md) | 日本語

## 役割

`internal/infrastructure/rdb` は **RDB（PostgreSQL）を利用するための Infrastructure サブシステム**です。

このディレクトリは次の責務を持ちます。

- PostgreSQL への接続管理
- SQL 実行（sqlc）
- Repository / QueryService 実装
- SQL 実行ログとトレース（Observability）
- PostgreSQL エラーの正規化
- DB nullable 型と Go 型の変換

Domain / Usecase は **RDB の実装詳細を意識せずにデータ永続化・検索を利用できます。**

## RDB アーキテクチャ

このディレクトリは次のレイヤー構造で構成されています。

```txt
Usecase
   ↓
Repository / QueryService
   ↓
loggingdb
   ↓
driver
   ↓
PostgreSQL
```

各レイヤーの責務は次の通りです。

|Layer|役割|
|---|---|
|Repository|Aggregate 永続化（Domain Repository Interface の実装）|
|QueryService|検索専用クエリーの提供（Usecase Interface の実装）|
|loggingdb|SQL ログ / トレース付与（Observability wrapper）|
|driver|DB 接続管理 / トランザクション管理|
|PostgreSQL|実際の DB|

補助コンポーネントとして以下が存在します。

|Component|役割|
|---|---|
|sqlc|SQL から生成された型安全なクエリ実行コード|
|conv|nullable 型と Go 型の変換|
|pgerror|PostgreSQL エラー → アプリケーションエラー変換|
|testkit|RDB テストユーティリティ（実DB + rollback）|

## ディレクトリ構成

```txt
internal/infrastructure/rdb

 ├ repository/        Repository 実装
 ├ query_service/     QueryService 実装
 ├ driver/            DB 接続 / トランザクション
 │   └ loggingdb/     SQL logging / tracing wrapper
 ├ sqlc/              sqlc 生成コード + SQL helper
 ├ conv/              nullable 型変換
 ├ postgres/
 │   └ pgerror/       PostgreSQL エラー正規化
 └ testkit/           RDB テストユーティリティ
```

## Repository

Repository は **Domain の Repository Interface を実装する層**です。

主な責務

- sqlc クエリ実行
- Row → Domain エンティティ変換
- DB エラー正規化

重要:

```txt
Repository はビジネスロジックを持たない
```

詳細は以下を参照してください。

[repository ディレクトリの README](repository/README.ja.md)

## QueryService

QueryService は **検索・一覧取得などの読み取り専用クエリーを提供する層**です。

Repository が Aggregate 永続化を扱うのに対し、
QueryService は検索用途に特化します。

主な責務

- 検索クエリ実行
- Row → Domain / DTO 変換
- DB エラー正規化

重要:

```txt
検索は Repository ではなく QueryService に実装する
```

詳細は以下を参照してください。

[query_service ディレクトリの README](query_service/README.ja.md)

## sqlc

`sqlc` は SQL から Go コードを生成するツールです。

このディレクトリでは

- sqlc 生成コード
- LIKE 検索 helper
- Enum / 状態変換 helper

などを提供します。

```txt
internal/infrastructure/rdb/sqlc/gen
```

に生成コードが配置されます。

詳細は以下を参照してください。

[sqlc ディレクトリの README](sqlc/README.md)

## conv

`conv` は **nullable 型と Go ポインタ型の変換ユーティリティ**です。

```txt
sql.NullString ⇔ *string
sql.NullTime   ⇔ *time.Time
```

Repository / QueryService 実装で利用されます。

詳細は以下を参照してください。

[conv ディレクトリの README](conv/README.md)

## driver

`driver` は **DB 接続とトランザクション管理を提供する最下位レイヤー**です。

主な機能

- DatabaseDriver 抽象
- DB 接続プール管理
- トランザクション管理（tx.Manager）
- sqlc 用 DBTX インターフェース

重要:

```txt
トランザクション境界は Usecase 層が管理する
```

詳細は以下を参照してください。

[driver ディレクトリの README](driver/README.md)

## loggingdb

`loggingdb` は **SQL 実行ログとトレースを付与する Observability wrapper** です。

```txt
Repository / QueryService
   ↓
loggingdb
   ↓
driver
```

主な機能

- SQL ログ出力
- OpenTelemetry span
- クエリ実行時間計測
- slow query 判定

重要:

```txt
loggingdb は DB 実行を行わない（pure wrapper）
```

詳細は以下を参照してください。

[loggingdb ディレクトリの README](driver/loggingdb/README.md)

## PostgreSQL エラー正規化

`postgres/pgerror` は **PostgreSQL 固有エラーをアプリケーションエラーへ変換するレイヤー**です。

Repository / QueryService は

```go
pgerror.NormalizeError(err)
```

を利用して DB エラーを正規化します。

主な変換

```txt
sql.ErrNoRows      → ErrNotFound
unique violation   → ErrConflict
connection error   → ErrUnavailable
others             → ErrInternal
```

詳細は以下を参照してください。

[pgerror ディレクトリの README](postgres/pgerror/README.md)

## testkit

`testkit` は **RDB を利用するテストのためのユーティリティ**です。

主な機能

- テスト用 DB 初期化
- LoggingDBProvider 生成
- トランザクション内テスト（自動 rollback）

テスト特性

```txt
実DB
+
トランザクション rollback
+
並列実行（Tx は直列）
```

詳細は以下を参照してください。

[testkit ディレクトリの README](testkit/README.md)

## 設計方針

この RDB サブシステムは次の設計原則に基づいています。

### 1. DB 実装の隠蔽

Domain / Usecase は

```txt
sql
pgx
sql.DB
```

などの DB 実装に依存しません。

### 2. 責務分離（Repository / QueryService）

```txt
書き込み / 永続化 → Repository
検索 / 読み取り   → QueryService
```

### 3. トランザクション境界の集中管理

```txt
Usecase が Tx を管理
Infra は Tx を開始しない
```

### 4. DB エラーの正規化

PostgreSQL 固有エラーは

```txt
pgerror
```

でアプリケーション共通エラーへ変換します。

### 5. SQL 型安全

SQL 実行はすべて

```txt
sqlc
```

を通して行います。

### 6. 可観測性

SQL 実行ログとトレースは

```txt
loggingdb
```

が提供します。

### 7. テスト戦略（Integration 前提）

Repository / QueryService テストは

```txt
実DB + rollback
```

で実行します。

```txt
testkit
```

を利用して安全に実現します。
