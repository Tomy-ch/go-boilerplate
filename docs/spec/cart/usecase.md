# Cart — Usecase Spec

> `/v1/carts/me`（カート参照・投入・削除・ログイン時マージ）の usecase spec。
> カートの主体は認証済みユーザーとゲストセッションの 2 種類あり、その解決を担うのがこの層。
> 購入への接続（カート → 購入確定）は本 usecase ではなく合成層
> [`internal/usecase/checkout`](../../../internal/usecase/checkout) が担う（後述の Notes）。

## Overview

カートユースケースは、主体（確定済みユーザー ID またはゲストのセッショントークン）からカートを解決し、
明細の設定・削除・マージ・期限切れ回収を行う。ドメインが商品を知らないため、**表示のたびに商品の現在値を
突き合わせて明細ごとの状態を判定するのはこの層の責務**である。

**再評価は失敗ではない。** 在庫不足・非公開化・値上がりのいずれも、カート取得を 4xx にはしない。カートを
見る操作は成功しており、問題があるのは明細であって要求ではないため、200 のまま明細ごとに `issues` を添えて
返す。呼び出し側（フロント）はこれを見て「この 1 件は今は買えません」と示す。エラーで返すと、買える 2 件が
何であったかを利用者に見せられなくなる。

**取得が書き込みを伴う。** 値上がり判定には前回提示した価格が要るため、表示のたびに `MarkSeen` で
`lastSeenPrice` を更新し、あわせて `Touch` で有効期限を延長する。GET が副作用を持つのは意図した設計で、
その代償として取得は tx を要求する。副作用を避けるなら値上がり通知と TTL 延長の両方を諦めることになる。

## Interface

```yaml
package: internal/usecase/cart
interface: Usecase
methods:
  - name: GetCart          # GET /v1/carts/me。再評価つき取得
    signature: GetCart(ctx context.Context, subject Subject) (CartView, error)
  - name: SetItem          # PUT /v1/carts/me/items/{productId}。数量の設定（upsert）
    signature: SetItem(ctx context.Context, params SetItemParams) (CartView, error)
  - name: RemoveItem       # DELETE /v1/carts/me/items/{productId}
    signature: RemoveItem(ctx context.Context, params RemoveItemParams) (CartView, error)
  - name: ClearCart        # DELETE /v1/carts/me
    signature: ClearCart(ctx context.Context, subject Subject) error
  - name: MergeOnLogin     # ログイン直後。ゲストカートをユーザーへ引き継ぐ
    signature: MergeOnLogin(ctx context.Context, params MergeOnLoginParams) (MergeCartView, error)
  - name: PurgeExpired     # 期限切れ掃除ジョブの実行本体
    signature: PurgeExpired(ctx context.Context, limit int32) (int, error)
```

## DTOs

```yaml
input:
  struct: Subject          # カートの主体。ちょうど一方だけが非 nil（controller が認証結果から組み立てる）
  fields:
    - name: UserID
      type: "*uuid.UUID"   # 認証済みの内部ユーザー ID
    - name: SessionToken
      type: "*string"      # 未認証時のゲストトークン。無い場合は SetItem が新規発行する

  struct: SetItemParams
  fields:
    - name: Subject
      type: Subject
    - name: ProductID
      type: uuid.UUID
    - name: Quantity
      type: int

  struct: RemoveItemParams
  fields:
    - name: Subject
      type: Subject
    - name: ProductID
      type: uuid.UUID

  struct: MergeOnLoginParams
  fields:
    - name: UserID
      type: uuid.UUID
    - name: SessionToken
      type: string         # ログイン前に持っていたゲストトークン

output:
  struct: CartView
  fields:
    - name: SessionToken
      type: "*string"      # ゲストのとき、controller が X-Cart-Session ヘッダで返す値。確定済みなら nil
    - name: Items
      type: "[]CartItemView"
    - name: SubtotalAmount
      type: int64          # 買える明細のみを合算した参考値（USD セント）。請求額ではない
    - name: ExpiresAt
      type: "*time.Time"   # カートが存在しない場合は nil（GET は行を作らないため）

  struct: CartItemView
  fields:
    - name: ProductID
      type: uuid.UUID
    - name: ProductName
      type: "*string"      # 商品の現在値。引けなかった場合は nil
    - name: Quantity
      type: int
    - name: UnitPrice
      type: "*decimal.Decimal" # 商品の現在値。引けなかった場合は nil（DTO は decimal で持つ）
    - name: Issues
      type: "[]ItemIssue"  # 空なら購入可能
    - name: AvailableQuantity
      type: "*int"         # insufficientStock のとき、今買える上限

  struct: MergeCartView
  fields:
    - name: Clamped
      type: "[]uuid.UUID"  # 数量が上限へ丸められた productID
    - name: Dropped
      type: "[]uuid.UUID"  # 明細数上限で切り捨てられた productID
```

