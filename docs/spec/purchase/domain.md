# Purchase — Domain Spec

> `POST /v1/purchases`（購入作成・CommandService 正例・原子性/売り越し禁止）のドメイン spec。
> 金額はすべて USD セント単位の整数で保持・計算する（float 不使用）。丸めはドメイン内 1 箇所に集約し切り捨てで統一する（[ADR-0100]）。
> `referenceAmount`（JPY 参考換算）はドメインの関心ではなく usecase 層で half-up 換算する（[ADR-0099]）。

## Overview

購入集約（Purchase）は、購入コード・購入者・初期ステータス・金額（小計 / 税 / 送料 / 合計）と明細（PurchaseDetail）を保持するドメイン集約。生成時に「明細が 1 件以上」「同一 productID の重複なし」「数量が 1 以上」「要求数量がロック済み在庫以下（売り越し禁止）」の不変条件を検証し、違反する `Purchase` は構築できない。

明細の `unitPrice` は在庫ロック取得直後の `products.price` スナップショットであり、購入成立後の価格改定に不変（CommandService 正例の本質＝価格の一貫性）。金額計算（`subtotal = Σ unitPrice × quantity` / `tax = subtotal × taxRate` / `shippingFee = 定数` / `total = subtotal + tax + shippingFee`）はドメイン内で完結し、税・送料の丸めは切り捨てで 1 箇所に集約する。

初期ステータスは「未処理」（`purchase_statuses.code = 1`）。ドメインは code（安定した業務キー）を定数として持ち、`purchase_statuses` の UUID は焼き込まない（seed との二重管理を避けるため、UUID 解決は永続化時の infra 責務）。`id` / `code` は UUIDv7（[ADR-0031]）で、生成は usecase 層が行いドメインへ渡す（ドメインは乱数・時刻に直接依存しない）。

書き込み後のドメイン整合の再検証とレスポンス組み立ての取得元として、`Repository.FindByID` で永続化済みの購入を明細込みで読み出す（[ADR-0027] / [ADR-0029]）。

## Entity

```yaml
package: internal/domain/purchase
struct: Purchase
constructors:
  - name: New            # 新規作成（金額計算・売り越し検証・未処理ステータス）
  - name: Reconstruct    # 永続化済みの再構築（Repository の読み出し / 再検証）
fields:
  - name: id
    type: uuid.UUID
    required: true          # IsNil の場合は ErrInvalidID
  - name: code
    type: string            # UUIDv7 文字列。空の場合は ErrInvalidCode
    required: true
  - name: userID
    type: uuid.UUID
    required: true          # IsNil の場合は ErrInvalidUserID
  - name: statusID
    type: uuid.UUID         # New ではゼロ値（DB が code から解決）。Reconstruct で設定
  - name: subtotalAmount
    type: int               # USD セント。負値は ErrInvalidAmount（Reconstruct）
  - name: taxAmount
    type: int               # USD セント（切り捨て）
  - name: shippingFee
    type: int               # USD セント（固定送料定数）
  - name: totalAmount
    type: int               # USD セント
  - name: details
    type: "[]PurchaseDetail" # 空は ErrEmptyDetails
  - name: orderedAt
    type: time.Time         # New ではゼロ値（DB 既定 NOW()）。Reconstruct で設定
  - name: statusCode
    type: int               # 現在状態の業務キー（status_id を JOIN で解決）。状態機械の source of truth
  - name: paidAt
    type: "*time.Time"      # 支払い日時（未支払いは nil）。支払い後も残る sticky な監査記録
  - name: canceledAt
    type: "*time.Time"      # キャンセル日時（未キャンセルは nil）。監査記録
  - name: shippedAt
    type: "*time.Time"      # 発送日時（未発送は nil）。キャンセル可否の timestamps ガード対象
  - name: deliveredAt
    type: "*time.Time"      # 配達日時（未配達は nil）。キャンセル可否の timestamps ガード対象
```

```yaml
struct: PurchaseDetail   # 値オブジェクト
fields:
  - name: id
    type: uuid.UUID
  - name: productID
    type: uuid.UUID
  - name: quantity
    type: int
  - name: unitPrice
    type: int             # 購入時点の単価スナップショット（USD セント）
```

```yaml
struct: LockedProduct    # 在庫ロック取得直後の商品スナップショット（New の入力）
fields:
  - name: id
    type: uuid.UUID
  - name: price
    type: int
  - name: quantity
    type: int             # ロック時点の在庫数
```

## Cross-field Invariants

- 明細は 1 件以上（空は `ErrEmptyDetails` → 422）。
- 明細内の `productID` は一意（重複は `ErrDuplicateProductID` → 422。在庫行のロック順序固定の前提を守る）。
- 各明細の `quantity >= 1`（違反は `ErrInvalidQuantity` → 422）。
- 各明細の `productID` に対応するロック済み商品が存在すること（欠落は `ErrProductNotFound` → 422）。
- 各明細の `quantity <= ロック済み在庫`（超過は `ErrInsufficientStock` → 409）。
- `unitPrice` は対応するロック済み商品の `price` スナップショット。
- `subtotal = Σ unitPrice × quantity` / `tax = subtotal × taxRatePercent / 100`（切り捨て）/ `total = subtotal + tax + shippingFee`。

