# インフラ層（`internal/infrastructure`）ガイド

[English](README.md) | 日本語

## 役割

Infrastructure 層は、**外部技術（DB・外部API・認証等）へのアクセス実装**を担う層です。

この層は以下の責務を持ちます。

- 外部I/Oの実装（RDB / API / 認証 / システム）
- Domain が定義した interface の実装
- 技術的詳細（接続・リトライ・ドライバ・ログ等）のカプセル化
- エラーの正規化
- Observability（ログ / トレース）の付与

上位層（Domain / Usecase）は、**Infrastructure の実装詳細を一切意識しません。**

## オニオンアーキテクチャでの位置

```mermaid
flowchart TB
    Infra["Infrastructure"]
    Usecase["Usecase"]
    Domain["Domain"]

    Infra --> Usecase --> Domain
```

- Domain / Usecase は抽象のみ
- Infrastructure は具体実装

## 依存関係

```mermaid
flowchart LR
    Infra["Infrastructure"]
    Domain["Domain（interface）"]
    Usecase["Usecase"]

    Infra --> Domain
    Infra --> Usecase
```

- Infrastructure は Domain に依存する
- Domain / Usecase は Infrastructure に依存してはならない

## Usecaseとの関係

- トランザクション境界は Usecase 層が管理
- Infrastructure はトランザクションを開始しない
- トランザクションは context.Context により伝搬される

```mermaid
flowchart TB
    UC["Usecase（Tx開始）"]
    Repo["Repository / QueryService"]
    Driver["driver（Tx使用）"]

    UC --> Repo --> Driver
```

## エラーハンドリング

Infrastructure 層は外部技術のエラーをそのまま返さず、  
**アプリケーション共通エラーへ変換**します。

例：

- PostgreSQL エラー → pgerror.NormalizeError
- 外部APIエラー → apperror に変換

## Observability

Infrastructure 層では以下の可観測性を提供します。

- SQL / 外部I/Oのログ出力
- OpenTelemetry によるトレース
- 実行時間計測（slow query）

主に driver の接続層に結線した pgx クエリトレーサー（`otelpgx` の span、ログ出力はクエリ失敗とスロークエリのみ）で実現します。

driver 層のトレーサーに加え、各 I/O コンポーネント（Repository / QueryService / SystemQuery / 外部 gateway / queue / publisher）は public メソッドごとにアプリケーション層の span を発行します。具体的には `observability.LayerTracer` フィールドをコンストラクタで `tf.Infra()` から初期化し、各メソッド先頭で `ctx, endSpan := r.tracer.Start(ctx); defer endSpan()` を書きます。実 I/O を持たない純粋なメモリ内コンポーネントは対象外です。

## 禁止事項

Infrastructure 層では以下を行ってはいけません。

- ビジネスロジックの実装
- Domain ルールの分岐
- Usecase の意思決定
- HTTP / Framework 依存コードの持ち込み
- トランザクションの開始

## 実装ルール

- SQL 実行は sqlc を使用する
- Repository に検索ロジックを書かない（QueryServiceへ）
- DBTX は `driver.New(ctx, db)` で取得する（ログ / トレースは driver の接続層で付与される）
- context を必ず伝搬する
- 外部エラーは必ず正規化する

## ディレクトリ構成

```mermaid
flowchart TB
    Root["internal/infrastructure"]
    Auth["auth/"]
    Authz["authz/"]
    HTTP["httpclient/"]
    Pub["publisher/"]
    Queue["queue/"]
    RDB["rdb/"]
    Sys["system/"]
    Web["webapi/"]

    Root --> Auth
    Root --> Authz
    Root --> HTTP
    Root --> Pub
    Root --> Queue
    Root --> RDB
    Root --> Sys
    Root --> Web
```

## サブディレクトリ

|ディレクトリ|説明|interface 配置|詳細|
|---|---|---|---|
|`auth/`|認証基盤（環境別 Authenticator 実装）|Usecase boundary|[README](auth/README.ja.md)|
|`authz/`|認可基盤（Authorizer 実装。本番以外はデフォルトの `allowall`）|Usecase boundary|[README](authz/README.ja.md)|
|`httpclient/`|resilient な HTTP client substrate（retry / circuit breaker / tracing）。`webapi/` と `publisher/` が共用する driver 相当の基盤|—（substrate、domain/usecase IF なし）|—|
|`publisher/`|transactional outbox の publish 先（`boundary.Publisher` の HTTP 実装）|Usecase boundary|—|
|`queue/`|メッセージキューの worker seam 実装（AWS SQS による `worker.Consumer` / `FailureHandler` 実装）|Usecase boundary（worker seam）|[README](queue/sqs/README.ja.md)|
|`rdb/`|RDB サブシステム（Repository / QueryService / driver / sqlc 等）|Domain / Usecase|[README](rdb/README.ja.md)|
|`system/`|システム依存処理（時刻取得等）|Usecase boundary|[README](system/README.ja.md)|
|`webapi/`|外部 Web API gateway（為替レート等、`boundary.Gateway` の実装）|Usecase boundary|—|

## テスト戦略

- 実DBを用いた Integration Test
- トランザクション rollback による状態隔離
- testkit を利用

```mermaid
flowchart TB
    DB["実DB"]
    Rollback["rollback"]
    Parallel["並列実行"]
    Serial["Txは直列化"]

    DB --> Rollback --> Parallel --> Serial
```

## 設計原則まとめ

### 1. 技術詳細のカプセル化

DB / API / 認証  
→ Infrastructure に閉じ込める

### 2. 依存関係の逆転

Domain が interface を定義  
Infrastructure が実装する

### 3. 責務分離

永続化 → Repository  
検索   → QueryService

### 4. トランザクション管理

Usecase が管理  
Infrastructure は関与しない

### 5. エラー統一

外部エラー → アプリケーションエラー

### 6. 可観測性

logging / tracing / metrics
