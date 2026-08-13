# ProductRanking — Usecase Spec

> 販売数量ランキングは購入明細・購入・商品の複数集約を横断する集計投影であり、domain Repository
> ではなく QueryService 経路（`internal/usecase/product/ranking/query.ProductRankingQueryService`）
> に委譲する（ADR-0030 (lightweight-cqrs) 軽量CQRS / `docs/rules.md` の Repository / QueryService 境界に準拠）。
> 集計の意味論（GROUP BY 合算 / JOIN / キャンセル除外 / tiebreak 安定）は infra QS の実 DB テストで担保する。

## Overview

商品売上ランキングユースケースは、購入明細を商品単位で集計し、販売数量（`SUM(quantity)`）の降順で上位 N 件を返す read-only なユースケース。`ProductRankingQueryService`（usecase 層の QueryService インターフェース、実装は infra）の `ListRanking` に委譲する thin orchestrator。認証不要の公開エンドポイント（`GET /v1/products/ranking`）の read source となる。

入力の正規化（`period` の区分解釈・`limit` の既定値適用とクランプ）のみを担い、集計本体・境界時刻の算出は QueryService／infra に委ねる。キャンセル済み（`canceled_at` 設定済み）の購入は除外し、未払いの購入は含む。`period=30d` の境界時刻（現在時刻依存）は infra 層に閉じ、usecase は集計区分の意図（`all` / `30d`）のみを伝達する。

QueryService の集計結果（`RankingResult`）は usecase 内で `RankingView` / `RankingItemView` へ写像してから返す（DTO Boundary）。

## Interface

```yaml
package: internal/usecase/product/ranking
name: Usecase
methods:
  - name: GetProductsRanking
    signature: GetProductsRanking(ctx context.Context, params GetRankingParams) (RankingView, error)
```

## DTOs

```yaml
- name: GetRankingParams
  description: ランキング取得の入力。period は "all" / "30d"（未知値・空は全期間）、limit は 0 以下で既定値・範囲外はクランプ。
  fields:
    - name: Period
      type: string
    - name: Limit
      type: int
- name: RankingView
  description: 商品売上ランキングの usecase 出力 DTO。販売数量の降順（同数量は商品 ID 昇順）で並ぶ。
  fields:
    - name: Rankings
      type: "[]RankingItemView"
- name: RankingItemView
  description: ランキング 1 商品分の出力 DTO。Price はサブセント精度を保持する十進量。
  fields:
    - name: ProductID
      type: uuid.UUID
    - name: Name
      type: string
    - name: Price
      type: decimal.Decimal
    - name: SoldQuantity
      type: int64
```

## Dependencies

```yaml
- tracer                          # observability.TracerFactory -> LayerTracer
- product_ranking_query_service   # usecase/product/ranking/query.ProductRankingQueryService（ListRanking で集計投影を取得）
```

## Workflow

### GetProductsRanking

```yaml
tx_required: false
steps:
  - period を集計区分へ正規化する（"30d" のみ直近30日、それ以外は全期間）
  - limit を正規化する（0 以下は既定 10、範囲外は [1, 100] にクランプ）
  - product_ranking_query_service.ListRanking で販売数量降順の集計結果を取得する
  - 取得した各行が product.IsPublished を満たすことを確かめる（集計が公開中の商品に絞られている前提の乖離検出）
  - 取得した RankingResult 一覧を RankingItemView（ProductID / Name / Price / SoldQuantity）へ写像し RankingView として返す
calls:
  - product_ranking_query_service.ListRanking
errors:
  - product_ranking_query_service.ListRanking のエラーをそのまま伝播する
  - 取得行が product.IsPublished を満たさない場合は apperror.ErrInternal（500。集計と公開判定の乖離）
```
