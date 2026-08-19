# Cart — Domain Spec

> カート（`/v1/carts/me`）のドメイン spec。ゲスト（未認証）でもカートを持てる点が purchase との
> 最大の違いで、行が作られた時点では所有者が確定しておらず、ログインによって所有者を得る。
> カートは**在庫を押さえない**（予約機構を持たない）。売り越しの禁止は購入成立時の関心であり、
> カートは「買うつもりの控え」でしかない（[`docs/spec/purchase/domain.md`](../purchase/domain.md)）。
> 単価も保持しない。価格の確定は購入明細のスナップショットが担う。

## Overview

カート集約（Cart）は、所有者（確定済みユーザー、または未確定のゲストセッション）と明細（CartItem）の集合、
および有効期限を保持するドメイン集約。生成・更新時に「同一 productID の重複なし」「数量が 1 以上かつ
上限以下」「明細数が上限以下」「所有者とセッションが排他」の不変条件を検証し、違反する `Cart` は構築できない。

**所有者は後から決まる。** ゲストは `sessionToken` で追跡され、`ownerID` は nil。ログイン時、ゲストカートの
明細は所有者のカートへ `Merge` で取り込まれ、取り込み元は行ごと破棄される。ゲストカートを再所有する操作は
持たない — 所有権の移行はこの一本だけで、token を知る第三者が認証済みユーザーのカートへ到達できる経路が
状態として残らないのは、取り込み元が消えるためである。

**カートは商品を保持しない。** 価格・在庫・公開状態はいずれも持たず、`productID` の参照のみを持つ。
ドメインが商品集約を参照しないため、カートの不変条件は商品の状態変化から独立し、商品が変わっても
カートが不正にならない。

**ただし「在庫不足 / 値上がり / 非公開化」の判定はドメインが行う。** 商品を参照できないことは、判定を
ドメインの外へ出す理由にはならない（[`internal/domain/README.md`](../../../internal/domain/README.md)
§ Not reaching another aggregate is not a reason to leave the domain）。判定に要る属性だけを持つ
`ProductSnapshot` を usecase から受け取り、`CartItem.Evaluate` が答える。スナップショットは保持せず
判定のたびに渡されるため、上の独立性はそのまま保たれる。

これは「この 1 明細とこの 1 商品で在庫は足りるか」という 1 つの物についての問いなので、明細自身が
答える（同 § One thing or a set）。結果は表示のための参考情報であり、カートの不変条件を左右しない。

`id` は UUIDv7（[ADR-0037 (uuidv7-identifiers)]）で、生成は usecase 層が行いドメインへ渡す。有効期限
`expiresAt` も同様に、時刻境界（clock）から供給された `now` を受け取って算出する（ドメインは時刻へ直接
依存しない）。

## Entity

```yaml
package: internal/domain/cart
struct: Cart
constructors:
  - name: NewForGuest    # ゲスト用（sessionToken あり / ownerID なし）
  - name: NewForOwner    # ログイン済みユーザー用（Attributes で受ける。ownerID あり / sessionToken なし）
  - name: Reconstruct    # 永続化済みの再構築（Repository の読み出し / 再検証）
fields:
  - name: id
    type: uuid.UUID
    required: true          # IsNil の場合は ErrInvalidID
  - name: ownerID
    type: "*uuid.UUID"      # 確定した所有者。ゲストの間は nil
  - name: sessionToken
    type: "*SessionToken"   # ゲスト追跡用。所有者確定時に nil へ破棄される
  - name: items
    type: "[]CartItem"      # 空を許す（空カートは正当な状態。ErrEmptyItems は存在しない）
    max: maxItems           # 超過は ErrTooManyItems
  - name: expiresAt
    type: time.Time         # 有効期限。操作のたびに Touch で延長される
    required: true
  - name: createdAt
    type: time.Time         # NewFor* ではゼロ値（DB 既定 NOW()）。Reconstruct で設定
  - name: updatedAt
    type: time.Time         # 同上
```

