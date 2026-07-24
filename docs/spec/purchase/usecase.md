# Purchase — Usecase Spec

> `POST /v1/purchases`（購入作成）の usecase spec。本リポジトリ初の CommandService（[ADR-0027]）を消費し、
> 在庫減算・購入作成・明細作成・outbox 発行を単一トランザクションで原子的に行う。最外 tx は `idempotency.Run` が所有し、
> 本 usecase は nested（`tx.Manager.Do`）で同一 tx に乗る（[ADR-0029] / [ADR-0100]）。

## Overview

購入作成ユースケースは、認証済みの内部ユーザー ID と購入明細（`productID` + 数量）を受け取り、購入を作成して DTO を返す。CommandService（infra）は「決定済みの書き込みの実行」のみを担い（Repository の write 側対称物）、outbox 発行（`purchase.created.v1`）は usecase の責務（[ADR-0028] の system_cqrs 区分）。`displayCurrency` 指定時のみ合計金額の参考換算額（`referenceAmount`）を tx 外で付与し、為替障害時は null で degrade する。

## Interface

```yaml
package: internal/usecase/purchase
interface: Usecase
methods:
  - name: CreatePurchase
    signature: CreatePurchase(ctx context.Context, params CreatePurchaseParams) (PurchaseView, error)
  - name: GetPurchases   # GET /v1/purchases（購入履歴一覧・cursor）。詳細は「## GET 一覧（購入履歴）」
    signature: GetPurchases(ctx context.Context, userID uuid.UUID, cursor *paging.Cursor) (*PurchaseListView, error)
```

## DTOs

```yaml
input:
  struct: CreatePurchaseParams
  fields:
    - name: UserID           # 認証済みの内部ユーザー ID（#583 IdentityResolver 解決済み）
      type: uuid.UUID
    - name: Details
      type: "[]DetailParam"  # { ProductID uuid.UUID; Quantity int }
    - name: DisplayCurrency
      type: "*string"        # nil の場合は referenceAmount を返さない

output:
  struct: PurchaseView
  fields:
    - name: ID
      type: uuid.UUID
    - name: Code
      type: string
    - name: UserID
      type: uuid.UUID
    - name: StatusID
      type: uuid.UUID
    - name: SubtotalAmount / TaxAmount / ShippingFee / TotalAmount
      type: int              # USD セント整数
    - name: Details
      type: "[]PurchaseDetailView"   # { ProductID, Quantity, UnitPrice }
    - name: OrderedAt
      type: time.Time
    - name: ReferenceAmount
      type: "*ReferenceAmountView"   # { Currency, Amount(int64), Rate, RateDate }。degrade 時 nil
```

## Dependencies

```yaml
- name: tx.Manager                       # nested で最外 idempotency tx に乗る
- name: command.CommandService           # LockProducts / CreatePurchase（infra 実装）
- name: purchase.Repository              # FindByID（書き込み後の再検証・DTO 取得元）
- name: outbox.EmitUsecase               # purchase.created.v1 の emit（同一 tx）
- name: exchangerate.Usecase             # referenceAmount の換算消費（#562 成果 / half-up）
- name: observability.TracerFactory
- name: pkg/uuid                         # id / code / detail id の UUIDv7 採番
```

## Workflow

```yaml
- method: CreatePurchase
  tx_required: true          # nested（最外は idempotency.Run が所有）
  steps:
    - id / code / 各 detail id を UUIDv7 で採番する
    - "txm.Do(nested) 内で:"
    - "  ① cmd.LockProducts(productID 昇順) で在庫をロックし price/quantity を得る"
    - "  ② purchase.New で入力検証・売り越し検証・金額計算・snapshot・未処理ステータスを行う"
    - "  ③ cmd.CreatePurchase で在庫減算 + purchases/purchase_details を書き込む"
    - "  ④ emit.Emit(purchase.created.v1) を同一 tx で発行する（自己完結 snapshot payload）"
    - "  ⑤ repo.FindByID で再検証しレスポンスの取得元とする"
    - tx 外で DisplayCurrency 指定時のみ referenceAmount を付与（xr.Convert / 障害時 nil degrade）
    - PurchaseView へ写像して返す（ドメインエンティティを外へ出さない）
  errors:
    - ErrInsufficientStock → 409（売り越し）
    - ErrEmptyDetails / ErrDuplicateProductID / ErrInvalidQuantity / ErrProductNotFound → 422
```

## GET 一覧（購入履歴・cursor）

`GET /v1/purchases`。認証主体（`userID`）の購入履歴を注文日時降順で cursor（keyset）ページネーション取得する読み取り経路。
一覧は概要のみ（明細を含まない）。ステータス名は購入ステータスマスタとの JOIN で解決する（単一集約 Repository read。
購入ステータスは購入集約に属する固定参照マスタで独立集約ではないため、[ADR-0027] の子参照マスタ例外に該当し QS ではなく Repository で JOIN する）。

```yaml
input:
  - userID: uuid.UUID          # #583 が解決する認証主体の内部ユーザー ID（所有権フィルタ）
  - cursor: "*paging.Cursor"   # first（件数上限）+ after（不透明カーソル）

output:
  struct: PurchaseListView
  fields:
    - name: Items
      type: "[]PurchaseSummaryView"   # { Code string; TotalAmount int(USD セント); Status string(名称); OrderedAt time.Time }
    - name: NextCursor
      type: "*string"                 # 最終ページは nil

cursor:
  boundary: purchaseCursor        # (orderedAt, id) の複合 keyset。usecase 層 private（domain へ持ち込まない）
  keys: [ordered_at(RFC3339Nano), id(UUID)]

dependencies:
  - purchase.Repository            # FindFeedByUserID（所有権フィルタ + 子マスタ JOIN、[]FeedItem を返す）
  - tools/paging                   # Cursor（decode/encode・件数ポリシー）

workflow:
  tx_required: false               # read-only
  steps:
    - cursor を decode し keyset 境界（ordered_at, id）へ解釈（不正は ErrInvalidArgument → 400）
    - repo.FindFeedByUserID(userID, Limit+1) で所有者の購入を注文日時降順に取得
    - 取得件数 > limit なら hasNext=true とし末尾を切り詰め、末尾行から nextCursor を encode
    - PurchaseSummaryView へ写像（他ユーザーの購入は SQL の所有権フィルタで空扱い）
  errors:
    - ErrInvalidArgument → 400（不正 cursor）
    - 未認証は controller で 401（Authn 不在）
```

## Notes

- 冪等スコープは内部 UserID（#581 の確定に追随）。middleware が Scope を設定し、本 usecase 側の固有作業はない。
- `referenceAmount` は非永続・参考表示専用。丸めは half-up（[ADR-0099]）で、ドメインの切り捨て金額とは目的が異なるため規則が分かれる（[ADR-0100]）。

[ADR-0027]: ../../adr/0027-lightweight-cqrs.md
[ADR-0028]: ../../adr/0028-system-cqrs-dml-category.md
[ADR-0029]: ../../adr/0029-commandservice-atomicity-criterion.md
[ADR-0099]: ../../adr/0099-reference-amount-half-up-rounding.md
[ADR-0100]: ../../adr/0100-purchase-stock-lock-and-amount-contract.md
