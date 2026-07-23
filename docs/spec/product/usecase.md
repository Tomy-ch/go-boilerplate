# Product — Usecase Spec

> 公開商品一覧（`GET /v1/products`）は単一集約・products 自身の列へのフィルタ / keyword / sort・cursor ページングであり、
> QueryService ではなく domain `product.Repository` の `FindPublishedList` に委譲する（ADR-0027 / `docs/rules.md` の Repository 境界に準拠）。

## Overview

商品一覧ユースケースは、公開済み商品を公開日時順（cursor ページネーション）で取得する read-only な thin orchestrator。`product.Repository`（domain Repository）の `FindPublishedList` に委譲し、取得した `Product` エンティティ一覧を usecase DTO（`ProductView`）へ写像して返す。

cursor ページングは「直前ページ末尾行のソートキー `(publishedAt, id)` を不透明トークン化して次ページを取得する」keyset 方式。usecase は不透明カーソルの符号化・復号（`encodeProductCursor` / `decodeProductCursor`）を担い、domain へは境界を primitive（`AfterPublishedAt` / `AfterID`）で渡す。次ページ有無は `limit + 1` 件取得して超過分で判定し（`hasNext`）、超過時のみ末尾行から `NextCursor` を生成する。

sort は `-publishedAt`（既定=降順）/ `publishedAt`（昇順）の 2 値のみ。controller が enum を `Ascending` bool に写像して渡す。ステータス可視範囲の絞り込みは後続 PBI（#555）で対応する。ドメイン集約を outer 層へ露出させないため、`Product` は usecase 内で `ProductView` へ写像してから返す（DTO Boundary）。

## Interface

```yaml
package: internal/usecase/product
name: Usecase
methods:
  - name: ListProducts
    signature: ListProducts(ctx context.Context, params ListProductsParams) (*ProductListView, error)
  - name: GetProduct
    signature: GetProduct(ctx context.Context, id uuid.UUID) (ProductView, error)
```

## DTOs

```yaml
- name: ListProductsParams
  description: 公開商品一覧取得の入力。cursor（取得件数 + 境界）とフィルタ・並び順を保持する。
  fields:
    - name: Cursor
      type: "*paging.Cursor"
    - name: CategoryID
      type: "*uuid.UUID"
    - name: StatusID
      type: "*uuid.UUID"
    - name: Keyword
      type: "*string"
    - name: Ascending
      type: bool
- name: ProductView
  description: 商品 1 件分の usecase 出力 DTO。domain エンティティ Product から写像する。price は USD セント整数。
  fields:
    - name: ID
      type: uuid.UUID
    - name: Name
      type: string
    - name: Description
      type: "*string"
    - name: Price
      type: int
    - name: Quantity
      type: int
    - name: StockWarningThreshold
      type: "*int"
    - name: StatusID
      type: uuid.UUID
    - name: CategoryID
      type: uuid.UUID
    - name: PublishedAt
      type: time.Time
- name: ProductListView
  description: 公開商品一覧（cursor ページネーション）の取得結果。
  fields:
    - name: Items
      type: "[]ProductView"
    - name: NextCursor
      type: "*string"        # 最終ページの場合は nil
```

## Dependencies

```yaml
- tracer              # observability.TracerFactory -> LayerTracer
- product_repository  # domain/product.Repository（FindPublishedList で公開商品を keyset 取得 / FindPublishedByID で公開商品を単件取得）
```

## Workflow

### ListProducts

```yaml
tx_required: false
steps:
  - Cursor が nil の場合は apperror.ErrInvalidArgument を返す
  - decodeProductCursor で不透明カーソルを keyset 境界（publishedAt, id）へ復号する（先頭ページは境界なし）
  - domain の ListParams を組み立てる（Limit=Cursor.Limit32()+1、Ascending、CategoryID/StatusID/Keyword、AfterPublishedAt/AfterID）
  - product_repository.FindPublishedList で公開商品を取得する
  - 取得件数が Cursor.Limit() を超える場合は次ページありと判定し、末尾を切り詰める
  - 各 Product を ProductView（ID / Name / Description / Price / Quantity / StockWarningThreshold / StatusID / CategoryID / PublishedAt）へ写像する
  - 次ページありの場合、切り詰め後の末尾行から encodeProductCursor で NextCursor を生成する
calls:
  - product_repository.FindPublishedList
errors:
  - Cursor が nil の場合は apperror.ErrInvalidArgument
  - カーソル復号失敗時は apperror.ErrInvalidArgument（decodeProductCursor 由来）
  - product_repository.FindPublishedList のエラーをそのまま伝播する
```

### GetProduct

```yaml
tx_required: false
steps:
  - product_repository.FindPublishedByID で公開中の単一商品を取得する（未存在・非公開はいずれも NotFound）
  - Product を ProductView（ID / Name / Description / Price / Quantity / StockWarningThreshold / StatusID / CategoryID / PublishedAt）へ写像する
calls:
  - product_repository.FindPublishedByID
errors:
  - product_repository.FindPublishedByID のエラーをそのまま伝播する（未存在・非公開は apperror.ErrNotFound → 404）
```