```yaml
struct: CartItem   # 値オブジェクト
fields:
  - name: id
    type: uuid.UUID
    required: true
  - name: productID
    type: uuid.UUID
    required: true          # IsNil の場合は ErrInvalidProductID
  - name: quantity
    type: int
    min: minQuantity        # 1。下回る場合は ErrInvalidQuantity
    max: maxQuantityPerItem # 超過は ErrInvalidQuantity（Merge のクランプ上限でもある）
  - name: addedAt
    type: time.Time         # 明細が最初に入った時刻。Merge の切り捨て順序を決める
  - name: lastSeenPrice
    type: "*money.Price"    # 最後に利用者へ提示した価格。未提示は nil
                            # 確定単価ではなく、値上がりを検出するための比較基準としてのみ存在する
```

```yaml
struct: MergeResult   # 値オブジェクト（Merge の報告。永続化しない）
fields:
  - name: clamped
    type: "[]uuid.UUID"     # 数量合算が maxQuantityPerItem を超えクランプされた productID
  - name: dropped
    type: "[]uuid.UUID"     # 明細数上限により切り捨てられた productID
```

## Cross-field Invariants

- `ownerID` と `sessionToken` は**ちょうど一方だけが非 nil**（排他）。両方 nil は到達不能なカート、
  両方非 nil は token 経由で他人のカートへ到達できる状態であり、どちらも構築を許さない
- 所有者は構築時に決まり、以後変わらない。`ownerID` を後から書き換える操作は存在しない
- `items` 内に同一 `productID` は 2 件以上現れない（自然キー）
- `len(items) <= maxItems`
- `expiresAt > createdAt`

## Behavior Methods

