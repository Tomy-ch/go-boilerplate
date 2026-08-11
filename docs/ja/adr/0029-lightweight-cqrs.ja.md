---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, cqrs, architecture]
---

# ADR-0029: 軽量CQRSの採用 — 書き込みにRepository、読み込みにQueryService

English canonical: [0029-lightweight-cqrs.md](../../adr/0029-lightweight-cqrs.md)

## ステータス

accepted

## 背景

純粋なオニオンアーキテクチャでは、すべての永続化はドメインレイヤーで定義されたRepositoryインターフェースを介して行われる。これは集約レベルの書き込みとシンプルな読み込み（IDによるフェッチ・自身の属性によるカウント）には適切である。RepositoryはドメインのInvariantを守り完全な集約を再構築するからである。

このモデルはユースケース固有の読み込み要件で崩壊する。

- **集約をまたぐジョイン**: 多くの集約をつなぎ合わせてダッシュボードを構築するケースや、複数の集約境界をまたいでデータをグループ化・集計するケースは、ジョインと集計の詳細をドメインに漏洩させることなく単一集約のRepositoryメソッドとして表現できない。
- **ビュー形状のDTO**: APIレスポンスは集約データのサブセットや再成形（ページネーションメタデータ・ネストされたオブジェクト）を必要とすることが多い。完全な集約を返してユースケースレイヤーでマッピングするのは正しいが、大きな読み込みセットには非効率である。
- **複雑なフィルタリングと全文検索**: 複数カラムにわたるキーワード検索・全文検索・ページネーション結果は、ドメインインターフェースにきれいに収まらないクエリパターンを必要とする。

これらのクエリをRepositoryに押し込めると、ドメインインターフェースにビュー固有のメソッドが蓄積し集約のカプセル化が損なわれ、ドメインがプレゼンテーション関心事に依存するようになる。逆の極端 — 別個の読み込みデータベース・イベントプロジェクション・結果整合性を持つフルCQRS — は、現在のスケールでは正当化されない大きなインフラと運用上の複雑さをもたらす。プロジェクトには中間的なアプローチが必要である。

## 決定

同一のPostgreSQLインスタンス上で**軽量CQRS**を採用し、永続化を3つの責務に分割する。

### Repository（集約単位に基づく CRUD と一部集計処理）

- インターフェースは**ドメインレイヤー**で定義する（`internal/domain/<aggregate>/<aggregate>_repository.go`）。
- 集約の永続化とシンプルな単一集約の読み込みを担当する：IDによるフェッチ・集約自身の属性によるシンプルなフィルタ / リスト / カウント。
- 集約をまたぐジョイン・集計・キーワード検索は扱わない。
- 実装は`internal/infrastructure/rdb/repository/<aggregate>/`に置く。

### QueryService（クエリ / 読み込みパス）

- インターフェースは**ユースケースレイヤー**で定義する（`internal/usecase/<aggregate>/query/`）。読み込みモデルはドメインのInvariantではなくユースケースの関心事であるため、ドメインではなくユースケースに属する。
- 集約境界をまたぐ読み込み・複数テーブルのJOIN・ページネーション・全文検索、または full aggregate として再構築するのが無駄になる read-model の projection（重い集約のサブセット/再成形、または結合ビュー）を返す読み込みを扱う。
- ドメインエンティティではなくDTOを返す。
- どの読み取りが集約単位の Repository 読み取りへ分解できないか —— そして一般的なケースを決める明確化（DTO の形、ストレージエンジン、全文検索、集約横断のフィールド解決、参照マスタ）—— は[`docs/design/data-access-pattern.md`](../design/data-access-pattern.ja.md)に一度だけ記述する。
- 実装は`internal/infrastructure/rdb/query_service/<aggregate>/`に置く。

### Command Service（コマンド / 書き込みパス）

- インターフェースは**ユースケースレイヤー**で定義する（`internal/usecase/<workflow>/command/`）。集約横断の書き込みポートの所有は、集約軸ではなく**ワークフロー軸**で決める。所有する集約を選ぼうとしても一般化しない：現実の書き込み（例えばクーポンの消込）は user・product・purchase の Invariant を同時に強制し得るため、「その Invariant を強制する集約」は関数ではなく関係にしかならない。対してトランザクションには常にちょうど 1 つの起点があり、[`docs/rules.md`](../rules.ja.md) は既にトランザクション境界の所有をユースケースレイヤーに与えている。`internal/usecase/purchase/command/` が名指すのはワークフローであって集約ではないため、「なぜ product ではなく purchase なのか」という問い自体が生じない。
- したがって Domain Service と CommandService は意図的に分岐する：Domain Service は*規則*でありトランザクションを所有しない。CommandService は*トランザクションの道具*であり、トランザクションを開く側が所有する。CommandService が強制する条件がドメインの Invariant から導出されていることは、後述の導出規則が担保するのであって、ファイルの置き場所が担保するのではない。
- 書き込み処理を実行後は Usecase 側で、変更した Command の所属するドメインを Repository 経由で呼び出して値の検証を実施することでドメインの健全性を保つ。
- Usecase の返り値はドメインエンティティではなく DTO を返す。
- 実装は `internal/infrastructure/rdb/command_service/<aggregate>/` に置く。

> **実装状況**: `persistenceModule`（`internal/di/module/persistence.go`）の `command_service` サブモジュールは、プロバイダーが 0 個でも正当である。本セクションは意図した設計を記述しており、自分自身の集約横断の書き込みを持たないシステムには登録すべきものが無い。空のサブモジュールは欠陥ではない。

