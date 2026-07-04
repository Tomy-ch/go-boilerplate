---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, cqrs, architecture]
---

# ADR-0025: 軽量CQRSの採用 — 書き込みにRepository、読み込みにQueryService

English canonical: [0025-lightweight-cqrs.md](../../adr/0025-lightweight-cqrs.md)

## ステータス

accepted

## 背景

純粋なオニオンアーキテクチャでは、すべての永続化はドメインレイヤーで定義されたRepositoryインターフェースを介して行われる。これは集約レベルの書き込みとシンプルな読み込み（IDによるフェッチ・自身の属性によるカウント）には適切である。RepositoryはドメインのInvariantを守り完全な集約を再構築するからである。

このモデルはユースケース固有の読み込み要件で崩壊する。

- **集約をまたぐジョイン**: ユーザー一覧APIはユーザーごとに都道府県名を返す必要がある。`users`と`prefectures`をジョインすることは、ジョインの詳細をドメインに漏洩させることなく単一集約のRepositoryメソッドとして表現できない。
- **ビュー形状のDTO**: APIレスポンスは集約データのサブセットや再成形（ページネーションメタデータ・ネストされたオブジェクト）を必要とすることが多い。完全な集約を返してユースケースレイヤーでマッピングするのは正しいが、大きな読み込みセットには非効率である。
- **複雑なフィルタリングと全文検索**: 複数カラムにわたるキーワード検索・GINインデックスクエリ・ページネーション結果は、ドメインインターフェースにきれいに収まらないクエリパターンを必要とする。

これらのクエリをRepositoryに押し込めると、ドメインインターフェースにビュー固有のメソッドが蓄積し集約のカプセル化が損なわれ、ドメインがプレゼンテーション関心事に依存するようになる。逆の極端 — 別個の読み込みデータベース・イベントプロジェクション・結果整合性を持つフルCQRS — は、現在のスケールでは正当化されない大きなインフラと運用上の複雑さをもたらす。プロジェクトには中間的なアプローチが必要である。

## 決定

同一のPostgreSQLインスタンス上で**軽量CQRS**を採用し、永続化を3つの責務に分割する。

### Repository（コマンド / 書き込みパス）

- インターフェースは**ドメインレイヤー**で定義する（`internal/domain/<aggregate>/repository.go`）。
- 集約の永続化とシンプルな単一集約の読み込みを担当する：IDによるフェッチ・集約自身の属性によるシンプルなフィルタ / リスト / カウント。
- 集約をまたぐジョイン・集計・キーワード検索は扱わない。
- 実装は`internal/infrastructure/rdb/repository/<aggregate>/`に置く。

### QueryService（クエリ / 読み込みパス）

- インターフェースは**ユースケースレイヤー**で定義する（`internal/usecase/<aggregate>/query/`）。読み込みモデルはドメインのInvariantではなくユースケースの関心事であるため、ドメインではなくユースケースに属する。
- 集約境界をまたぐ読み込み・複数テーブルのJOIN・ページネーション・全文検索・APIレスポンス形状のDTOを返す読み込みを扱う。
- ドメインエンティティではなくDTOを返す。
- 実装は`internal/infrastructure/rdb/query_service/<aggregate>/`に置く。

### Command Service（予約スロット）

- ドメインRepositoryに収まらない書き込み側の複雑さのために`persistenceModule`内に予約されたサブモジュール（例：バルク操作・非ドメインテーブルへの書き込み最適化コマンド）。現在は空。

RepositoryとQueryServiceはいずれも`internal/di/module/persistence.go`の`persistenceModule`に登録され、Uber Fx経由でインジェクトされる（[ADR-0030](0030-uber-fx-di.ja.md)参照）。これはフルCQRSではない：別個の読み込みストア・イベントソーシング・結果整合性のプロジェクションパイプラインは存在しない。

日々の境界適用ルールは[`docs/rules.md`](../../rules.md)の§ "Repository / QueryService Rules"参照。