```yaml
- name: SetItem
  signature: SetItem(attrs SetItemAttributes, now time.Time) error   # itemID / productID は同型のため構造体で受ける
  behavior: |
    明細の数量を設定する（PUT /v1/carts/me/items/{productId} の upsert）。同一 productID の明細が既に
    あれば数量を置換し（addedAt は保持）、無ければ追加する。productID が未設定なら
    ErrInvalidProductID（422）。quantity が範囲外なら ErrInvalidQuantity（422）。追加時のみ、itemID が
    未設定なら ErrInvalidID（422）、明細数が maxItems を超えるなら ErrTooManyItems（422）。
    数量 0 は削除ではなくエラーとする — 削除は RemoveItem が担い、1 つの操作に 2 つの意味を持たせない。
    冪等性は自然キー（productID）と「設定」という意味論から来る。同じ要求を 2 回受けても結果は同じで、
    冪等キーの発行を要さない（purchase の作成とはここが異なる）。
  invariants:
    - 同一 productID の明細は常に高々 1 件
    - itemID は新規追加時のみ使われ、置換時は既存の id を保つ

- name: RemoveItem
  signature: RemoveItem(productID uuid.UUID) error
  behavior: |
    指定商品の明細を取り除く（DELETE /v1/carts/me/items/{productId}）。該当明細が無い場合もエラーに
    せず成功を返す（削除の冪等）。呼び出し側が 204 を返すため、「無かった」と「消した」を区別しない。
    ただし productID が未設定なら ErrInvalidProductID（422）。明細の productID は Cart への組み込み時に
    非 nil が保証されるため、nil は「該当明細が無い」ではなく引数そのものが無効な入力であり、冪等の
    対象にならない。

- name: Clear
  signature: Clear()
  behavior: |
    明細をすべて取り除く（DELETE /v1/carts/me、および購入確定後の空化）。空カートは正当な状態のため
    エラーを返さず、カート自体は残る（有効期限も維持される）。

- name: Merge
  signature: Merge(other *Cart, now time.Time) MergeResult
  behavior: |
    別のカート（ゲスト側）の明細を自身（ユーザー側）へ取り込む（ログイン時のマージ）。
      - 同一 productID → 数量を合算し、maxQuantityPerItem を超える場合は上限へクランプして clamped に記録
      - 自身に無い productID → 追加。maxItems を超える分は addedAt の新しい順に切り捨て dropped に記録
        （先に入っていたものを優先して残す）
    **error を返さない。** ログインは認証の成否で決まるべきで、カートの都合で失敗させない。不変条件は
    クランプと切り捨てによって保たれ、失われた分は MergeResult として呼び出し側へ報告される（利用者に
    「一部が入りませんでした」と伝える取得元）。
    other は変更しない（取り込み元の破棄は usecase の責務）。
  invariants:
    - マージ後も同一 productID の重複なし / 数量・明細数の上限を満たす
    - 数量合算の結果が上限を超えても不正状態を作らない（クランプで吸収する）

- name: Touch
  signature: Touch(now time.Time, ttl time.Duration)
  behavior: |
    有効期限を now + ttl へ延長する。参照・更新のいずれの操作でも呼ばれ、使われている間のカートが
    掃除対象にならないようにする。ttl は usecase 層から供給される（ドメインは設定値を持たない）。

- name: MarkSeen
  signature: MarkSeen(prices map[uuid.UUID]money.Price)
  behavior: |
    利用者へ提示した価格を明細へ記録する（カート表示のたびに呼ばれる）。次回表示時の「値上がり」判定は
    この値との比較で行われ、記録が無い明細（初回表示）は値上がりなしと扱う。
    prices に現れない productID の明細は変更しない（非公開化などで価格を引けなかった場合）。
    ここに記録される値は表示の履歴であって約束ではない。請求額を拘束するのは購入明細のスナップショットだけで、
    値上がりの通知は「気づかせる」ためにあり、旧価格での購入を認めるものではない。

- name: IsExpired
  signature: IsExpired(now time.Time) bool
  behavior: |
    有効期限を過ぎているかを返す（「期限切れ」の定義そのもの）。掃除ジョブの削除条件（SQL の WHERE）は
    この述語の実行形であって定義ではない。片方だけを変更してはならない。

- name: IsEmpty
  signature: IsEmpty() bool
  behavior: |
    明細を 1 件も持たないかを返す。購入確定へ進めるかの一次判定に用いる（空カートからの checkout は
    usecase が拒否する）。

- name: IsOwnedBy
  signature: IsOwnedBy(userID uuid.UUID) bool
  behavior: |
    指定ユーザーが所有者かを返す。所有者未確定（ゲスト）のカートは常に false。認可判断そのものは
    usecase 層の Authorizer が担い、本述語はその入力となる所有関係を定義する。

- name: CartItem.Evaluate
  signature: Evaluate(snapshot *ProductSnapshot) Evaluation
  behavior: |
    明細 1 件を商品の観測値と突き合わせ、再評価結果を返す。snapshot が nil は商品を引けなかったことを
    表し、IssueNotFound だけが立つ（在庫も価格も判定材料が無いため）。それ以外では公開状態・在庫・
    価格差をそれぞれ独立に判定し、成立したものを併記する。在庫 0 は「不足」ではなく「無い」であるため
    IssueOutOfStock と IssueInsufficientStock は同時に立たない。lastSeenPrice が無い（初回表示）明細は
    価格差を判定しない。
    結果は表示のための参考情報であり、カートの不変条件を左右しない。拘束力を持たないことは配置の理由に
    ならず、業務上の判断である以上ドメインが持つ。

- name: Cart.Subtotal
  signature: Subtotal(snapshots map[uuid.UUID]ProductSnapshot) (int64, error)
  behavior: |
    自分の明細のうち購入可能なものだけを合算し、決済スケール（USD セント）の整数へ落とす。
    丸めは合算の後に一度だけ行い、明細ごとの丸め誤差が積み上がらないようにする。snapshots に
    含まれない明細と、突き合わせで issue が立った明細は入らない。幅に収まらない場合は
    ErrSubtotalOutOfRange（422）。
    自分の明細集合から決まる問いなので、集約ルート自身が答える（同 § One thing or a set）。
    **提示価格を書き換える前に呼ぶこと。** MarkSeen の後では値上がりが常に「差が無い」になり、
    除外されるはずの明細が入る。
```

