# Product — Domain Spec

> `GET /v1/products`（公開商品一覧・cursor + フィルタ + keyword + sort）の read source、および
> `POST /v1/products`（admin 商品作成）/ `PATCH /v1/products/{productId}`（admin 商品部分更新）/
> `PATCH /v1/products/{productId}/stock`（admin 在庫の増減）の write target となる商品集約の spec。分類は `StatusRef` / `CategoryRef`（ID と名称の組）で保持し、
> 名称は作成・更新時に usecase が別集約（商品ステータス / 商品カテゴリのマスタ）から解決して埋める。

## Overview

商品集約は、商品の基本情報（名称・説明・価格・在庫）と分類の参照（`StatusRef` / `CategoryRef`。いずれも ID と名称の組）、および公開日時を保持するドメインエンティティ。`price` はサブセント精度を保持する価格スケール（Decimal）の値オブジェクト `money.Price` で保持する（非負は VO が担保。2 スケールモデルは ADR-0035 (two-scale-quantity-model)）。生成時に必須・長さの不変条件を検証し、違反する `Product` は構築できない。

一覧取得は「公開済み（`publishedAt` 非 NULL）の商品を `(publishedAt, id)` の keyset ページネーションで返す」read-only な集約読み取りであり、すべて products 自身の列への操作のため QueryService ではなく domain `product.Repository` に委譲する（ADR-0029 (lightweight-cqrs) / `docs/rules.md` の Repository 境界に準拠）。ステータスによる可視範囲の絞り込みは後続 PBI（#555）で対応する。

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
    min: 0                  # 範囲外（負数 / 32bit 整数幅の上限超過）は ErrInvalidQuantity
    max: 2147483647         # 在庫は 32bit 整数幅で表現する
  - name: stockWarningThreshold
    type: "*int"
    required: false         # nil 許容。非 nil の場合のみ 0 以上を検証（ErrInvalidStockWarningThreshold）
    min: 0
  - name: status
    type: StatusRef         # ID と名称の組。ID が nil の場合は ErrInvalidStatusID
    required: true
  - name: category
    type: CategoryRef       # ID と名称の組。ID が nil の場合は ErrInvalidCategoryID
    required: true
  - name: publishedAt
    type: "*time.Time"
    required: false         # nil 許容（未公開）。一覧取得は published_at 非 NULL のみを返すため一覧経由では常に非 nil
  - name: images
    type: "[]Image"
    required: false         # 空許容（画像未設定）。表示順の昇順で保持し、構築時に並べ替える
  - name: version
    type: int
    required: true          # initialVersion(=1) 未満は ErrInvalidVersion。生成時は initialVersion から始まる
```

> `version` は監査列ではなく並行制御のトークンであり、API 契約（`ProductResponse` / `ProductPatchRequest`）に
> 露出して部分更新の前提条件として往復するため、エンティティが保持する。
> `createdAt` / `updatedAt`（監査列）は本 read model のドメイン不変条件に不要なため保持しない。
> DB カラム `published_at` は NULL 許容で、未公開商品を作成できる。一覧クエリは `published_at IS NOT NULL` で
> 絞り込むため、一覧経由で再構築されるエンティティの `publishedAt` は常に値を持つ。

### Image（子の値オブジェクト）

```yaml
package: internal/domain/product
struct: Image
fields:
  - name: id
    type: uuid.UUID
    required: true          # IsNil の場合は ErrInvalidID。採番は usecase が行う
  - name: imagePath
    type: string
    required: true          # 空文字は ErrInvalidImagePath。無検証で保持（サニタイズは表示側の責務）
  - name: sortKey
    type: int
    required: true
    min: 1                  # 範囲外は ErrInvalidImageSortKey
    max: 32767              # 表示順は 16bit 整数幅で表現する
```

> `NewImage` は組み立てのみを行い検証しない。表示順の重複は兄弟を見なければ判定できず、
> 子 1 件では答えが出ないため、不変条件は集約の入口（`validateImages`）が集合として検証する。
>
> このサンプルでは `product_images` に `deleted_at` を持たせ、差し替えられた画像を論理削除として残している。
> 論理削除か物理削除か、`deleted_at` か `is_deleted` かといった選択は本 boilerplate が規定するものではない。

## Cross-field Invariants

- なし（各フィールドの単独制約のみ）。

## Behavior Methods

```yaml
- name: Update
  signature: Update(attrs Attributes) error
  behavior: |
    商品属性を更新する。生成・再構築と同じ Attributes を受け取る。渡す値は部分更新を解決した後の
    確定値であり、据え置く属性には現在値が入る
    （未送信と null 明示の区別は usecase が解決する。domain は 3 状態を知らない）。
    生成時と同一の不変条件を課し、違反時はエンティティを変更せず ErrValidation 系（422）を返す。
    version はここでは進めない（採番は Repository.Update の条件付き UPDATE が行う）。