## Behavior Methods

```yaml
- name: Cancel
  signature: Cancel(now time.Time) error
  behavior: |
    購入をキャンセル状態へ遷移させる（PATCH /v1/purchases/{purchaseId}/cancel の状態遷移）。
    遷移可否は statusCode の許可遷移表を一次判定とし、status enum に現れないイベント既発生は timestamps ガードで補完する:
      - 既にキャンセル（statusCode == 6 または canceledAt != nil）→ ErrAlreadyCanceled（409）
      - 完了（statusCode == 5）・発送済み（shippedAt != nil）・配達済み（deliveredAt != nil）→ ErrCancelNotAllowed（409）
      - それ以外（未処理 / 受付中 / 確認中 / 処理中 / 支払い済み）→ 許可
    許可時は statusCode をキャンセル（6）へ、canceledAt を now へ同時にセットする（status と timestamp の同時セット・
    既存 timestamps は不変という不変条件）。now は時刻境界（clock）から供給し、ドメインは時刻へ直接依存しない。
  invariants:
    - status（statusCode）と対応 timestamp（canceledAt）は必ず同時セット
    - 既にセット済みの timestamps は不変（NULL → 値 の単調セットのみ）

- name: Pay
  signature: Pay(now time.Time) error
  behavior: |
    購入を支払い済み状態へ遷移させる（PATCH /v1/purchases/{purchaseId}/pay の状態遷移・擬似決済。決済 seam の除外は nextjs-boilerplate ADR-0076）。
    遷移可否は statusCode を一次判定とし、status enum に現れないイベント既発生は timestamps ガードで補完する:
      - 既に支払い済み（statusCode == 7）→ ErrAlreadyPaid（409。二重支払い）
      - キャンセル済み（statusCode == 6）・完了（statusCode == 5）・発送済み（shippedAt != nil）・配達済み（deliveredAt != nil）→ ErrPayNotAllowed（409）
      - それ以外（未処理 / 受付中 / 確認中 / 処理中）→ 許可
    許可時は statusCode を支払い済み（7）へ、paidAt を now へ同時にセットする。now は時刻境界（clock）から供給する。
    決済 SDK / PSP 連携・金額検証は行わず、paidAt とステータスの記録のみを担う。
  invariants:
    - 支払い済み status（statusCode == 7）は paidAt を必須とする（一方向。paidAt は支払い後も残り、以降の遷移で status が変わっても保持されるためキャンセルのような双条件にはしない）

- name: Ship
  signature: Ship(now time.Time) error
  behavior: |
    購入を発送済み状態へ遷移させる（PATCH /v1/purchases/{purchaseId}/ship の状態遷移）。遷移可否は statusCode の等値比較のみで判定する:
      - 既に発送済み（statusCode == 8）→ ErrAlreadyShipped（409。二重発送）
      - 支払い済み（statusCode == 7）→ 許可
      - それ以外（未払い相当 / 完了 / キャンセル済み / 配達済み）→ ErrShipNotAllowed（409）
    許可時は statusCode を発送済み（8）へ、shippedAt を now へ同時にセットする。now は時刻境界（clock）から供給する。
    配送追跡（追跡番号 / 配送業者 / 追跡 URL）は扱わず、shippedAt とステータスの記録のみを担う。
    Cancel / Pay と違い timestamps ガードを併用しないのは、遷移元が支払い済みの 1 状態に限られ statusCode の等値比較だけで
    判別できるため（配達済みは statusCode != 7 として不正遷移側に落ちる）。
  invariants:
    - 発送済み status（statusCode == 8）は shippedAt を必須とする（一方向。shippedAt は発送後も残り、以降の配達済み等の遷移で status が変わっても保持されるためキャンセルのような双条件にはしない）
```

## Value Objects

- `PurchaseDetail`（明細）/ `LockedProduct`（ロック済み在庫スナップショット・New の入力）。上記 Entity 参照。
- `Detail`（購入 1 件の詳細読み取りモデル・read 側）。書き込み集約 Purchase とは別型で、ステータス名（`StatusName`）を
  購入ステータスマスタとの JOIN で解決済み、`PaidAt` は支払い日時（未支払いは nil）、`CanceledAt` はキャンセル日時
  （未キャンセルは nil）、`ShippedAt` は発送日時（未発送は nil）。`FindDetailByID` の返り値。

## Repository Methods