```yaml
enum: ItemIssue            # 明細ごとの再評価結果。複数同時に立ちうる
values:
  - notFound               # 商品が存在しない（削除された）。単独で立ち、他の issue は併記しない
  - unpublished            # 非公開化された
  - outOfStock             # 在庫 0。insufficientStock とは排他
  - insufficientStock      # 在庫 < 要求数量（AvailableQuantity に上限を添える）
  - priceIncreased         # lastSeenPrice より高い
  - priceDecreased         # lastSeenPrice より安い（値下がりも知らせる。買い時の情報であり、隠す理由がない）
```

## Dependencies

```yaml
- name: tx.Manager                       # 取得も書き込みを伴うため read 系でも要求する
- name: cart.Repository                  # 解決 / ロック / 明細保存 / 所有者確定 / 期限切れ削除
- name: product.Repository               # FindByIDs（再評価の現在値取得。読み取りのみでロックしない）
- name: clock.Clock                      # now（有効期限の算出・addedAt）
- name: token.Generator                  # ゲストセッショントークンの採番（新規 boundary。暗号論的乱数）
- name: observability.TracerFactory
- name: pkg/uuid                         # cart id / cart item id の UUIDv7 採番
```

## Workflow

### GetCart

```yaml
tx_required: true            # MarkSeen / Touch の書き込みを伴う
steps:
  - subject からカートを解決する（UserID → FindByOwnerID / SessionToken → FindBySessionToken）
  - 見つからない、または IsExpired の場合は空の CartView を 200 で返して終了（GET では行を作らない）
  - 明細の productID を集めて product.Repository.FindByIDs で現在値を取得する（ロックしない）
  - 明細ごとに issues を判定する（存在 / 公開状態 / 在庫 / lastSeenPrice との比較）
  - MarkSeen で提示価格を更新し、Touch で有効期限を延長して Update
  - 買える明細のみを合算した SubtotalAmount を添えて CartView を返す
calls:
  - cart_repository.FindByOwnerID
  - cart_repository.FindBySessionToken
  - cart_repository.Update
  - product_repository.FindByIDs
  - clock.Now
errors:
  - なし（カート未作成・期限切れ・明細の問題はいずれも 200 で表現する）
```

**在庫をロックしない。** 再評価は参考情報であり、返した瞬間から古くなる。ここで `FOR UPDATE` を取ると
カート表示が購入と在庫行を奪い合い、表示のたびに購入が待たされる。正確さは購入成立時にのみ必要で、
そこでは購入側が商品行をロックして検証する。

### SetItem

```yaml
tx_required: true
steps:
  - subject からカートを解決する。無ければ作成する
      （UserID あり → NewForOwner / SessionToken なし → token.Generator で採番して NewForGuest）
  - 既存カートは LockByID で悲観ロックする
  - product_repository.FindByID で商品の存在と公開状態を確認する
      （非公開・不存在の商品はカートへ入れさせない。再評価と違い、投入は要求そのものが不正なため 422）
  - cart.SetItem（数量の設定。上限超過は ErrTooManyItems）
  - Touch して Update
  - GetCart と同じ再評価を行って CartView を返す
calls:
  - cart_repository.FindByOwnerID / FindBySessionToken / LockByID / Create / Update
  - product_repository.FindByID / FindByIDs
  - token_generator.Generate
  - clock.Now
errors:
  - ErrInvalidQuantity     -> 422
  - ErrTooManyItems        -> 422
  - product NotFound / 非公開 -> 422
```

