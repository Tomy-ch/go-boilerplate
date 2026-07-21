# ProductStatus — Usecase Spec

> 全件一覧は単一集約・無フィルタ・無ページングの simple list であり、QueryService ではなく
> domain `product_status.Repository` の `FindAll` に委譲する（ADR-0027 / `docs/rules.md` の
> Repository 境界に準拠）。

## Overview

商品ステータス一覧ユースケースは、商品ステータスマスタの全件を `sortKey` 昇順で返す read-only なユースケース。`product_status.Repository`（domain Repository）の `FindAll` に委譲し、取得した `ProductStatus` エンティティ一覧を usecase DTO（`ProductStatusDTO`）へ写像して返す thin orchestrator。少量（10 件）のため全件返却し、ページング・トランザクションは不要。認証不要の公開エンドポイント（`GET /v1/product-statuses`）の read source となる。

ドメイン集約を outer 層へ露出させないため、`ProductStatus` エンティティは usecase 内で `ProductStatusDTO` へ写像してから返す（DTO Boundary）。

## Interface

```yaml
package: internal/usecase/product_status
name: Usecase
methods:
  - name: ListProductStatuses
    signature: ListProductStatuses(ctx context.Context) (ProductStatusDTOs, error)
```

## DTOs

```yaml
- name: ProductStatusDTO
  description: 商品ステータス 1 件分の usecase 出力 DTO。domain エンティティ ProductStatus から写像する。
  fields:
    - name: ID
      type: uuid.UUID
    - name: Code
      type: int
    - name: Name
      type: string
    - name: SortKey
      type: int
- name: ProductStatusDTOs
  description: ProductStatusDTO の一覧（sortKey 昇順）。
  type: "[]ProductStatusDTO"
```

## Dependencies

```yaml
- tracer                      # observability.TracerFactory -> LayerTracer
- product_status_repository   # domain/product_status.Repository（FindAll で全件取得）
```

## Workflow

### ListProductStatuses

```yaml
tx_required: false
steps:
  - product_status_repository.FindAll で全商品ステータスを sortKey 昇順で取得する
  - 取得した ProductStatus エンティティ一覧を ProductStatusDTO（ID / Code / Name / SortKey）へ写像して返す
calls:
  - product_status_repository.FindAll
errors:
  - product_status_repository.FindAll のエラーをそのまま伝播する
```
