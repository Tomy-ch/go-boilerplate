# PurchaseStatus — Usecase Spec

> 全件一覧は単一集約・無フィルタ・無ページングの simple list であり、QueryService ではなく
> domain `status.Repository` の `FindAll` に委譲する（ADR-0032 (lightweight-cqrs) / `docs/rules.md` の
> Repository 境界に準拠）。

## Overview

購入ステータス一覧ユースケースは、購入ステータスマスタの全件をマスタの表示順で返す read-only なユースケース。`status.Repository`（domain Repository）の `FindAll` に委譲し、取得した `Status` エンティティ一覧を usecase DTO（`StatusDTO`）へ写像して返す thin orchestrator。少量（9 件）のため全件返却し、ページング・トランザクションは不要。認証不要の公開エンドポイント（`GET /v1/purchases/statuses`）の read source となる。

ドメイン集約を outer 層へ露出させないため、`Status` エンティティは usecase 内で `StatusDTO` へ写像してから返す（DTO Boundary）。

## Interface

```yaml
package: internal/usecase/purchase/status
name: Usecase
methods:
  - name: ListStatuses
    signature: ListStatuses(ctx context.Context) (StatusDTOs, error)
```

## DTOs

```yaml
- name: StatusDTO
  description: 購入ステータス 1 件分の usecase 出力 DTO。domain エンティティ Status から写像する。
  fields:
    - name: ID
      type: uuid.UUID
    - name: Code
      type: int
    - name: Name
      type: string
- name: StatusDTOs
  description: StatusDTO の一覧。並びはマスタの表示順で、順序そのものが表示順を表す（値は外へ出さない）。
  type: "[]StatusDTO"
```

## Dependencies

```yaml
- tracer              # observability.TracerFactory -> LayerTracer
- status_repository   # domain/purchase/status.Repository（FindAll で全件取得）
```

## Workflow

### ListStatuses

```yaml
tx_required: false
steps:
  - status_repository.FindAll で全購入ステータスをマスタの表示順で取得する
  - 取得した Status エンティティ一覧を StatusDTO（ID / Code / Name）へ写像して返す
calls:
  - status_repository.FindAll
errors:
  - status_repository.FindAll のエラーをそのまま伝播する
```
