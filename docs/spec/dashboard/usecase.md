# Dashboard — Usecase Spec

> ダッシュボードの横断集計は購入・商品の複数集約にまたがる派生投影であり、domain Repository では
> なく QueryService 経路（`internal/usecase/dashboard/query.DashboardQueryService`）に委譲する
> （ADR-0032 (lightweight-cqrs) 軽量CQRS / `docs/rules.md` の Repository / QueryService 境界に準拠）。既存の判断基準で
> 経路が確定するため**新規 ADR は発行しない**。
> ただし**商品の登録件数は products 集約の属性による単純な COUNT** であり、`docs/rules.md` の
> 「simple filter / list / count by the Aggregate's own attributes は Repository」に該当するため
> QueryService ではなく `domain/product.Repository.Count` に置く。
> 集計は既存の購入・商品から導かれる読み取り投影のみであり、**domain エンティティを新設しないため
> `domain.md` は作成しない**。集計の意味論（除外条件 / GROUP BY 合算 / 期間境界）は infra QS の実 DB
> テストで担保する。

## Overview

ダッシュボード集計ユースケースは、集計対象期間の売上・購入ステータス別件数と、商品数を 1 つの出力 DTO へ合成して返す read-only なユースケース。1 画面 1 API（backend 合成）の正例であり、`DashboardQueryService`（usecase 層の QueryService インターフェース、実装は infra）の 2 つの集計メソッドと、商品集約の Repository による件数取得を束ねる thin orchestrator。admin 専用エンドポイント（`GET /v1/dashboard/summary`）の read source となる。

認可は Action と所有者なしリソースを宣言して Authorizer へ委譲する（ユースケースは role を検査しない）。所有者を渡さないことで所有者フォールバックが働かず、管理者だけが通る。

集計対象期間は瞬時の半開区間として受け取り、そのまま QueryService へ渡す（区分の解決も暦日への変換も行わない）。集計本体は QueryService／infra に委ねる。売上はキャンセル済みの購入を除外し未払いの購入を含む（購入レベルの絞りは商品ランキングと同じだが、ランキングはさらに公開済み商品に限るため合計は一致しない）。一方ステータス別件数はキャンセル済みも 1 ステータスとして含むため、両者の母集団は一致しない。商品数は集計期間に依存しないマスタの現在値。

QueryService の集計結果は usecase 内で `SummaryView` へ写像してから返す（DTO Boundary）。

## Interface

```yaml
package: internal/usecase/dashboard
name: Usecase
methods:
  - name: GetDashboardSummary
    signature: GetDashboardSummary(ctx context.Context, authn *auth.Authn, window timewindow.Window) (SummaryView, error)
```

## DTOs

```yaml
- name: SummaryView
  description: ダッシュボード横断集計の usecase 出力 DTO。SalesAmount は USD セント単位の整数。
  fields:
    - name: SalesAmount
      type: int64
    - name: SalesCount
      type: int64
    - name: PurchaseStatusCounts
      type: "[]StatusCountView"
    - name: TotalProductCount
      type: int64
    - name: PublishedProductCount
      type: int64
- name: StatusCountView
  description: ステータス別件数 1 件分の出力 DTO。ステータスは購入ステータスマスタで解決済み。
  fields:
    - name: StatusID
      type: uuid.UUID
    - name: StatusName
      type: string
    - name: Count
      type: int64
```

## Dependencies

```yaml
- tracer                     # observability.TracerFactory -> LayerTracer
- authorizer                 # usecase/boundary/authz.Authorizer（admin 限定の認可判断を委譲）
- dashboard_query_service    # usecase/dashboard/query.DashboardQueryService（期間集計の投影を取得）
- product_repository         # domain/product.Repository（商品の登録件数を取得）
```

## Workflow

### GetDashboardSummary

```yaml
tx_required: false
steps:
  - authn が nil の場合は未認証エラーを返す
  - dashboard 参照 Action と所有者なしリソースで認可する（管理者以外は拒否）
  - dashboard_query_service.SummarizeSales で期間内の売上合計と件数を取得する
  - dashboard_query_service.CountPurchasesByStatus で期間内のステータス別件数を取得する
  - product_repository.Count で商品の総数と公開数を取得する
  - 3 つの集計結果を SummaryView へ合成して返す
calls:
  - authorizer.Authorize
  - dashboard_query_service.SummarizeSales
  - dashboard_query_service.CountPurchasesByStatus
  - product_repository.Count
errors:
  - authn が nil の場合は ErrUnauthenticated を返す
  - 管理者でない場合は Authorizer のエラー（ErrPermissionDenied）を伝播する
  - dashboard_query_service および product_repository のエラーをそのまま伝播する
```

## Notes

- **スナップショット一貫性は保証しない。** 3 つの取得はそれぞれ独立したクエリであり、実行の合間に書き込みが入ると指標間でわずかにずれうる。トランザクションで束ねても現行の分離レベルでは文ごとに新しいスナップショットを取るため一貫性は得られず、read-only の反復可能読み取りを導入するには tx 境界の拡張が要る。運用ダッシュボードの用途ではこのずれを許容し、必要になった時点で別途対応する。
- **集計対象期間は瞬時の半開区間で受け取り、サーバは時計も暦も持たない。** 「今日」「今月」のような相対的な期間は呼び出し側が瞬時へ解決してから送るため、集計経路にタイムゾーンの解釈が入らず、同じ URL はいつ呼んでも同じ期間を指す。境界は各々独立に省略でき、両方省略すれば全期間が対象になる。区間の相関検証（上限が下限以下なら不正引数エラー）は `timewindow.New` が担い、ハンドラで組み立てた時点で検証済みの区間だけが usecase へ届く。
- **売上の期間絞り込みに専用の索引は追加していない。** 既存の複合索引は先頭列が別の絞り込み条件のため、期間だけを条件とする本集計には効かない。サンプル規模では逐次走査を許容する判断であり、データ量が増える利用では期間列の索引追加を検討する。
