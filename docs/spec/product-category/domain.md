# ProductCategory — Domain Spec

> `products`（#563）が商品カテゴリ参照（`CategoryRef`。ID と名称の組の値オブジェクト）で保持する
> 商品カテゴリマスタ集約。`GET /v1/products/categories`（一覧取得 usecase は `usecase.md`）の全件一覧は
> QueryService ではなく Repository の simple list（`FindAll`）として提供する
> （ADR-0031 (lightweight-cqrs) / `docs/rules.md` の Repository 境界に準拠）。

## Overview

商品カテゴリ集約は、商品カテゴリの ID・名称・コード・表示順（`sortKey`）を保持する参照系のエンティティ。`products` 集約は商品カテゴリを `CategoryRef`（ID と名称の組）で保持し、名称は生成・更新の時点で `FindByID` からこの集約を引いて埋め込む。一覧の表示順は `code` ではなく `sortKey` 昇順で管理する。生成時に ID・名称長・コード範囲・表示順範囲を検証する。マスタは migration で seed され、書き込み API を持たない。

## Entity

```yaml
package: internal/domain/product/category
struct: Category
fields:
  - name: id
    type: uuid.UUID
    required: true
  - name: name
    type: string
    required: true
    min_length: 1
    max_length: 100       # VARCHAR(100)
  - name: code
    type: int
    required: true
    min: 1                # 正の SMALLINT
    max: 32767            # SMALLINT 上限
  - name: sortKey
    type: int
    required: true
    min: 1                # 正の SMALLINT
    max: 32767            # SMALLINT 上限
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
  signature: FindAll(ctx context.Context) (Categories, error)
  behavior: 全商品カテゴリを sortKey 昇順で取得する（GET /v1/products/categories の全件一覧。単一集約・無フィルタ・無ページングの simple list）。
- name: FindByID
  signature: FindByID(ctx context.Context, id uuid.UUID) (*Category, error)
  behavior: ID から単一の商品カテゴリを取得する。未存在は NotFound を返す（products 集約が生成・更新時に CategoryRef を解決するために使う）。
```
