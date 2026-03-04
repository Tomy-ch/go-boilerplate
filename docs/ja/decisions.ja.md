# アーキテクチャ決定事項

このドキュメントでは、このBoilerplateで採用されている **技術選定の理由** を説明します。

ここでの目的は、これらの技術が常に最良であると主張することではなく、  
**なぜこのアーキテクチャにおいて採用されたのか** を明確にすることです。

これらの技術選定は、以下の設計目標に基づいて行われています。

## 設計目標

このBoilerplateは以下を優先しています。

- 保守性（Maintainability）
- 構造的安全性（Structural safety）
- 型安全性（Type safety）
- インフラの交換可能性（Replaceable infrastructure）
- 長期運用性（Long-term operability）

パフォーマンスや抽象化の最小化は、  
このテンプレートの **主要な目的ではありません**。

## なぜ Onion Architecture なのか

### Intent（Onion Architecture）

ビジネスロジックをインフラやフレームワーク依存から分離するため。

### Decision（Onion Architecture）

このプロジェクトでは **Pragmatic Onion Architecture** を採用しています。

この構造では、依存関係の方向が以下のように強制されます。

```txt
controller → usecase → domain
                     ↑
              infrastructure
```

Domain レイヤは外部システムから独立した状態を保ちます。

### Benefits（Onion Architecture）

- 責務の明確な分離
- テストの容易性
- インフラの交換可能性
- 安定したドメインコア

### Alternatives Considered（Onion Architecture）

#### Layered MVC

シンプルですが、ドメインロジックとインフラロジックが混在しやすい構造です。

#### Clean Architecture

概念的には非常に近いですが、  
追加の抽象レイヤが導入されることが多い傾向があります。

本プロジェクトでは **より実用的な簡略版** を採用しています。

## なぜ OpenAPI-first なのか

### Intent（OpenAPI-first）

実装より前に API 契約を明確に定義するため。

### Decision（OpenAPI-first）

API仕様は **OpenAPI** を使用して定義し、  
`oapi-codegen` を使ってサーバコードを生成します。

### Benefits（OpenAPI-first）

- API契約の明確化
- 型安全なリクエスト/レスポンス構造
- フロントエンドとの整合性
- APIドキュメントの自動生成

### Alternatives Considered（OpenAPI-first）

#### Code-first API

コードから OpenAPI を生成する方法は、  
API契約が不明確になりやすい問題があります。

#### GraphQL-first

GraphQL は強力ですが、一般的なバックエンドサービスでは複雑性が高くなる場合があります。

## なぜ SQL-first なのか

### Intent（SQL-first）

SQL を ORM の裏側に隠すのではなく、**契約として明示的に扱うため**。

### Decision（SQL-first）

クエリは SQL で直接記述し、`sqlc` によって Go コードを生成します。

### Benefits（SQL-first）

- クエリの完全な制御
- パフォーマンス特性の明確化
- 明示的なデータアクセスパターン

### Alternatives Considered（SQL-first）

#### Full ORM

ORM は便利ですが、  
クエリの挙動やパフォーマンスが見えにくくなる場合があります。

#### Query Builder

SQLの可視性が下がり、追加の抽象化によって複雑性が増す場合があります。

## なぜ sqlc なのか

### Intent（sqlc）

明示的な SQL と **型安全な Go コード** を組み合わせるため。

### Decision（sqlc）

`sqlc` を使用して SQL クエリから Go コードを生成します。

### Benefits（sqlc）

- コンパイル時の型安全性
- 明確な SQL 定義
- ランタイム抽象化の最小化

### Alternatives Considered（sqlc）

#### GORM

便利な ORM ですが、  
ORM抽象化と暗黙のクエリ生成が発生します。

#### Ent

スキーマファーストのアプローチであり、異なる開発フローが必要になります。

## なぜ Echo なのか

### Intent（Echo）

軽量で予測可能な HTTP フレームワークを提供するため。

### Decision（Echo）

HTTP ルーティングとミドルウェアに **Echo** を使用します。

### Benefits（Echo）

- シンプルで明確なミドルウェア構造
- 抽象化が少ない
- 良好なパフォーマンス

### Alternatives Considered（Echo）

#### Gin

非常に似たフレームワークですが、Echo の方がミドルウェア構成がややシンプルです。

#### Chi

優れたルーターですが、Echo はよりフレームワークとしての機能が揃っています。

## なぜ Fx なのか

### Intent（Fx）

構造化された依存関係解決と  
アプリケーションライフサイクル管理を提供するため。

### Decision（Fx）

依存性注入コンテナとして **Uber Fx** を採用しています。

### Benefits（Fx）

- 明示的な依存関係の配線
- アプリケーションライフサイクル管理
- モジュール構成の整理

### Alternatives Considered（Fx）

#### 手動DI

小規模システムでは有効ですが、システムが大きくなると管理が難しくなります。

#### Google Wire

コンパイル時DIですが、ランタイムのライフサイクル管理は提供されません。

## 将来の進化

これらの技術選定は **不変ではありません**。

以下のような場合には変更される可能性があります。

- エコシステムの進化
- より良いツールの登場
- アーキテクチャ制約の変化

ただし変更を行う場合でも、  
**このテンプレートの設計目標** は維持されるべきです。
