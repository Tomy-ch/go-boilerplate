# Prefecture — Usecase Spec

> 全件一覧は単一集約・無フィルタ・無ページングの simple list であり、QueryService ではなく
> domain `prefecture.Repository` の `FindAll` に委譲する（ADR-0031 (lightweight-cqrs) / `docs/rules.md` の Repository 境界に準拠）。

## Overview

都道府県一覧ユースケースは、都道府県マスタの全件を `code` 昇順で返す read-only なユースケース。`prefecture.Repository`（domain Repository）の `FindAll` に委譲し、取得した `Prefecture` エンティティ一覧を usecase DTO（`PrefectureDTO`）へ写像して返す thin orchestrator。47 件と少量のため全件返却し、ページング・トランザクションは不要。認証不要の公開エンドポイント（`GET /v1/prefectures`）の read source となる。

ドメイン集約を outer 層へ露出させないため、`Prefecture` エンティティは usecase 内で `PrefectureDTO` へ写像してから返す（DTO Boundary）。

## Interface

```yaml
package: internal/usecase/prefecture
name: Usecase
methods:
  - name: ListPrefectures
    signature: ListPrefectures(ctx context.Context) (PrefectureDTOs, error)
```

## DTOs

```yaml
- name: PrefectureDTO
  description: 都道府県 1 件分の usecase 出力 DTO。domain エンティティ Prefecture から写像する。
  fields:
    - name: ID
      type: uuid.UUID
    - name: Code
      type: int
    - name: Name
      type: string
- name: PrefectureDTOs
  description: PrefectureDTO の一覧（code 昇順）。
  type: "[]PrefectureDTO"
```

## Dependencies

```yaml
- tracer                 # observability.TracerFactory -> LayerTracer
- prefecture_repository  # domain/prefecture.Repository（FindAll で全件取得）
```

## Workflow

### ListPrefectures

```yaml
tx_required: false
steps:
  - prefecture_repository.FindAll で全都道府県を code 昇順で取得する
  - 取得した Prefecture エンティティ一覧を PrefectureDTO（ID / Code / Name）へ写像して返す
calls:
  - prefecture_repository.FindAll
errors:
  - prefecture_repository.FindAll のエラーをそのまま伝播する
```
