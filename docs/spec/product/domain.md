# Product — Domain Spec

> `GET /v1/products`（公開商品一覧・cursor + フィルタ + keyword + sort）の read source となる商品集約の spec。
> 本 PBI では参照（一覧取得）のみを対象とし、生成・更新系は含まない。名称解決を要する `statusID` / `categoryID` は
> ID 参照のみを保持し、名称は別集約（商品ステータス / 商品カテゴリのマスタ API）で解決する（self-contained / option b）。

## Overview

商品集約は、商品の基本情報（名称・説明・価格・在庫）と分類の ID 参照（ステータス ID・カテゴリ ID）、および公開日時を保持するドメインエンティティ。`price` は USD セント単位の整数で保持する（小数を持たない）。生成時に必須・長さ・非負の不変条件を検証し、違反する `Product` は構築できない。

一覧取得は「公開済み（`publishedAt` 非 NULL）の商品を `(publishedAt, id)` の keyset ページネーションで返す」read-only な集約読み取りであり、すべて products 自身の列への操作のため QueryService ではなく domain `product.Repository` に委譲する（ADR-0027 / `docs/rules.md` の Repository 境界に準拠）。ステータスによる可視範囲の絞り込みは後続 PBI（#555）で対応する。

不透明カーソル（cursor）の符号化・復号は usecase 層の責務であり、domain の Repository は keyset 境界を primitive（`AfterPublishedAt` / `AfterID`）で受け取る。

## Entity

```yaml
package: internal/domain/product
struct: Product
fields:
  - name: id
    type: uuid.UUID
    required: true          # IsNil の場合は ErrInvalidID
  - name: name
    type: string
    required: true
    min_length: 1
    max_length: 255         # 違反時 ErrInvalidName
  - name: description
    type: "*string"
    required: false         # nil 許容（説明未設定）
  - name: price
    type: int               # USD セント単位の整数
    required: true
    min: 0                  # 負数は ErrInvalidPrice
  - name: quantity
    type: int
    required: true
    min: 0                  # 負数は ErrInvalidQuantity
  - name: stockWarningThreshold
    type: "*int"
    required: false         # nil 許容。非 nil の場合のみ 0 以上を検証（ErrInvalidStockWarningThreshold）
    min: 0
  - name: statusID
    type: uuid.UUID
    required: true          # IsNil の場合は ErrInvalidStatusID
  - name: categoryID
    type: uuid.UUID
    required: true          # IsNil の場合は ErrInvalidCategoryID
  - name: publishedAt
    type: time.Time
    required: true          # ゼロ値は ErrInvalidPublishedAt。一覧は公開済みのみを対象とするため常に非ゼロ
```

> `createdAt` / `updatedAt`（監査列）は本 read model のドメイン不変条件に不要なため保持しない。
> DB カラム `published_at` は NULL 許容だが、一覧クエリが `published_at IS NOT NULL` で絞り込むため、
> 再構築されるエンティティの `publishedAt` は常に値を持つ（NULL 行は ErrInvalidPublishedAt として ErrInternal に正規化）。

## Cross-field Invariants

- なし（各フィールドの単独制約のみ）。

## Behavior Methods

```yaml
# 派生メソッドは持たない（単純フィールド getter のみ）。本 PBI は参照専用で状態遷移メソッドを持たない。
```

## Value Objects

```yaml
# 値オブジェクトは導入しない（price は USD セント整数、分類は ID 参照）。
```

## Repository Methods

```yaml
- name: FindPublishedList
  signature: FindPublishedList(ctx context.Context, params ListParams) (Products, error)
  behavior: |
    公開済み（published_at 非 NULL）の商品を (published_at, id) の keyset ページネーションで取得する。
    params.Ascending により昇順（publishedAt ASC, id ASC）／降順（publishedAt DESC, id DESC）を切り替える。
    params.CategoryID / StatusID が非 nil の場合は該当 ID で絞り込む。
    params.Keyword が非 nil の場合は name / description への部分一致（ILIKE）で絞り込む。
    params.AfterPublishedAt / AfterID が非 nil の場合、その keyset 境界より次ページ側の行のみを返す。
    取得件数は params.Limit で上限を課す（hasNext 判定のため usecase は limit+1 を渡す）。

- name: FindPublishedByID
  signature: FindPublishedByID(ctx context.Context, id uuid.UUID) (*Product, error)
  behavior: |
    ID から公開中（published_at 非 NULL）の単一商品を取得する。
    公開述語は FindPublishedList と同一（published_at 非 NULL）で、一覧と詳細の可視範囲を一致させる。
    未存在・非公開はいずれも取得失敗を NotFound として返し、未ログイン経路へ商品の存在を秘匿する。
    可視性判断は SQL に閉じ、usecase / controller には分岐を置かない（ADR-0027: 単一集約の ID fetch は Repository）。

# ListParams（domain の read クエリ条件。不透明カーソルは持たず、境界を primitive で受け取る）
- struct: ListParams
  fields:
    - Limit int32              # 取得件数の上限
    - Ascending bool           # true=公開日時昇順 / false=降順
    - CategoryID *uuid.UUID    # nil=絞り込まない
    - StatusID *uuid.UUID      # nil=絞り込まない
    - Keyword *string          # nil=絞り込まない（name / description への ILIKE）
    - AfterPublishedAt *time.Time  # keyset 境界（先頭ページは nil）
    - AfterID *uuid.UUID           # keyset 境界（先頭ページは nil）
```
