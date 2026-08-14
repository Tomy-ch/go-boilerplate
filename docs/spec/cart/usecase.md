# Cart — Usecase Spec

> `/v1/carts/me`（カート参照・投入・削除・ログイン時マージ）の usecase spec。
> カートの主体は認証済みユーザーとゲストセッションの 2 種類あり、その解決を担うのがこの層。
> 購入への接続（カート → 購入確定）は本 usecase ではなく合成層
> [`internal/usecase/checkout`](../../../internal/usecase/checkout) が担う（後述の Notes）。

## Overview

カートユースケースは、主体（確定済みユーザー ID またはゲストのセッショントークン）からカートを解決し、
明細の設定・削除・マージ・期限切れ回収を行う。表示のたびに商品の現在値を突き合わせる再評価で
**この層が担うのは、商品の取得・観測値の切り出し・結果の DTO への写像だけ**であり、判定そのものは
ドメインの `CartItem.Evaluate` が持つ（[`domain.md`](./domain.md)）。

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
    signature: RemoveItem(ctx context.Context, params RemoveItemParams) error
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
enum: ItemIssue            # 出力 DTO 側の再評価結果。ドメインの cart.Issue と 1:1 に対応する
values:
  - notFound               # 商品が存在しない（削除された）。単独で立ち、他の issue は併記しない
  - unpublished            # 非公開化された
  - outOfStock             # 在庫 0。insufficientStock とは排他
  - insufficientStock      # 在庫 < 要求数量（AvailableQuantity に上限を添える）
  - priceIncreased         # lastSeenPrice より高い
  - priceDecreased         # lastSeenPrice より安い（値下がりも知らせる。買い時の情報であり、隠す理由がない）
```

判定の意味は [`domain.md`](./domain.md) の `Issue` が定義する。ここに別の型を置くのは DTO 境界の
規約であり、ドメインの値をそのまま外へ出さないため。写像は網羅的で、対応を持たない値は写像せず
panic する（黙って既定値へ倒すと、ドメインに値が増えたとき応答だけが静かに嘘になる）。

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
  - LockByID で取り直す。解決との間にカート行が消えていた場合も空の CartView を返して終了
  - 明細の productID を集めて product.Repository.FindByIDs で現在値を取得する（ロックしない）
  - 商品から ProductSnapshot を切り出し、明細ごとに CartItem.Evaluate へ渡して結果を受け取る
      （引けなかった明細には nil を渡す。判定の中身はドメインが持つ）
  - MarkSeen で提示価格を更新し、Touch で有効期限を延長して Update
  - 商品の観測値を Cart.Subtotal へ渡し、返った値を SubtotalAmount として CartView に添える
      （合算も丸めもドメインが持つ。提示価格を書き換える前に呼ぶ）
calls:
  - cart_repository.FindByOwnerID
  - cart_repository.FindBySessionToken
  - cart_repository.LockByID
  - cart_repository.Update
  - product_repository.FindByIDs
  - cart_item.Evaluate
  - cart.Cart.Subtotal
  - clock.Now
errors:
  - 業務的な失敗は無い（カート未作成・期限切れ・明細の問題はいずれも 200 で表現する）
  - ErrSubtotalOutOfRange: 合算が決済スケールへ落とせない場合（422）。単価 1 件は money.Price の
    構築時に検証されるため、明細が積み上がった結果に限られる
```

**書き戻す対象はロックで取り直した集約に限る。** 解決は行ロックを取らないため、その結果は既に古く
なっている可能性がある。書き込みは集約単位で明細集合を丸ごと反映するので、解決から書き戻しまでの
間に他の操作が明細を消していた場合、古い集約を書き戻すとそれが復活する。取得は「提示価格と有効期限
だけを更新する」操作に見えるが、その永続化は集約全体の書き込みであり、他の操作と同じ規律が要る。

**在庫をロックしない。** 再評価は参考情報であり、返した瞬間から古くなる。ここで `FOR UPDATE` を取ると
カート表示が購入と在庫行を奪い合い、表示のたびに購入が待たされる。正確さは購入成立時にのみ必要で、
そこでは購入側が商品行をロックして検証する。

### SetItem

```yaml
tx_required: true
steps:
  - subject からカートを解決する。無ければ作成する
      （UserID あり → NewForOwner / ゲスト → token.Generator で採番して NewForGuest）
  - 既存カートは LockByID で悲観ロックする
  - product_repository.FindPublishedByID で商品を引く
      （非公開・不存在の商品はカートへ入れさせない。再評価と違い、投入は要求そのものが不正なため 422。
        両者を区別しないのは、未ログインの呼び出し元へ非公開商品の存在を漏らさないため）
  - cart.SetItem（数量の設定。上限超過は ErrTooManyItems）
  - GetCart と同じ再評価を行い、その後 Update する
calls:
  - cart_repository.FindByOwnerID / FindBySessionToken / LockByID / Create / Update
  - product_repository.FindPublishedByID / FindByIDs
  - token_generator.Generate
  - clock.Now
errors:
  - ErrInvalidQuantity      -> 422
  - ErrTooManyItems         -> 422
  - ErrUnavailableProduct   -> 422   # 不存在・非公開のいずれも
  - ErrSubtotalOutOfRange   -> 422   # 合計が決済スケールに収まらない
  - 作成の衝突が解消しない -> 409
```

