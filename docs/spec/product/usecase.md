# Product — Usecase Spec

> 公開商品一覧（`GET /v1/products`）は単一集約・products 自身の列へのフィルタ / keyword / sort・cursor ページングであり、
> QueryService ではなく domain `product.Repository` の `FindPublishedList` に委譲する（ADR-0027 / `docs/rules.md` の Repository 境界に準拠）。

## Overview

商品一覧ユースケースは、公開済み商品を公開日時順（cursor ページネーション）で取得する read-only な thin orchestrator。`product.Repository`（domain Repository）の `FindPublishedList` に委譲し、取得した `Product` エンティティ一覧を usecase DTO（`ProductView`）へ写像して返す。

cursor ページングは「直前ページ末尾行のソートキー `(publishedAt, id)` を不透明トークン化して次ページを取得する」keyset 方式。usecase は不透明カーソルの符号化・復号（`encodeProductCursor` / `decodeProductCursor`）を担い、domain へは境界を primitive（`AfterPublishedAt` / `AfterID`）で渡す。次ページ有無は `limit + 1` 件取得して超過分で判定し（`hasNext`）、超過時のみ末尾行から `NextCursor` を生成する。

sort は `-publishedAt`（既定=降順）/ `publishedAt`（昇順）の 2 値のみ。controller が enum を `Ascending` bool に写像して渡す。ステータス可視範囲の絞り込みは後続 PBI（#555）で対応する。ドメイン集約を outer 層へ露出させないため、`Product` は usecase 内で `ProductView` へ写像してから返す（DTO Boundary）。

商品作成（`POST /v1/products`）は admin 認可のうえ、`tx.Manager` の境界内で商品ステータス / カテゴリの名称を ID から解決し、`Product` を構築して登録する write ユースケース。マスタ不在はサーバ側整合性異常（500）、価格・在庫などの業務不変条件違反は 422 に落とす。

## Interface

```yaml
package: internal/usecase/product
name: Usecase
methods:
  - name: ListProducts
    signature: ListProducts(ctx context.Context, params ListProductsParams) (*ProductListView, error)
  - name: GetProduct
    signature: GetProduct(ctx context.Context, id uuid.UUID) (ProductView, error)
  - name: CreateProduct
    signature: CreateProduct(ctx context.Context, authn *auth.Authn, params CreateProductParams) (ProductView, error)
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
  description: 商品 1 件分の usecase 出力 DTO。domain エンティティ Product から写像する。Price は価格スケールの十進量（pkg/decimal.Decimal）で、controller が decimal 文字列へ整形する。
  fields:
    - name: ID
      type: uuid.UUID
    - name: Name
      type: string
    - name: Description
      type: "*string"
    - name: Price
      type: decimal.Decimal
    - name: Quantity
      type: int
    - name: StockWarningThreshold
      type: "*int"
    - name: StatusID
      type: uuid.UUID
    - name: CategoryID
      type: uuid.UUID
    - name: PublishedAt
      type: "*time.Time"     # 未公開は nil
    - name: ImagePath
      type: "*string"        # 画像未設定は nil
- name: CreateProductParams
  description: 商品作成の入力。price は十進文字列で受け取り usecase で decimal へ解釈する（負値は 422）。publishedAt / imagePath は nil 許容。
  fields:
    - name: Name
      type: string
    - name: Description
      type: "*string"
    - name: Price
      type: string
    - name: Quantity
      type: int
    - name: StockWarningThreshold
      type: "*int"
    - name: CategoryID
      type: uuid.UUID
    - name: StatusID
      type: uuid.UUID
    - name: PublishedAt
      type: "*time.Time"
    - name: ImagePath
      type: "*string"
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
- txm                 # boundary/tx.Manager（CreateProduct のトランザクション境界）
- product_repository  # domain/product.Repository（FindPublishedList / FindPublishedByID / Create）
- category_repository # domain/product/category.Repository（FindByID でカテゴリ名称を解決）
- status_repository   # domain/product/status.Repository（FindByID でステータス名称を解決）
- authorizer          # boundary/authz.Authorizer（CreateProduct の admin 認可）
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

### CreateProduct

```yaml
tx_required: true
steps:
  - authn が nil の場合は apperror.ErrUnauthenticated（401）を返す
  - authorizer.Authorize（ActionProductCreate / resource=product）で admin 認可を確認する（拒否は 403）
  - price（文字列）を decimal へ解釈し money.NewPrice で非負を検証する（負値は 422 / 非数値は 400。非数値は OpenAPI でも弾く）
  - uuid.New で商品 ID を採番する
  - txm.Do 内で以下を実行する:
      - status_repository.FindByID / category_repository.FindByID で名称を解決する（未存在は整合性異常として ErrInternal=500）
      - product.New で商品エンティティを構築する（負在庫・名称長超過は 422）
      - product_repository.Create で登録する（DB の FK 制約は多層防御の保険。正典の 500 は上のマスタ確認）
  - 生成した Product を ProductView（ImagePath / PublishedAt を含む）へ写像して返す
calls:
  - status_repository.FindByID
  - category_repository.FindByID
  - product_repository.Create
errors:
  - authn が nil の場合は apperror.ErrUnauthenticated（401）
  - 認可拒否は authz 由来の apperror.ErrPermissionDenied（403）
  - 負価格・負在庫・名称長超過は apperror.ErrValidation（422）
  - status_id / category_id 不在は apperror.ErrInternal（500・整合性異常）
```
