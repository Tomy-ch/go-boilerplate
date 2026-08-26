# ProductRanking — Usecase Spec

> 商品ランキングは購入明細・購入・商品の複数集約を横断する集計投影であり、domain Repository
> ではなく QueryService 経路（`internal/usecase/product/ranking/query.ProductRankingQueryService`）
> に委譲する（ADR-0032 (lightweight-cqrs) 軽量CQRS / `docs/rules.md` の Repository / QueryService 境界に準拠）。
> 集計の意味論（GROUP BY 合算 / JOIN / キャンセル除外 / tiebreak 安定）は infra QS の実 DB テストで担保する。

## Overview

商品売上ランキングユースケースは、購入明細を商品単位で集計し、**軸ごとに別の口で**上位 N 件を返す read-only なユースケース。販売数量（`SUM(quantity)`）の降順と、売上金額（`SUM(unit_price * quantity)`）の降順の 2 軸を持つ。`ProductRankingQueryService`（usecase 層の QueryService インターフェース、実装は infra）の `ListQuantityRanking` / `ListAmountRanking` に委譲する thin orchestrator。`GET /v1/products/ranking/quantity`（認証不要の公開）と `GET /v1/products/ranking/amount`（認証必須）の
read source となる。金額軸だけ認証を要求するのは、販売数量と違い実売上額を露出するためで、
`GET /v1/dashboard/summary` の `salesAmount` と扱いを揃えている。

2 つの軸は**母集団が同一**（公開中の商品 × キャンセルされていない購入の明細）で、集計する指標と並び順だけが異なる。軸を 1 応答に混ぜないのは、ランキングでは並び順そのものが情報であり、数量順の表に金額の列を添えるとその列も順位として読まれるため。

入力の正規化（`limit` の既定値適用とクランプ）のみを担い、集計本体は QueryService／infra に委ねる。キャンセル済み（`canceled_at` 設定済み）の購入は除外し、未払いの購入は含む。集計対象期間は瞬時の半開区間として受け取り、そのまま QueryService へ渡す。境界の算出はどの層も行わない（相対的な期間は呼び出し側が瞬時へ解決してから送る）ため、infra は現在時刻に依存しない。

QueryService の集計結果（`QuantityRankingResult` / `AmountRankingResult`）は usecase 内で軸ごとの出力 DTO へ写像してから返す（DTO Boundary）。

売上金額は**価格スケールの正確な decimal** で返し、決済スケール（セント整数）へは丸めない。ADR-0038 (two-scale-quantity-model) が丸めを「正確な量が決済される値になる 1 箇所」に限っており、参照系の集計はその 1 箇所ではないため。同じ判断は `GET /v1/users/me/purchases/summary` の `ItemsTotal` が先行している。丸めると順位の根拠が表示値と食い違い、同額へ潰れた 2 商品の並びが説明できなくなる副作用もある。

## Interface

```yaml
package: internal/usecase/product/ranking
name: Usecase
methods:
  - name: GetQuantityRanking
    signature: GetQuantityRanking(ctx context.Context, params GetRankingParams) (QuantityRankingView, error)
  - name: GetAmountRanking
    signature: GetAmountRanking(ctx context.Context, params GetRankingParams) (AmountRankingView, error)
```

## DTOs

```yaml
- name: GetRankingParams
  description: ランキング取得の入力。window は瞬時の半開区間（ゼロ値は全期間）、limit は 0 以下で既定値・範囲外はクランプ。
  fields:
    - name: Window
      type: timewindow.Window
    - name: Limit
      type: int
- name: QuantityRankingView
  description: 販売数量ランキングの usecase 出力 DTO。販売数量の降順（同数量は商品 ID 昇順）で並ぶ。
  fields:
    - name: Rankings
      type: "[]QuantityRankingItemView"
- name: QuantityRankingItemView
  description: 販売数量ランキング 1 商品分の出力 DTO。Price はサブセント精度を保持する十進量。
  fields:
    - name: ProductID
      type: uuid.UUID
    - name: Name
      type: string
    - name: Price
      type: decimal.Decimal
    - name: SoldQuantity
      type: int64
- name: AmountRankingView
  description: 売上金額ランキングの usecase 出力 DTO。売上金額の降順（同額は商品 ID 昇順）で並ぶ。
  fields:
    - name: Rankings
      type: "[]AmountRankingItemView"
- name: AmountRankingItemView
  description: 売上金額ランキング 1 商品分の出力 DTO。SalesAmount は明細金額の合計で、決済スケールへ丸めない。
  fields:
    - name: ProductID
      type: uuid.UUID
    - name: Name
      type: string
    - name: Price
      type: decimal.Decimal
    - name: SalesAmount
      type: decimal.Decimal
```

## Dependencies

```yaml
- tracer                          # observability.TracerFactory -> LayerTracer
- product_ranking_query_service   # usecase/product/ranking/query.ProductRankingQueryService（ListQuantityRanking / ListAmountRanking で集計投影を取得）
```

## Workflow

### GetQuantityRanking

```yaml
tx_required: false
steps:
  - limit を正規化する（0 以下は既定 10、範囲外は [1, 100] にクランプ）
  - product_ranking_query_service.ListQuantityRanking で販売数量降順の集計結果を取得する
  - 取得した各行が product.IsPublished を満たすことを確かめる（集計が公開中の商品に絞られている前提の乖離検出）
  - 取得した QuantityRankingResult 一覧を QuantityRankingItemView へ写像し QuantityRankingView として返す
calls:
  - product_ranking_query_service.ListQuantityRanking
errors:
  - product_ranking_query_service.ListQuantityRanking のエラーをそのまま伝播する
  - 取得行が product.IsPublished を満たさない場合は apperror.ErrInternal（500。集計と公開判定の乖離）
```

### GetAmountRanking

```yaml
tx_required: false
steps:
  - limit を正規化する（GetQuantityRanking と同一の方針）
  - product_ranking_query_service.ListAmountRanking で売上金額降順の集計結果を取得する
  - 取得した各行が product.IsPublished を満たすことを確かめる（同上）
  - 取得した AmountRankingResult 一覧を AmountRankingItemView へ写像し AmountRankingView として返す
    （SalesAmount は価格スケールのまま写す。丸めない理由は Overview を参照）
calls:
  - product_ranking_query_service.ListAmountRanking
errors:
  - product_ranking_query_service.ListAmountRanking のエラーをそのまま伝播する
  - 取得行が product.IsPublished を満たさない場合は apperror.ErrInternal（500。集計と公開判定の乖離）
```