**期限切れのカートは空から始まる。** 所有者の確定したカートは行を作り直せない（1 ユーザー 1 カート）ため、
行を保ったまま明細を捨てる。ゲストは提示されたトークンを使わず新しいカートを採番する。行の扱いは
非対称だが、観測される意味論は一致し、`GetCart` が期限切れを空のカートとして見せる契約とも食い違わない。

**提示されたセッショントークンでカートを作らない。** 未知・期限切れのいずれも採番し直し、新しい値を
`SessionToken` に載せて返す。ゲストカートへ到達できるかはトークンの秘匿だけが決めているため、その値を
クライアントに選ばせると、推測できないことの保証が丸ごと迂回される。

**作成の衝突はトランザクションごとやり直す。** カートを持たない同一主体からの並行要求は、片方が
一意制約に当たる。一意制約違反はトランザクション自体を中断させるため、同じトランザクションの中では
解決からやり直せない。やり直しは 1 回で、そこでは勝った側の行が見える（READ COMMITTED）。
やり直しても作れなかった場合だけ 409 を返す。

**数量の範囲外は 400 であって 422 ではない。** OpenAPI が `minimum` / `maximum` を宣言しているため、
範囲外はミドルウェアがドメインより手前で落とす（ADR-0015・リクエスト境界の権威は spec）。
`ErrInvalidQuantity` は第 2 の網として残り、HTTP 以外の経路から呼ばれたときに効く。

**冪等キーを使わない。** 冪等性は明細の自然キー `(cart_id, product_id)` と「設定」という意味論から
来るため、`docs/design/idempotency.md` § 1 の適用条件（自然キーを持たない書き込み）に当たらない。

### RemoveItem

```yaml
tx_required: true
steps:
  - productID が nil なら「該当する明細が無い」として何もせず終了（削除の冪等）
  - subject からカートを解決する。無ければ何もせず終了（カートは作らない）
  - LockByID。解決との間にカート行が消えていた場合も何もせず終了
  - cart.RemoveItem → Touch → Update
calls:
  - cart_repository.FindByOwnerID / FindBySessionToken / LockByID / Update
errors:
  - なし（対象が無くても成功）
```

**明細ごとの再評価を行わない。** 応答は 204 で本文を持たず、価格を 1 つも提示していない。ここで
`MarkSeen` を走らせると「提示していない価格を提示したことにする」記録が残り、次の `GetCart` で立つ
はずの `priceIncreased` を消してしまう。`SetItem` が 200 で `unitPrice` を返すのとは前提が違う。

**商品を引かない。** 非公開になった商品こそカートから外したいため、`SetItem` の
`FindPublishedByID` に当たる確認を持たない。「非公開でも許す」という分岐があるのではなく、
`product_repository` への呼び出しが 1 つも無い。

**カートを作らない。** 未知・期限切れのセッショントークンでも採番し直さない。本文が無いため
新しいトークンを返す場所がなく、削除の要求でカートが生まれるのは意味論として倒錯している
（`SetItem` との意図的な非対称）。

**有効期限は延ばす。** 削除も利用であり、明細が 0 件になってもカート自体は残る。

**カートの不在は 1 つの事実として扱う。** 解決で見つからなかった場合も、解決とロックの間に消えていた場合も
同じく成功で返す。どちらの読み取りで気づいたかによって応答を変えると、この op が宣言していない 404 が
競合したときだけ現れることになる。

### ClearCart

```yaml
tx_required: true
steps:
  - subject からカートを解決する。無ければ何もせず終了（カートは作らない）
  - LockByID。解決との間にカート行が消えていた場合も何もせず終了
  - cart.Clear → Touch → Update（カート自体は残す）
calls:
  - cart_repository.FindByOwnerID / FindBySessionToken / LockByID / Update
errors:
  - なし
```

**カートの行は消さない。** 消すと直後の操作でセッショントークンが発行し直され、利用者の同一性が
切れる。`Delete` はログイン時のマージ後のゲストカート破棄と期限切れの掃除に限り、この op は呼ばない。

**明細ごとの再評価を行わない。** 204 は本文を持たず価格を 1 つも提示していないため、`MarkSeen` を
走らせると次の `GetCart` で立つはずの `priceIncreased` を消してしまう（`RemoveItem` と同じ理由）。

**有効期限は延ばす。** 空にするのも利用であり、いま空にした利用者は戻ってきている。TTL は
「持ち主が二度と戻らないカート」を回収するために在るので、`RemoveItem` と揃えて延ばす。
明細が既に空でも延ばす（そうしないと「1 件消すと延びるのに、全部消すと延びない」が生じる）。

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