## Value Objects

```yaml
- name: SessionToken
  underlying_type: string
  validation: |
    長さが sessionTokenLength に一致し、URL-safe な文字のみで構成されること。
    値の生成（乱数）は usecase 層の境界が行い、ドメインは受け取った値の形式のみを検証する
    （ドメインは乱数へ直接依存しない）。不正な場合は ErrInvalidSessionToken（422）。
  factory: NewSessionToken
  methods:
    - name: Value
      returns: string

- name: ProductSnapshot
  underlying_type: struct    # quantity int / price money.Price / published bool
  validation: |
    検証しない。再評価した時点で観測した商品の値をそのまま運ぶ値であり、正しさは観測元（商品集約）が
    既に保証している。カートはこれを保持せず、判定のたびに受け取って捨てる。
    商品集約を import できないカートが、判定に要る属性だけを値として受け取るための型
    （purchase の LockedProduct と同型）。
  factory: NewProductSnapshot
  methods:
    - name: Price
      returns: money.Price

- name: Evaluation
  underlying_type: struct    # issues []Issue / availableQuantity *int
  validation: |
    検証しない。CartItem.Evaluate の戻り値であり、外から組み立てない。
  methods:
    - name: Issues
      returns: "[]Issue"
    - name: AvailableQuantity
      returns: "*int"
```

`Evaluate` は必ず非 nil の `issues` を返す。`Evaluation` のゼロ値は「まだ突き合わせていない」を
表し、「問題が無い」ではない。合算へ入れるかの判定（パッケージ内の `hasNoIssue`）はこれを区別し、
判らないものを問題無しへ倒さない。

```yaml
enum: Issue                  # 明細を商品の観測値と突き合わせた結果
values:
  - notFound                 # 商品が引けない。単独で立ち、他の issue は併記しない
  - unpublished              # 非公開化された
  - outOfStock               # 在庫 0。insufficientStock とは排他
  - insufficientStock        # 在庫 < 要求数量（AvailableQuantity に上限を添える）
  - priceIncreased           # lastSeenPrice より高い
  - priceDecreased           # lastSeenPrice より安い
```

## Repository Methods

