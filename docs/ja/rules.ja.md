# アーキテクチャルール

このドキュメントは、このプロジェクトにおける **破ってはいけないアーキテクチャルール** を定義します。

これらのルールは **人間の開発者** と **AIエージェント** の両方が必ず遵守する必要があります。

これらのルールに違反すると、システムのアーキテクチャ整合性が損なわれる可能性があります。

## レイヤ依存ルール

依存関係は常に **内側のレイヤへ向かう** 必要があります。

### 許可される依存

```mermaid
flowchart LR
    Controller --> Usecase --> Domain
    Infrastructure --> Domain
```

### 禁止される依存

```mermaid
flowchart LR
    Domain -.-> Infra["infrastructure"]
    Domain -.-> Controller
    Usecase -.-> Controller
```

**domain レイヤは常に最も独立したレイヤである必要があります。**

### 理由

このルールにより、ドメインモデルがフレームワークやインフラストラクチャに依存することを防ぎます。

## Usecase 依存ルール

Usecase は Infrastructure に直接依存してはなりません。

- 依存は必ず Boundary（interface）を通す
- Infrastructure 実装は DI によって注入される

```txt
Usecase → Boundary(interface) → Infrastructure
```

## 生成コードルール

一部のファイルは **自動生成されるコード** です。

これらのファイルは **手動で編集してはいけません**。

### 生成コードの例

例：

- OpenAPI から生成されたサーバコード
- sqlc によって生成されたクエリバインディング
- mock 生成ファイル

### ルール

生成コードは常に **ソース定義から完全に再生成できる状態**である必要があります。

生成コードを変更する必要がある場合は、  
**生成元の定義を変更してください。**

例：

|生成コード|ソース|
|---|---|
|OpenAPI server code|OpenAPI specification|
|SQL bindings|SQL query files|
|Mocks|Interface definitions|

## OpenAPI-first

API の変更は必ず **OpenAPI 定義から開始**します。

```mermaid
flowchart TB
    OpenAPI --> Gen["oapi-codegen"] --> IF["Server Interface"] --> Handler["Handler Implementation"]
```

### OpenAPI-first ルール

- API 契約を定義する前に handler を実装してはいけません
- 生成された API インターフェースを手動で編集してはいけません

OpenAPI 定義は **APIの唯一のソース（Single Source of Truth）** です。

## データベースマイグレーション

データベーススキーマの変更は、厳格なマイグレーションルールに従う必要があります。

### マイグレーションルール

- 既存の migration ファイルを **変更してはいけません**
- migration は **append-only（追記のみ）** です
- スキーマ変更は必ず **migration から開始**します

### 典型的なフロー

```mermaid
flowchart TB
    Migration --> Schema["Schema change"] --> SQL["SQL query update"] --> Gen["sqlc regeneration"]
```

これにより、データベースの履歴を常に再現可能に保つことができます。

## Domain レイヤ制約

Domain レイヤは **純粋かつ独立した状態** を保つ必要があります。

Domain レイヤでは以下の処理を **行ってはいけません**。

### Domain で禁止されること

- 外部 I/O
- データベースアクセス
- 環境変数の取得
- フレームワーク依存
- ログ出力
- HTTP ロジック

### Domain で許可されるもの

- Entity
- Value Object
- Domain Service
- ビジネスルール
- Repository インターフェース

## Context 伝搬ルール

- context.Context は必ず下位レイヤへ伝搬する
- 新規に context を生成してはいけない（例: context.Background）

## Infrastructure 実装ルール

Infrastructure コンポーネントは  
**Domain のインターフェースを実装する役割**を持ちます。

ルール：

- Infrastructure にドメインロジックを書いてはいけません
- Infrastructure は domain インターフェースに依存します
- Infrastructure は外部システムにアクセスできます

例：

- database adapter
- 外部 API クライアント
- repository 実装

## Repository / QueryService ルール

- Repository は Aggregate 永続化のみを扱う
- 検索・一覧取得は QueryService に実装する

禁止：

- Repository に検索ロジックを書くこと
- QueryService にドメインロジックを書くこと

## DTO / 型境界ルール

- OpenAPI の型を Usecase に渡してはいけない
- Controller で DTO に変換すること
- Domain は OpenAPI 型を知らない

### 境界の型変換

