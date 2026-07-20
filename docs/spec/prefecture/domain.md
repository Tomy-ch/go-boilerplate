# Prefecture — Domain Spec

> `user` usecase 等の依存集約として参照される。`GET /v1/prefectures`（一覧取得 usecase は `usecase.md`）の
> 全件一覧は QueryService ではなく Repository の simple list（`FindAll`）として提供する（ADR-0027 /
> `docs/rules.md` の Repository 境界に準拠）。

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
- name: FindAll
  signature: FindAll(ctx context.Context) (Prefectures, error)
  behavior: 全都道府県を code 昇順で取得する（GET /v1/prefectures の全件一覧。単一集約・無フィルタ・無ページングの simple list）。
```