### RemoveItem

```yaml
tx_required: true
steps:
  - subject からカートを解決する。無ければ空の CartView を返して終了（削除の冪等）
  - LockByID → cart.RemoveItem → Touch → Update
  - 再評価つき CartView を返す
calls:
  - cart_repository.FindByOwnerID / FindBySessionToken / LockByID / Update
  - product_repository.FindByIDs
errors:
  - なし（対象が無くても成功）
```

### ClearCart

```yaml
tx_required: true
steps:
  - subject からカートを解決する。無ければ何もせず終了
  - LockByID → cart.Clear → Update（カート自体は残す）
calls:
  - cart_repository.LockByID / Update
errors:
  - なし
```

### MergeOnLogin

```yaml
tx_required: true
steps:
  - SessionToken からゲストカートを取得する。無ければ空の MergeCartView を返して終了
  - UserID からユーザーカートを取得する
  - ユーザーカートが無い場合（高速路）:
      ゲストカートに AssignOwner して Update する。所有者の確定は明細の内容を変えない
  - ユーザーカートがある場合:
      2 件を id 昇順で LockByID し（ロック順序の固定）、user.Merge(guest) → Update → guest を Delete
  - Merge が報告した clamped / dropped を MergeCartView として返す
calls:
  - cart_repository.FindBySessionToken / FindByOwnerID / LockByID / Update / Delete
  - clock.Now
errors:
  - ErrAlreadyOwned -> 409（同一セッションのマージが並行して 2 回走った場合）
```

**ログインをカートの都合で失敗させない。** `Merge` 自体は error を返さず、数量は上限へクランプ、明細数超過は
古い順に残して切り捨てる。失われた分は握り潰さず `MergeCartView` で呼び出し側へ渡し、利用者に伝える。
唯一の 409 は所有者確定の二重適用で、これは失われるものが無い代わりに**起きたことを呼び出し側から
見えなくしてはならない**ため衝突として返す。

### PurgeExpired

```yaml
tx_required: false           # DeleteExpired が単一文で完結する
steps:
  - clock.Now を基準に cart_repository.DeleteExpired(now, limit) を呼び、削除件数を返す
  - 消し切ることを意図せず、ジョブ側が件数上限で区切って繰り返し呼ぶ
calls:
  - cart_repository.DeleteExpired
  - clock.Now
errors:
  - なし（削除 0 件は正常）
```

## Notes

- **checkout への接続はこの usecase が持たない。** 業務ユースケース同士が直接呼び合うと連鎖ができて
  業務操作の境界が追跡できなくなるため、カートから購入を作る経路は合成層
  [`internal/usecase/checkout`](../../../internal/usecase/checkout) が担う。checkout は cart Repository から
  明細を読み、purchase へ渡し、成立後に同一 tx でカートを空にして `cart.checkedOut.v1` を outbox へ発行する
  （その Workflow は checkout 側の spec が持つ）。cart usecase は自分から購入を知らない。
- **カートは在庫を押さえない。** 売り越しの禁止は購入成立時に商品行を悲観ロックして行われる
  （[`docs/spec/purchase/domain.md`](../purchase/domain.md)）。本 usecase の再評価は拘束力を持たず、
  表示から購入までの間に在庫が尽きることは正常な結末として扱う。
- **`token.Generator` は新規の boundary。** ゲストトークンは推測不能である必要があるため UUIDv7 では
  代替できず、暗号論的乱数を境界越しに供給する（clock と同じ理由で、usecase が乱数へ直接依存しない）。
- 認可は所有者ベースで、`Cart.IsOwnedBy` が定義する所有関係を `authz.Resource` へ渡して判定する。
  ゲストカートは `ownerID = nil`（所有者概念なし）として扱われる本リポジトリ唯一の例になり、到達経路は
  セッショントークンの秘匿のみが担保する。