- name: AdjustStock
  signature: AdjustStock(delta int) error
  behavior: |
    在庫数を delta の分だけ増減する（正で補充、負で差し引き）。
    増減後の在庫が保持できる範囲（0 以上、32bit 整数幅の上限以下）を外れる場合は、
    エンティティを変更せず ErrInvalidQuantity（422）を返す。生成・更新と同一の検証を共有する。
    version はここでは進めない（採番は Repository.UpdateStock の条件付き UPDATE が行う）。
    在庫の増減は取得から更新までを直列化したうえで行う前提であり、直列化は Repository.LockByID が担う。

- name: EnsureVersion
  signature: EnsureVersion(expected int) error
  behavior: |
    更新要求が指すバージョンが現在のバージョンと一致することを確認し、不一致なら ErrVersionConflict（409）を返す。
    読み込み後に他者が更新した状態を、書き込みに至る前に明示的な衝突として弾く。
    並行更新に対する最終的なガードは Repository.Update の条件付き UPDATE（WHERE version = ...）であり、
    本メソッドはそこへ到達する前の早期失敗を担う。
```

## Value Objects

```yaml
# price は money.Price VO（internal/domain/lexicon/money）で保持する。非負の価格スケール（サブセント可の Decimal）を
# 内包し、決済スケール（最小単位整数）への変換 policy（ToMinorUnit）を所有する。器の正確な十進量は
# pkg/decimal.Decimal（ADR-0035 (two-scale-quantity-model)）。
# StatusRef / CategoryRef は分類の ID と名称の組を保持する VO。名称は作成・更新時に usecase が
# マスタ集約から解決して埋め、商品エンティティ自身は再解決しない。
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
    params.MinPrice / MaxPrice が非 nil の場合は、それぞれ価格の包含下限／包含上限として絞り込む。
    params.MinQuantity / MaxQuantity が非 nil の場合は、それぞれ在庫数の包含下限／包含上限として絞り込む。
    params.AfterPublishedAt / AfterID が非 nil の場合、その keyset 境界より次ページ側の行のみを返す。
    取得件数は params.Limit で上限を課す（hasNext 判定のため usecase は limit+1 を渡す）。

- name: FindAllLowStock
  signature: FindAllLowStock(ctx context.Context, limit int32) (Products, error)
  behavior: |
    在庫が在庫警告閾値以下（quantity <= stock_warning_threshold）まで減った商品を、
    在庫の少ない順（quantity ASC / 同数は id ASC）で最大 limit 件返す。
    在庫警告閾値が未設定（NULL）の商品は警告対象を持たないため WHERE で明示的に除外する
    （3 値論理による暗黙除外に頼らない）。
    補充の要否は公開状態に依存しないため published_at で絞らず、未公開商品も返す。
    在庫僅少の判定は FindPublishedList の公開判定と同じく SQL に閉じ、domain の述語メソッドも
    usecase / controller の分岐も置かない（ADR-0029 (lightweight-cqrs): 単一集約の自属性フィルタは Repository）。
    cursor ページングを持たない top-N で、limit の既定値適用とクランプは usecase が担う。

- name: FindPublishedByID
  signature: FindPublishedByID(ctx context.Context, id uuid.UUID) (*Product, error)
  behavior: |
    ID から公開中（published_at 非 NULL）の単一商品を取得する。
    公開述語は FindPublishedList と同一（published_at 非 NULL）で、一覧と詳細の可視範囲を一致させる。
    未存在・非公開はいずれも取得失敗を NotFound として返し、未ログイン経路へ商品の存在を秘匿する。
    可視性判断は SQL に閉じ、usecase / controller には分岐を置かない（ADR-0029 (lightweight-cqrs): 単一集約の ID fetch は Repository）。

- name: FindByID
  signature: FindByID(ctx context.Context, id uuid.UUID) (*Product, error)
  behavior: |
    ID から公開状態を問わない単一商品を取得する。未存在は NotFound を返す。
    公開日時の設定そのものを更新対象とするため、FindPublishedByID と異なり未公開商品も返す。
    admin の read-modify-write（部分更新）の read に用いる経路であり、存在秘匿は認可（403）が担う。

- name: LockByID
  signature: LockByID(ctx context.Context, id uuid.UUID) (*Product, error)
  behavior: |
    更新のために、ID から公開状態を問わない単一商品を悲観ロック（SELECT ... FOR UPDATE）して取得する。
    未存在は NotFound を返す。同一商品を対象とする他の書き込み（購入の在庫減算・在庫の増減）は、
    先行トランザクションの commit まで待たされるため、取得〜更新の read-modify-write が直列化される
    （ADR-0033 (ordered-pessimistic-row-locks): 購入と在庫補充は同じ行を同じロック規律で扱う）。
    結合する固定参照マスタ（ステータス / カテゴリ）はロック対象に含めない。

