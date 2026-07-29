# ProductCategory — Usecase Spec

> 全件一覧は単一集約・無フィルタ・無ページングの simple list であり、QueryService ではなく
> domain `product_category.Repository` の `FindAll` に委譲する（ADR-0027 / `docs/rules.md` の
> Repository 境界に準拠）。

## Overview

商品カテゴリ一覧ユースケースは、商品カテゴリマスタの全件を `sortKey` 昇順で返す read-only なユースケース。`product_category.Repository`（domain Repository）の `FindAll` に委譲し、取得した `ProductCategory` エンティティ一覧を usecase DTO（`ProductCategoryDTO`）へ写像して返す thin orchestrator。少量（5 件）のため全件返却し、ページング・トランザクションは不要。認証不要の公開エンドポイント（`GET /v1/product-categories`）の read source となる。

ドメイン集約を outer 層へ露出させないため、`ProductCategory` エンティティは usecase 内で `ProductCategoryDTO` へ写像してから返す（DTO Boundary）。

## Interface

```yaml
package: internal/usecase/product_category
name: Usecase
methods:
  - name: ListProductCategories
    signature: ListProductCategories(ctx context.Context) (ProductCategoryDTOs, error)
```

## DTOs

```yaml
- name: ProductCategoryDTO
  description: 商品カテゴリ 1 件分の usecase 出力 DTO。domain エンティティ ProductCategory から写像する。
  fields:
    - name: ID
      type: uuid.UUID
    - name: Code
      type: int
    - name: Name
      type: string
    - name: SortKey
      type: int
- name: ProductCategoryDTOs
  description: ProductCategoryDTO の一覧（sortKey 昇順）。
  type: "[]ProductCategoryDTO"
```

## Dependencies

```yaml
- tracer                        # observability.TracerFactory -> LayerTracer
- product_category_repository   # domain/product_category.Repository（FindAll で全件取得）
```

## Workflow

### ListProductCategories

```yaml
tx_required: false
steps:
  - product_category_repository.FindAll で全商品カテゴリを sortKey 昇順で取得する
  - 取得した ProductCategory エンティティ一覧を ProductCategoryDTO（ID / Code / Name / SortKey）へ写像して返す
calls:
  - product_category_repository.FindAll
errors:
  - product_category_repository.FindAll のエラーをそのまま伝播する
```
