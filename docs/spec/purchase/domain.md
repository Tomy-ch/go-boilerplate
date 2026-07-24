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

## Value Objects

- `PurchaseDetail`（明細）/ `LockedProduct`（ロック済み在庫スナップショット・New の入力）。上記 Entity 参照。

## Repository Methods

```yaml
- name: FindByID
  signature: FindByID(ctx context.Context, id uuid.UUID) (*Purchase, error)
  behavior: |
    ID から購入を明細込みで取得し Reconstruct で再構築する。存在しない場合は NotFound。
    書き込み後のドメイン整合の再検証とレスポンスの取得元に用いる。
- name: FindFeedByUserID
  signature: FindFeedByUserID(ctx context.Context, userID uuid.UUID, params ListFeedParams) ([]FeedItem, error)
  behavior: |
    指定ユーザーの購入履歴を (ordered_at DESC, id DESC) の安定順で keyset ページネーション取得する
    （GET /v1/purchases 一覧の取得元）。ステータス名は購入ステータスマスタとの JOIN で解決する
    （購入ステータスは購入集約に属する固定参照マスタで、[ADR-0027] の子参照マスタ例外により単一集約の
    Repository read。QS ではない）。params.AfterOrderedAt / AfterID が nil の場合は先頭ページを返す。
    返す FeedItem は書き込み集約 Purchase とは別の読み取りモデル（Code / TotalAmount(USD セント) /
    StatusName / OrderedAt / ID）。不透明カーソルの符号化・復号は usecase 層の責務。
```

## Notes

- 定数（[ADR-0100] の placeholder）: `StatusCodeUnprocessed = 1` / `taxRatePercent = 10` / `shippingFeeCents = 500`。
- エラー写像: `ErrInsufficientStock` → `apperror.ErrConflict`（409）、その他検証系 → `apperror.ErrValidation`（422）。

[ADR-0027]: ../../adr/0027-lightweight-cqrs.md
[ADR-0029]: ../../adr/0029-commandservice-atomicity-criterion.md
[ADR-0031]: ../../adr/0031-uuidv7-identifiers.md
[ADR-0099]: ../../adr/0099-reference-amount-half-up-rounding.md
[ADR-0100]: ../../adr/0100-purchase-stock-lock-and-amount-contract.md
