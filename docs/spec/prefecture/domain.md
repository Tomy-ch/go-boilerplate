# Prefecture — Domain Spec

> 既存実装（`internal/domain/prefecture`）を spec 化したもの。手書き実装から逆生成した現状仕様。
> `prefecture` は単独の usecase パッケージを持たず、`user` usecase 等の依存集約として参照される（usecase spec は持たない）。

## Overview

都道府県集約は、都道府県の ID・名称・コードを保持する参照系のエンティティ。`user` 集約は都道府県を ID 参照（`prefectureID`）で保持し、表示名やコードはこの集約から解決する。生成時に ID・名称長・コード範囲を検証する。

## Entity

```yaml
package: internal/domain/prefecture
struct: Prefecture
fields:
  - name: id
    type: uuid.UUID
    required: true        # IsNil の場合は ErrInvalidID
  - name: name
    type: string
    required: true
    min_length: 1         # MinPrefectureNameLength
    max_length: 100       # MaxPrefectureNameLength
  - name: code
    type: int
    required: true
    min: 1                # MinCode
    max: 47               # MaxCode（都道府県数）
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
- name: FindByID
  signature: FindByID(ctx context.Context, id uuid.UUID) (*Prefecture, error)
  behavior: ID から都道府県を 1 件取得する。
- name: FindByIDs
  signature: FindByIDs(ctx context.Context, ids []uuid.UUID) (Prefectures, error)
  behavior: 複数 ID から都道府県一覧を取得する（user 一覧の都道府県名解決で N+1 を回避する用途）。
- name: FindByName
  signature: FindByName(ctx context.Context, name string) (*Prefecture, error)
  behavior: 都道府県名から都道府県を 1 件取得する（user 作成時の名前→ID 解決で使用）。
```