- name: LockByIDs
  signature: LockByIDs(ctx context.Context, ids []uuid.UUID) (Products, error)
  behavior: |
    更新のために、ID の集合から公開状態を問わない商品群を悲観ロック（SELECT ... FOR UPDATE）して
    ID 昇順で取得する。順序を id 昇順に固定するのは、複数商品を同時にロックする処理同士が
    デッドロックしないためである（ADR-0033 (ordered-pessimistic-row-locks): ロック順序を単一の全域順序に固定する規律）。
    不存在の ID はロックできず結果に現れないため、要素数は ids より少なくなり得る（不存在の検証は
    呼び出し側の責務であり、ここでは NotFound を返さない）。
    結合する固定参照マスタ（ステータス / カテゴリ）はロック対象に含めない。

- name: Create
  signature: Create(ctx context.Context, p *Product) error
  behavior: |
    商品を新規登録する。products の行に加え、保持する画像を product_images へ INSERT する。
    マスタ存在は usecase が作成前に確認し、不在は整合性異常として ErrInternal（500）に落とす（正典の 500 経路）。
    テーブルの status_id / category_id の FK 制約は多層防御の保険であり、通常経路では到達しない。

- name: Update
  signature: Update(ctx context.Context, p *Product) (int, error)
  behavior: |
    p が保持するバージョンを条件に商品を更新し（WHERE id = ... AND version = ...）、採番後のバージョンを返す。
    画像は対象に含まない（置換は ReplaceImages が担う）。
    version の加算は SQL 側（version = version + 1）で行い、採番の権威を DB に一本化する。
    条件に一致する行が無い場合は、読み込み後に他トランザクションが更新したものとして ErrVersionConflict（409）を返す
    （存在は同一トランザクション内の FindByID で確認済みのため、0 行はバージョン不一致のみを意味する）。
    この衝突は tx.Manager が透過リトライする serialization_failure（ADR-0031 (commandservice-atomicity-criterion)）と異なり、同じ内容の再送では解消しない。

- name: UpdateStock
  signature: UpdateStock(ctx context.Context, p *Product) (int, error)
  behavior: |
    p が保持するバージョンを条件に在庫数のみを更新し（WHERE id = ... AND version = ...）、採番後のバージョンを返す。
    在庫の更新でも version を進めるため、更新前のバージョンを条件とする部分更新（Update）は在庫の変化を
    上書きできず 0 行で弾かれる（悲観ロックと楽観ロックが同じ行で噛み合う）。
    LockByID で取得した行に対して呼ぶ前提であり、その経路では条件が外れることはない。
    ロックを取らずに呼ばれた場合の 0 行は ErrVersionConflict（409）として返し、在庫の上書きを防ぐ。

# 未参照画像の回収ジョブ向け（追記分）
- name: ReplaceImages
  signature: ReplaceImages(ctx context.Context, p *Product) error
  behavior: |
    商品が現在参照している画像を p が保持する画像で置き換える。
    生存している行を一括で論理削除したうえで、p の画像を INSERT する。置き換え前の行は履歴として残る。

    Update が成功した後、同じトランザクションの中で呼び出す必要がある。Update の条件付き更新が商品行の
    ロックを取ることで同一商品への置換が直列化され、バージョンが一致しない場合は画像に触れる前に中断できる。
    順序を入れ替えると、後から弾かれる更新の画像だけが先に入れ替わる。

- name: FilterExistingImagePaths
  signature: FilterExistingImagePaths(ctx context.Context, paths []string) ([]string, error)
  behavior: |
    paths のうち、いずれかの商品が現在の画像として参照しているものを重複排除して返す。
    論理削除された画像は現在の参照ではないため、生存行だけを参照元として数える。
    順序は保証せず、paths が空なら問い合わせずに空を返す。返らなかったパスは
    「どの商品からも参照されていない」＝孤児であることを意味する。
    products は論理削除列を持たないため、生存行だけが参照元になる。
    エンティティを再構築せずパス文字列だけを返すのは、後続がオブジェクトの削除可否しか見ないため。

# ListParams（domain の read クエリ条件。不透明カーソルは持たず、境界を primitive で受け取る）
- struct: ListParams
  fields:
    - Limit int32              # 取得件数の上限
    - Ascending bool           # true=公開日時昇順 / false=降順
    - CategoryID *uuid.UUID    # nil=絞り込まない
    - StatusID *uuid.UUID      # nil=絞り込まない
    - Keyword *string          # nil=絞り込まない（name / description への ILIKE）
    - MinPrice *money.Price    # nil=下限なし。指定値以上を対象とする
    - MaxPrice *money.Price    # nil=上限なし。指定値以下を対象とする
    - MinQuantity *int32       # nil=下限なし。指定値以上を対象とする
    - MaxQuantity *int32       # nil=上限なし。指定値以下を対象とする
    - AfterPublishedAt *time.Time  # keyset 境界（先頭ページは nil）
    - AfterID *uuid.UUID           # keyset 境界（先頭ページは nil）
```