## 影響

### ポジティブな影響

- Repositoryが集約に集中したまま保たれる。ドメインインターフェースにビュー固有のメソッドが蓄積せず、[ADR-0002](0002-onion-architecture.ja.md)に従ってドメインの純粋性が保たれる。
- QueryServiceはドメインロジックに触れることなく、また読み込みパスにドメインエンティティを露出させることなく、クエリ（ジョイン・ページネーション・全文検索・GINインデックス）を自由に最適化できる。
- ユースケースレイヤーがQueryServiceインターフェースを所有する：読み込みモデルはユースケースの関心事であるため、そのインターフェースはドメインではなくユースケースに属する。
- 両側がインターフェース背後に置かれDI経由でインジェクトされ、[ADR-0001](0001-avoid-lock-in.ja.md)に従って交換可能に保たれる。
- 新しいインフラ依存はなく、両パスが同一のPostgreSQLインスタンスで動作する。

### ネガティブな影響

- 2つの永続化抽象（RepositoryとQueryService）があるため、開発者は特定の読み込みに対してどちらを使うかを決める必要がある。境界は`docs/rules.md`に文書化されているが理解を要する。
- ユースケースレイヤーのQueryServiceインターフェースはドメインからより遠く、ドメインコードを単独で読む際に意図が分かりにくくなることがある。
- 「Repositoryに複雑な読み込みを置かない」境界はレビューで維持する必要があり、この区別にコンパイラによる強制はない。

## 検討した代替案

### ファットRepository（すべての読み込みをRepositoryに）

ジョイン・ページネーション・キーワード検索を含むすべての読み込みをドメインRepositoryインターフェースに置く。理解しやすくシンプルで1つの永続化抽象のみが必要になる。

ドメインインターフェースにビュー固有のクエリが蓄積し、ドメインをプレゼンテーション要件に結合し、時間とともに集約のカプセル化が損なわれるため却下。ドメインはAPIがどのようにレスポンスを整形するかを知るべきでない。

### 別個の読み込みストアを持つフルCQRS

専用の読み込みデータベース（例：Elasticsearch・イベントプロジェクションで更新されるマテリアライズドビューを持つ読み込みレプリカ）を維持する。強力な読み込みスケーラビリティとNLPグレードの検索を提供する。

現在のデータセットとクエリの複雑さは別個のストアを必要としない。結果整合性とプロジェクションの維持が、まだ正当化されない運用オーバーヘッドを追加するため時期尚早として却下。

### すべての読み込みをQueryService経由（Repositoryの読み込みを廃止）

Repositoryから読み込みメソッドを完全に排除し、すべての読み込みをQueryService経由にする。境界をシンプルにするが、些細な単一集約の参照（例：書き込み前提条件チェックのためのIDによるユーザーフェッチ）にQueryServiceのオーバーヘッドを強制する。Repositoryは集約ライフサイクルに不可欠な読み込みの自然な場所であるため却下。

### ユースケースレベルのみのCQRS（QueryService抽象なし）

ユースケースが複雑な読み込みのためにRepositoryメソッドを直接呼び出しインメモリジョインを適用する。新しい抽象を避けられるが、N+1クエリとパフォーマンスの懸念をアプリケーション層に移す。パフォーマンスと正確性の理由で却下。

## 補足

- Source: [`internal/infrastructure/rdb/query_service/README.md`](../../../internal/infrastructure/rdb/query_service/README.md)の§ "Relationship to CQRS"および§ "When to Use QS Over Repository"。
- Source: [`docs/rules.md`](../../rules.md)の§ "Repository / QueryService Rules"。
- DI登録: [`internal/di/module/persistence.go`](../../../internal/di/module/persistence.go)。
- 関連: [ADR-0026](0026-system-query-dml-category.ja.md)（CQRSの外に位置する第4カテゴリとしてのsystem_query）；[ADR-0028](0028-in-database-full-text-search.ja.md)（FTSに使用されるQueryService）。
