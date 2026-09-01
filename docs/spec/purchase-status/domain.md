# PurchaseStatus — Domain Spec

> `purchases` が必須かつ一意に定まる FK（`status_id`）で参照する **参照マスタ**であり、独立集約ではない
> （`docs/design/data-access-pattern.md` §3.2 が `purchase_statuses` を参照マスタの実例として名指ししている）。
> `purchases` 自身の読み取りは今も JOIN で `StatusName` を解決しており、本パッケージの Repository を経由しない。
> `GET /v1/purchases/statuses`（一覧取得 usecase は `usecase.md`）の全件一覧は
> QueryService ではなく Repository の simple list（`FindAll`）として提供する
> （ADR-0032 (lightweight-cqrs) / `docs/rules.md` の Repository 境界に準拠）。
> 一覧の口を持つことは独立集約であることを意味しない。`internal/domain/prefecture` は
> 特定の集約に従属しない一般概念であるため独立集約だが、購入ステータスは購入の状態を読み解くための
> 従属次元であり、属性を usecase 層のバッチ取得で解決する形（`FindByIDs`）を取らない。

## Overview

購入ステータスは、ID・名称・コード・表示順（`sortKey`）を保持する参照マスタのエンティティ。書き込み操作を持たない
lookups-only な Repository を伴い、その不在自体が「アプリケーションはこのデータを書かない」という契約を表す
（`internal/domain/README.md` § Reference master aggregates）。業務が値集合を決める語彙（分類・ステータス）の型であり、
参照する集約に従属する次元として置く。`purchases` 集約は購入ステータスを `StatusID` と `StatusName` で保持し、名称は再構築時に JOIN で埋め込む。一覧の表示順は `sortKey` 昇順で管理する。生成時に ID・名称長・コード範囲・表示順範囲を検証する。値は migration で seed される。

`code` の検証範囲は業務上の値集合（1〜9）ではなく格納幅（1〜32767）を採る。マスタの値集合を決めるのは業務であり、行はあとから足されうるためで、範囲を業務値へ狭めると、行を足した時点で既存行の再構築が失敗する。

状態遷移可否（状態機械）はこのマスタの責務外で、`purchase` 集約の値オブジェクト `Status` が持つ。両者は同じ `code` を業務キーとして共有するが、`Status` が分岐と遷移を担い、このマスタは行の提示だけを担う。集約を跨いだ import は行わないため、`code` の一致は seed（`database/migrations/000012_create_purchase_statuses.up.sql`）が担保する。`Status` という名前が両者で重複している点は `docs/spec/glossary.md` の同音異義（未決）に記録してある。

## Entity

```yaml
package: internal/domain/purchase/status
struct: Status
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
# 状態遷移メソッドは未実装（getter のみ）。遷移可否は purchase 集約の値オブジェクト Status が持つ。
```

## Value Objects

```yaml
# 値オブジェクトは利用しない。
```

## Repository Methods

```yaml
- name: FindAll
  signature: FindAll(ctx context.Context) (Statuses, error)
  behavior: 全購入ステータスを sortKey 昇順で取得する（GET /v1/purchases/statuses の全件一覧。単一集約・無フィルタ・無ページングの simple list）。
```