<!-- boilerplate-only:begin -->
> 上流のボイラープレートがここにサンプルの占有者を 1 つ残している理由は [boilerplate-only conventions](../get-started/boilerplate-only-conventions.ja.md) に記録されている。そこから作られたプロジェクトには適用されない。
<!-- boilerplate-only:end -->

Repository・QueryService・CommandService はいずれも `internal/di/module/persistence.go` の `persistenceModule` に登録され、Uber Fx 経由でインジェクトされる（[ADR-0037](0037-uber-fx-di.ja.md)参照）。これはフルCQRSではない：別個の読み込みストア・イベントソーシング・結果整合性のプロジェクションパイプラインは存在しない。

日々の境界適用ルールは[`docs/rules.md`](../rules.ja.md)の§ "Repository / QueryService Rules"参照。

## 影響

### ポジティブな影響

- Repositoryが集約に集中したまま保たれる。ドメインインターフェースにビュー固有のメソッドが蓄積せず、[ADR-0002](0002-onion-architecture.ja.md)に従ってドメインの純粋性が保たれる。
- QueryServiceはドメインロジックに触れることなく、また読み込みパスにドメインエンティティを露出させることなく、クエリ（ジョイン・ページネーション・全文検索）を自由に最適化できる。
- 2 つの Service インターフェースはともにユースケースレイヤーに置かれるが、所有される軸は**異なる**。QueryService は集約軸（`<aggregate>/query/`）——リードモデルは、それが射影する状態を持つ集約が形を決めるため。CommandService はワークフロー軸（`<workflow>/command/`）——トランザクションには起点がちょうど 1 つあり、集約跨ぎの書き込みを所有する単一の集約は存在しないため。いずれにせよドメインレイヤーが持つ永続化契約は Repository ただ 1 つに保たれる。
- CommandService はドメインロジックに触れることなく、柔軟な更新や削除などの処理を自由に最適化できる。最後にドメインを経由することでドメインとしての健全性の毀損を回避できる。
- 3 つの抽象がすべてインターフェース背後に置かれ DI 経由でインジェクトされ、[ADR-0001](0001-avoid-lock-in.ja.md)に従って交換可能に保たれる。
- 新しいインフラ依存はなく、3 つのパスすべてが同一の PostgreSQL インスタンスで動作する。

### ネガティブな影響

- 3 つの永続化抽象（Repository と QueryService、CommandService）があるため、開発者は特定の読み込みに対してどちらを使うかを決める必要がある。境界は `docs/rules.md` に文書化されているが理解を要する。
- 2 つの Service インターフェースはユースケースレイヤーにありドメインからより遠いため、ドメインコードを単独で読む際に意図が分かりにくくなることがある。とくに、その集約の Repository を通らない書き込み経路が存在することは、ドメインパッケージを読むだけでは分からない。後述の適格基準と導出規則が、その経路が汎用の逃げ道になることを防ぐ役割を負う。
- 「Repositoryに複雑な読み込みを置かない」境界はレビューで維持する必要があり、この区別にコンパイラによる強制はない。

## 検討した代替案

### ファットRepository（すべての読み込みをRepositoryに）

ジョイン・ページネーション・キーワード検索を含むすべての読み込みをドメインRepositoryインターフェースに置く。理解しやすくシンプルで1つの永続化抽象のみが必要になる。

ドメインインターフェースにビュー固有のクエリが蓄積し、ドメインをプレゼンテーション要件に結合し、時間とともに集約のカプセル化が損なわれるため却下。ドメインはAPIがどのようにレスポンスを整形するかを知るべきでない。

### 別個の読み込みストアを持つフルCQRS

専用の読み込みデータベース（例：Elasticsearch・イベントプロジェクションで更新されるマテリアライズドビューを持つ読み込みレプリカ）を維持する。強力な読み込みスケーラビリティとNLPグレードの検索を提供する。

現在のデータセットとクエリの複雑さは別個のストアを必要としない。結果整合性とプロジェクションの維持が、まだ正当化されない運用オーバーヘッドを追加するため時期尚早として却下。

### すべての読み込みをQueryService経由（Repositoryの読み込みを廃止）

Repositoryから読み込みメソッドを完全に排除し、すべての読み込みをQueryService経由にする。境界をシンプルにするが、些細な単一集約の参照（例：書き込み前提条件チェックのためのIDによるユーザーフェッチ）にQueryServiceのオーバーヘッドを強制する。

**ドメイン貧血**を引き起こす完全な DDD アンチパターンとして却下。集約の操作（ビジネスルール・Invariant チェック・コマンドの前提条件）はRepositoryを通じて集約の状態を読み込むことに依存しており、その読み込みを排除すると、ドメインが自身のInvariantを検証するために必要なデータを失い、空洞化した貧血ドメインになる。これは軽微なトレードオフではなく、[ADR-0002](0002-onion-architecture.ja.md)で確立したオニオンアーキテクチャを根本的に損なう。

### ユースケースレベルのみのCQRS（QueryService抽象なし）

ユースケースが複雑な読み込みのためにRepositoryメソッドを直接呼び出しインメモリジョインを適用する。新しい抽象を避けられるが、N+1クエリとパフォーマンスの懸念をアプリケーション層に移す。パフォーマンスと正確性の理由で却下。

## 補足

- Source: [`internal/infrastructure/rdb/query_service/README.md`](../../../internal/infrastructure/rdb/query_service/README.ja.md)の§ "Relationship to CQRS"および§ "When to Use QS Over Repository"。
- Source: [`docs/rules.md`](../rules.ja.md)の§ "Repository / QueryService Rules"。
- DI登録: [`internal/di/module/persistence.go`](../../../internal/di/module/persistence.go)。
- 関連: [ADR-0030](0030-system-cqrs-dml-category.ja.md)（CQRSの外に位置する第4カテゴリとしてのsystem_cqrs）。