- フレームワーク/生成型（例: OpenAPI `openapi_types.UUID`）→ ドメイン型の変換は、**その型を所有する層にのみ**置く。HTTP では Controller の専用ヘルパー `internal/controller/conv` 経由とする。他層は生成/フレームワーク型を import しないため、依存方向で変換用途が境界に構造的に限定される。
- 利便性のために共有パッケージ（`pkg/`）へ**検証バイパスの公開コンストラクタを足さない**。バイパス入口は利用方針が時間とともに形骸化し、コードベース全体で乱用される。既存の検証済みコンストラクタ（`New` / `Parse`）を再利用すること。

## Infrastructure 型漏洩禁止

- sqlc の生成型を Usecase / Domain に渡してはいけない
- 必ず Domain Entity または DTO に変換する

## レイヤ責務ルール

各レイヤには明確な責務があります。

### Controller

責務：

- HTTP トランスポート
- リクエストバリデーション
- エラー変換

Controller に **ビジネスロジックを書いてはいけません**。

### Usecase

責務：

- アプリケーションワークフロー
- トランザクション境界
- ドメインロジックの調整

Usecase は **直接 Infrastructure に依存することを避けるべき**です。

### トランザクションルール

- トランザクションは Usecase 層でのみ開始する
- Infrastructure / Repository はトランザクションを開始してはいけない

## エラーハンドリングルール

- エラーを黙って握り潰さない。各エラーは「処理する」「`apperror` / `xerrors` でラップして伝播する」「**論理的に到達不能**で発生＝前提崩壊を意味するなら `panic` でラウドに通知する」のいずれかでなければならない。
- 「起き得ない失敗」は構造的に起こせなくするのを優先する。境界で値の有効性が既に保証される場合（例: echo で検証済みの path パラメータ）、防御的な `error` 戻りをスタックに引き回さず、**到達不能エラーで `panic` するヘルパー**経由で変換する。そのヘルパーは `Must` 系の明示的な命名にし、panic 経路を単体テストする。
- 理由: 到達不能な `if err != nil { return err }` はデッドコードであり、テスト不能・カバレッジ低下・意図の隠蔽を招く。`panic` は不変条件を文書化し、前提が破られたら確実に気づける。

## テストと完了の定義（Definition of Done）

- テストは**各層の実装と同居**させ、層ごとに検証する — その層のテストを書き、次へ進む前に `make test`（カバレッジ付き）を回す。テストを最終ステップにまとめないこと。先送りはカバレッジ不足を終盤まで隠す。
- 変更が「完了」とは、コンパイルが通ることではなく、**テスト済みでカバレッジ基準を満たす**こと（新規/変更パッケージ > 90%、handler はおおむね 100%）。`go build` の成功は完了の signal では**ない**。
- 「コンパイル可 ≠ 完了」は配線にも適用する: DI グラフの正しさは **runtime**（アプリが起動し `[Fx] RUNNING` に到達する）で検証する。ビルドや unit テストだけでは不十分。
- 意図的に到達不能な防御分岐（あり得ない `switch` の default、失敗し得ない前提を守る `panic`、強制困難なインフラのエラー経路）は未カバーで許容する — テストを歪めて到達させるのではなく、アサーションとして残す。
- テスト環境は `make db-init` 実行済みを前提とする: local/test 両 DB を migrate **かつ seed** する。`make serve` の後にまず `make db-init` を走らせること。seed を伴わない `db-*-migrate-up` 単体では DB 依存テストが落ちる。
- mock した unit / integration テストに加え、`make serve` の実アプリに `curl` で新エンドポイントを smoke 検証する: mock は実 DI グラフ・実 DB・end-to-end 配線を通らない。構造化 observability ログ（`docker compose logs api_server` — 層ごとの span + logging DB driver が出力する実 SQL）を durable な検証エビデンスとして扱い、再確認は再実行せずログを読む。
- 破壊的なランタイム検証（dev DB に対する `POST` / `PUT` / `PATCH` / `DELETE`）は、非破壊な代替が無い場合は事前にユーザー確認する。検証後は `make db-init` で復元する。

## AIエージェントルール

AI が生成するコードも、すべてのアーキテクチャルールに従う必要があります。

AI エージェントは以下を守る必要があります。

- レイヤ境界を守る
- OpenAPI-first 開発を遵守する
- SQL ファイルを契約として扱う
- 生成コードを編集しない

コード生成を行う前に、AI エージェントは以下のドキュメントを参照してください。

- `architecture.ja.md`
- `development-flow.ja.md`

## Summary

これらのルールは以下を実現するために存在します。

- アーキテクチャ整合性の維持
- 保守しやすいコード構造
- 再現可能なビルド
- 人間とAIの安全な協働