```yaml
- name: FindByID
  signature: FindByID(ctx context.Context, id uuid.UUID) (*Purchase, error)
  behavior: |
    ID から購入を明細込みで取得し Reconstruct で再構築する。存在しない場合は NotFound。
    書き込み後のドメイン整合の再検証とレスポンスの取得元に用いる。
- name: LockByID
  signature: LockByID(ctx context.Context, id uuid.UUID) (*Purchase, error)
  behavior: |
    ID から購入を購入行のみ悲観ロック（SELECT FOR UPDATE OF p）して明細込みで再構築する。存在しない場合は NotFound。
    支払いの状態遷移の競合（同一購入への並行支払い）を購入行ロックで直列化する。擬似決済は単一集約書き込みのため
    CommandService ではなく Repository が担う（[ADR-0029] の判定軸）。
- name: UpdatePaid
  signature: UpdatePaid(ctx context.Context, p *Purchase) error
  behavior: |
    購入の状態更新（status_id は code から解決 / paid_at）を渡された ctx の tx 内で実行する。擬似決済のため
    単一集約（purchases）のみを更新し在庫操作は伴わない。対象行は LockByID で取得・検証済み（遷移可否ガードは付けない）。
- name: UpdateShipped
  signature: UpdateShipped(ctx context.Context, p *Purchase) error
  behavior: |
    購入の状態更新（status_id は code から解決 / shipped_at）を渡された ctx の tx 内で実行する。配送追跡を扱わないため
    単一集約（purchases）のみを更新し在庫操作は伴わない。対象行は LockByID で取得・検証済み（遷移可否ガードは付けない）。
- name: FindFeedByUserID
  signature: FindFeedByUserID(ctx context.Context, userID uuid.UUID, params ListFeedParams) ([]FeedItem, error)
  behavior: |
    指定ユーザーの購入履歴を (ordered_at DESC, id DESC) の安定順で keyset ページネーション取得する
    （GET /v1/purchases 一覧の取得元）。ステータス名は購入ステータスマスタとの JOIN で解決する
    （購入ステータスは購入集約に属する固定参照マスタで、[ADR-0027] の子参照マスタ例外により単一集約の
    Repository read。QS ではない）。params.AfterOrderedAt / AfterID が nil の場合は先頭ページを返す。
    返す FeedItem は書き込み集約 Purchase とは別の読み取りモデル（Code / TotalAmount(USD セント) /
    StatusName / OrderedAt / ID）。不透明カーソルの符号化・復号は usecase 層の責務。
- name: FindDetailByID
  signature: FindDetailByID(ctx context.Context, id uuid.UUID) (*Detail, error)
  behavior: |
    ID から購入詳細（読み取りモデル Detail）を明細込みで取得する。ステータス名は購入ステータスマスタとの
    JOIN で解決する（FindFeedByUserID と同じ子参照マスタ例外）。存在しない場合は NotFound。キャンセル後の
    状態名解決・レスポンスの取得元に用いる（GET 詳細 #569 でも再利用可能）。
```

なお、状態遷移に伴う購入行の悲観ロック（`FOR UPDATE`）は、書き込みが集約を跨ぐかで担い手が分かれる（[ADR-0029] の判定軸）:

- **キャンセル**は在庫復元（`products`）+ 購入更新（`purchases`）の**複数集約への原子的書き込み**のため CommandService（`LockPurchase` / `CancelPurchase`）が担う（[ADR-0027] / [ADR-0029]）。
- **支払い**は `purchases` の status/paid_at のみの**単一集約書き込み**（在庫操作なし）のため、CommandService ではなく **Repository（`LockByID` / `UpdatePaid`）**が担う。行ロックは並行制御（二重支払い防止）であって集約横断の原子性ではない。
- **発送**も `purchases` の status/shipped_at のみの単一集約書き込みのため、支払いと同じく **Repository（`LockByID` / `UpdateShipped`）**が担う。

## Notes

- 定数（[ADR-0100] の placeholder）: `StatusCodeUnprocessed = 1` / `StatusCodeCompleted = 5` / `StatusCodeCanceled = 6` / `StatusCodePaid = 7` / `StatusCodeShipped = 8` / `StatusCodeDelivered = 9` / `taxRatePercent = 10` / `shippingFeeCents = 500`。ステータス UUID は焼き込まず、code から infra で解決する（支払い済み=7 / 発送済み=8 / 配達済み=9 を含め、購入ステータスマスタで定義）。code 値は状態の到達順序を意味しないため、遷移判定は等値比較のみで行う。
- エラー写像: `ErrInsufficientStock` / `ErrAlreadyCanceled` / `ErrCancelNotAllowed` / `ErrAlreadyPaid` / `ErrPayNotAllowed` / `ErrAlreadyShipped` / `ErrShipNotAllowed` → `apperror.ErrConflict`（409）、その他検証系 → `apperror.ErrValidation`（422）。

[ADR-0027]: ../../adr/0027-lightweight-cqrs.md
[ADR-0029]: ../../adr/0029-commandservice-atomicity-criterion.md
[ADR-0031]: ../../adr/0031-uuidv7-identifiers.md
[ADR-0099]: ../../adr/0099-reference-amount-half-up-rounding.md
[ADR-0100]: ../../adr/0100-purchase-stock-lock-and-amount-contract.md
