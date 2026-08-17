# ProductCategory — Usecase Spec

> 全件一覧は単一集約・無フィルタ・無ページングの simple list であり、QueryService ではなく
> domain `category.Repository` の `FindAll` に委譲する（ADR-0030 (lightweight-cqrs) / `docs/rules.md` の
> Repository 境界に準拠）。

## Overview

商品カテゴリ一覧ユースケースは、商品カテゴリマスタの全件をマスタの表示順で返す read-only なユースケース。`category.Repository`（domain Repository）の `FindAll` に委譲し、取得した `Category` エンティティ一覧を usecase DTO（`CategoryDTO`）へ写像して返す thin orchestrator。少量（5 件）のため全件返却し、ページング・トランザクションは不要。認証不要の公開エンドポイント（`GET /v1/products/categories`）の read source となる。

ドメイン集約を outer 層へ露出させないため、`Category` エンティティは usecase 内で `CategoryDTO` へ写像してから返す（DTO Boundary）。

## Interface

```yaml
package: internal/usecase/product/category
name: Usecase
methods:
  - name: ListCategories
    signature: ListCategories(ctx context.Context) (CategoryDTOs, error)
```

## DTOs

```yaml
- name: CategoryDTO
  description: 商品カテゴリ 1 件分の usecase 出力 DTO。domain エンティティ Category から写像する。
  fields:
    - name: ID
      type: uuid.UUID
    - name: Code
      type: int
    - name: Name
      type: string
- name: CategoryDTOs
  description: CategoryDTO の一覧。並びはマスタの表示順で、順序そのものが表示順を表す（値は外へ出さない）。
  type: "[]CategoryDTO"
```

## Dependencies

```yaml
- tracer                # observability.TracerFactory -> LayerTracer
- category_repository   # domain/product/category.Repository（FindAll で全件取得）
```

## Workflow

### ListCategories

```yaml
tx_required: false
steps:
  - category_repository.FindAll で全商品カテゴリをマスタの表示順で取得する
  - 取得した Category エンティティ一覧を CategoryDTO（ID / Code / Name / SortKey）へ写像して返す
calls:
  - category_repository.FindAll
errors:
  - category_repository.FindAll のエラーをそのまま伝播する
```