```yaml
- name: FindBySessionToken
  signature: FindBySessionToken(ctx context.Context, token SessionToken) (*Cart, error)
  behavior: |
    セッショントークンからゲストカートを明細込みで取得する。存在しない場合は NotFound。
    所有者確定済みのカートは token を持たないため、本メソッドで引くことはできない。

- name: FindByOwnerID
  signature: FindByOwnerID(ctx context.Context, userID uuid.UUID) (*Cart, error)
  behavior: |
    所有者からカートを明細込みで取得する（GET /v1/carts/me の取得元）。存在しない場合は NotFound。
    ユーザー 1 人につきカートは高々 1 件（carts.user_id の一意制約）。

- name: LockByID
  signature: LockByID(ctx context.Context, id uuid.UUID) (*Cart, error)
  behavior: |
    更新のためにカートを 1 件、悲観ロック（FOR UPDATE）して取得する。存在しない場合は NotFound。

- name: LockByIDs
  signature: LockByIDs(ctx context.Context, ids []uuid.UUID) (Carts, error)
  behavior: |
    更新のためにカート群を id 昇順にまとめてロックして取得する。順序を固定するのは、複数カートを
    同時にロックする処理どうしのデッドロックを構造的に避けるため（ADR-0036
    (ordered-pessimistic-row-locks)）。順序の義務を呼び出し側へ残さないよう、複数件のロックは
    このメソッドだけで行う。不存在の id は結果に現れないため、返る件数は引数より少なくなり得る
    （不存在の検証は呼び出し側の責務）。

- name: Create
  signature: Create(ctx context.Context, c *Cart) error
  behavior: |
    カートを明細込みで新規登録する。user_id / session_token の一意制約違反は Conflict へ正規化する。

- name: CreateOwnerIfAbsent
  signature: CreateOwnerIfAbsent(ctx context.Context, c *Cart) (*Cart, error)
  behavior: |
    所有者のカートが無ければ空のカート（明細なし・session_token は NULL）を作り、確定したカートを返す。
    既にある場合は衝突として扱わず既存のカートを返す点が Create との違い。並行して作成が競合した
    場合も、勝ったほうのカートが返る。存在確認と作成を分けると、その間に他の要求が作った場合に
    一意制約違反でトランザクションごと中断してしまうため、単一文で確保して行を返す。

- name: Update
  signature: Update(ctx context.Context, c *Cart) error
  behavior: |
    カートを渡された ctx の tx 内で現在の状態へ一致させる（差分ではなく集約単位の書き込み）。
    所有者・セッショントークン・有効期限といった親行の状態と、明細の集合の両方が対象。
    明細はカート集約に属する子であり単独では存在しないため、明細単位の Repository は設けない。
    明細は自然キー（productID）で置き換えられ、addedAt は保持される。集合から消えた明細は取り除かれる。
    対象が存在しない場合は NotFound。
    親行と明細の書き込み順序は実装内部の不変条件であり、呼び出し側の義務ではない。

- name: Delete
  signature: Delete(ctx context.Context, id uuid.UUID) error
  behavior: |
    カートを明細ごと削除する（マージ後のゲストカート破棄）。存在しない場合もエラーとしない。

- name: DeleteExpired
  signature: DeleteExpired(ctx context.Context, now time.Time, limit int32) (int, error)
  behavior: |
    有効期限を過ぎたカートを最大 limit 件削除し、削除件数を返す（期限切れ掃除ジョブの実行本体）。
    1 回の呼び出しで消し切ることを意図せず、件数上限で区切って繰り返し呼ばれる前提。
    削除条件は Cart.IsExpired の実行形であり、述語と WHERE の片方だけを変更してはならない。
```

## Notes

- **在庫を押さえない。** カートへの投入は引き当てではなく、在庫の検証は購入成立時に商品行を悲観ロック
  したうえで行われる（[`docs/spec/purchase/domain.md`](../purchase/domain.md)）。カート表示時の在庫判定は
  参考情報であって拘束力を持たない。予約（TTL 付き引き当て）を導入する場合は在庫台帳という別の機構が
  必要になり、本 spec の範囲外。
- **確定単価を持たない。** 表示価格は毎回商品の現在値を引き、確定は購入明細のスナップショットが担う。
  カートに確定単価を持たせると値上げに追随できず「表示と請求が違う」状態を作る。`lastSeenPrice` は
  その例外ではなく、値上がりを検出するための比較基準であって金額の根拠ではない — この 2 つを混同した
  瞬間にカートが価格を約束し始めるため、型と名前で区別する。
- 定数（sample の placeholder。実要件が立った時点で config へ移す）: `minQuantity = 1` /
  `maxQuantityPerItem = 99` / `maxItems = 50` / `sessionTokenLength = 43`（256bit を base64url で表現した長さ）。
- エラー写像: 検証系（`ErrInvalidID` / `ErrInvalidUserID` / `ErrInvalidProductID` /
  `ErrInvalidQuantity` / `ErrTooManyItems` / `ErrInvalidSessionToken` / `ErrSubtotalOutOfRange`）
  → `apperror.ErrValidation`（422）。集約は衝突を表すエラーを持たない。

[ADR-0037 (uuidv7-identifiers)]: ../../adr/0037-uuidv7-identifiers.md
