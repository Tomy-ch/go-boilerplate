# Product — Domain Spec

> `GET /v1/products`（公開商品一覧・cursor + フィルタ + keyword + sort）の read source、および
> `POST /v1/products`（admin 商品作成）の write target となる商品集約の spec。`statusID` / `categoryID` は
> ID と名称の参照（`StatusRef` / `CategoryRef`）で保持し、名称は作成時に usecase が別集約（商品ステータス /
> 商品カテゴリのマスタ）から解決して埋める。

## Overview

商品集約は、商品の基本情報（名称・説明・価格・在庫）と分類の ID 参照（ステータス ID・カテゴリ ID）、および公開日時を保持するドメインエンティティ。`price` はサブセント精度を保持する価格スケール（Decimal）の値オブジェクト `money.Price` で保持する（非負は VO が担保。2 スケールモデルは ADR-0101）。生成時に必須・長さの不変条件を検証し、違反する `Product` は構築できない。

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
    type: money.Price       # 価格スケール（サブセント可の Decimal）を内包する VO。DB は無指定 NUMERIC
    required: true          # 非負は money.NewPrice が担保（負値は money.ErrNegativePrice）
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
    type: "*time.Time"
    required: false         # nil 許容（未公開）。一覧取得は published_at 非 NULL のみを返すため一覧経由では常に非 nil
  - name: imagePath
    type: "*string"
    required: false         # nil 許容（画像未設定）。無検証で保持（サニタイズは表示側の責務）
```

> `createdAt` / `updatedAt`（監査列）は本 read model のドメイン不変条件に不要なため保持しない。
> DB カラム `published_at` は NULL 許容で、未公開商品を作成できる。一覧クエリは `published_at IS NOT NULL` で
> 絞り込むため、一覧経由で再構築されるエンティティの `publishedAt` は常に値を持つ。

## Cross-field Invariants

- なし（各フィールドの単独制約のみ）。

## Behavior Methods

```yaml
# 派生メソッドは持たない（単純フィールド getter のみ）。本 PBI は参照専用で状態遷移メソッドを持たない。
```

## Value Objects

```yaml
# price は money.Price VO（internal/domain/kernel/money）で保持する。非負の価格スケール（サブセント可の Decimal）を
# 内包し、決済スケール（最小単位整数）への変換 policy（ToMinorUnit）を所有する。器の正確な十進量は
# pkg/decimal.Decimal（ADR-0102）。分類は ID 参照のまま VO を持たない。
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

- name: Create
  signature: Create(ctx context.Context, p *Product) error
  behavior: |
    商品を新規登録する。image_path を含む全列を INSERT する。
    マスタ存在は usecase が作成前に確認し、不在は整合性異常として ErrInternal（500）に落とす（正典の 500 経路）。
    テーブルの status_id / category_id の FK 制約は多層防御の保険であり、通常経路では到達しない。

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
