# ProductStatus — Domain Spec

> `products`（#563）が `statusID` で参照する商品ステータスマスタ集約。`GET /v1/product-statuses`
> （一覧取得 usecase は `usecase.md`）の全件一覧は QueryService ではなく Repository の simple list
> （`FindAll`）として提供する（ADR-0027 / `docs/rules.md` の Repository 境界に準拠）。

## Overview

商品ステータス集約は、商品ステータスの ID・名称・コード・表示順（`sortKey`）を保持する参照系のエンティティ。`products` 集約は商品ステータスを ID 参照（`statusID`）で保持し、表示名はこの集約から解決する。一覧の表示順は `code` ではなく `sortKey` 昇順で管理する（seed の `sortKey` は `code` と非連動）。生成時に ID・名称長・コード範囲・表示順範囲を検証する。マスタは migration で seed され、書き込み API を持たない。状態遷移可否（状態機械）はこの集約の責務外。

## Entity

```yaml
package: internal/domain/product_status
struct: ProductStatus
fields:
  - name: id
    type: uuid.UUID
    required: true        # IsNil の場合は ErrInvalidID
  - name: name
    type: string
    required: true
    min_length: 1         # MinProductStatusNameLength
    max_length: 100       # MaxProductStatusNameLength（VARCHAR(100)）
  - name: code
    type: int
    required: true
    min: 1                # MinCode（正の SMALLINT）
    max: 32767            # MaxCode（SMALLINT 上限）
  - name: sortKey
    type: int
    required: true
    min: 1                # MinSortKey（正の SMALLINT）
    max: 32767            # MaxSortKey（SMALLINT 上限）
```

## Cross-field Invariants

- なし（各フィールドは独立して検証され、複数フィールド間の整合条件はない）

## Behavior Methods

```yaml
# 状態遷移メソッドは未実装（getter のみ）。
```

## Value Objects

```yaml
# 値オブジェクトは利用しない。
```

## Repository Methods

```yaml
- name: FindAll
  signature: FindAll(ctx context.Context) (ProductStatuses, error)
  behavior: 全商品ステータスを sortKey 昇順で取得する（GET /v1/product-statuses の全件一覧。単一集約・無フィルタ・無ページングの simple list）。
```
