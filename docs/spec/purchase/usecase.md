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
  - name: GetPurchaseDetail # GET /v1/purchases/{purchaseId}（購入詳細・集約跨ぎ QS）。詳細は「## GET 詳細（購入詳細）」
    signature: GetPurchaseDetail(ctx context.Context, authn *auth.Authn, purchaseID uuid.UUID) (PurchaseGetDetailView, error)
  - name: CancelPurchase # PATCH /v1/purchases/{purchaseId}/cancel。詳細は「## PATCH キャンセル」
    signature: CancelPurchase(ctx context.Context, params CancelPurchaseParams) (CancelPurchaseView, error)
  - name: PayPurchase   # PATCH /v1/purchases/{purchaseId}/pay。詳細は「## PATCH 支払い」
    signature: PayPurchase(ctx context.Context, params PayPurchaseParams) (PayPurchaseView, error)
  - name: ShipPurchase  # PATCH /v1/purchases/{purchaseId}/ship（admin のみ）。詳細は「## PATCH 発送」
    signature: ShipPurchase(ctx context.Context, authn *auth.Authn, purchaseID uuid.UUID) (ShipPurchaseView, error)
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
- name: command.CommandService           # CreatePurchase（infra 実装）
- name: purchase.Repository              # FindByID（書き込み後の再検証・DTO 取得元）
- name: product.Repository               # LockByIDs（在庫行の悲観ロック）
- name: user.LockRepository              # LockShareByID（購入者の共有ロック取得。ADR-0107 / withdrawal-purchase-row-lock-serialization）
- name: domain/service/membership        # EnsurePurchasable（在籍の判定）
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
    - "  ⓪ userLock.LockShareByID で購入者を共有ロック付きで読み出し、membership.EnsurePurchasable で在籍を判定する（退会と直列化。ADR-0107 / withdrawal-purchase-row-lock-serialization）"
    - "  ① productRepo.LockByIDs(productID 昇順) で在庫行をロックし price/quantity を得る"
    - "  ② purchase.New で入力検証・売り越し検証・金額計算・snapshot・未処理ステータスを行う"
    - "  ③ cmd.CreatePurchase で在庫減算 + purchases/purchase_details を書き込む"
    - "  ④ emit.Emit(purchase.created.v1) を同一 tx で発行する（自己完結 snapshot payload）"
    - "  ⑤ repo.FindByID で再検証しレスポンスの取得元とする"
    - tx 外で DisplayCurrency 指定時のみ referenceAmount を付与（xr.Convert / 障害時 nil degrade）
    - PurchaseView へ写像して返す（ドメインエンティティを外へ出さない）
  errors:
    - ErrInsufficientStock → 409（売り越し）
    - 在籍ガードの NotFound → 409（受付から成立までの間に退会が確定した競合時のみ。退会確定済みの主体は useridentity.Resolver が 401 で弾くためここへ到達しない。主体の状態と操作の衝突であり購入対象の不存在ではないため 404 へは畳まない）
    - 在籍ガードのその他のエラーはそのまま伝播（障害を退会済みと区別できなくしないため 409 へ化けさせない）
    - ErrEmptyDetails / ErrDuplicateProductID / ErrInvalidQuantity / ErrProductNotFound → 422
```

ロックはユーザー行 → 商品行（id 昇順）の順で取る。全 tx で順序を固定することでデッドロックを構造的に
避ける（商品行の順序固定は [ADR-0100]、ユーザー行を先頭に置く根拠は [ADR-0107]）。

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

## GET 詳細（購入詳細・集約跨ぎ QS）

`GET /v1/purchases/{purchaseId}`。本人の購入 1 件を明細（商品名込み）とともに取得する読み取り経路。購入（`purchases` / `purchase_details`）と
商品（`products`）は独立集約であり、明細に商品名を含む集約跨ぎの read 投影のため **QueryService**（`internal/usecase/purchase/query`）で取得する
（[ADR-0027]。子参照マスタへの JOIN で済む一覧・cancel/pay の `purchase.Detail`（Repository read）とは経路を分ける）。商品名は `products` との
JOIN でサーバー解決した現在名（live・非スナップショット）、ステータス名は購入ステータスマスタとの JOIN で解決する。所有権は QS 本体クエリの
`WHERE p.id = @id AND p.user_id = @user_id` で担保し、他人の購入・不存在はいずれも 0 行 → NotFound（404）で存在を秘匿する（403 は用いない）。
固定 2 クエリ（本体 + 明細 JOIN products）で N+1 を避ける。書き込みを伴わないため tx / authorizer は不要。

```yaml
input:
  - authn: "*auth.Authn"       # 認証主体。nil は Unauthenticated（401）。UserID() を QS の所有権述語へ渡す
  - purchaseID: uuid.UUID

output:
  struct: PurchaseGetDetailView
  fields:
    - name: ID / Code / UserID
      type: uuid.UUID / string / uuid.UUID
    - name: StatusID / StatusName   # 購入ステータスマスタで解決済み
      type: uuid.UUID / string
    - name: SubtotalAmount / TaxAmount / ShippingFee / TotalAmount
      type: int64                   # USD セント整数
    - name: Details
      type: "[]PurchaseDetailItemView"   # { ProductID, ProductName(products JOIN の現在名), Quantity, UnitPrice(価格スケール decimal) }
    - name: OrderedAt
      type: time.Time
    - name: PaidAt / CanceledAt
      type: "*time.Time"            # 未確定なら nil

dependencies:
  - query.PurchaseDetailQueryService   # FindDetailByUserAndID（集約跨ぎ read 投影・所有権 SQL 述語・0 行は NotFound）
  - observability.TracerFactory

workflow:
  tx_required: false               # read-only
  steps:
    - authn == nil なら ErrUnauthenticated（401）
    - authn.UserID() を取得（未解決はエラー伝播）
    - qs.FindDetailByUserAndID(userID, purchaseID) で本体 + 明細（商品名 JOIN）を取得
    - PurchaseGetDetailView へ写像して返す（読み取りモデルを外へ出さない）
  errors:
    - ErrNotFound → 404（不存在・他人の購入の存在秘匿）
    - ErrUnauthenticated → 401（Authn 不在）
```

## GET 集計（購入サマリ・me）

`GET /v1/users/me/purchases/summary`。認証主体自身の購入の総件数・合計金額・ステータス別内訳を返す集計読み取り経路
（マイページの集計カード用。一覧・明細は返さない）。`COUNT` / `SUM` / `GROUP BY` の結果は購入集約を再構成できない
**派生投影**であり、[ADR-0027] が Repository から集計を明示除外しているため **QueryService**（`internal/usecase/purchase/query`）に置く
（ステータス名解決だけで済む一覧の Repository read とは経路を分ける。集計は購入集約側に置き、user 配下には置かない）。
配置判断は ADR-0027 + `docs/rules.md` § Repository / QueryService Rules で既に固定化されているため**新規 ADR は発行しない**。
所有権は QS の `WHERE p.user_id = @user_id` で担保し、パスに他者識別子を持たないため所有権チェックは不要。書き込みを伴わないため tx は不要。

ユースケースは `internal/usecase/purchase/purchase_usecase.go`（tx / CommandService / outbox を持つ書き込み中心の集約）ではなく、
読み取り専用の別パッケージ `internal/usecase/purchase/summary` に置く（`product/ranking` と同じ集計 read の形。
`purchase.PurchaseSummaryView` は一覧要素の DTO 名として既に使われており、集計 DTO と名前空間が衝突する点も理由）。

```yaml
input:
  - authn: "*auth.Authn"       # 認証主体。nil は Unauthenticated（401）。UserID() を QS の所有権述語へ渡す

output:
  struct: SummaryView          # package summary
  fields:
    - name: TotalCount / TotalAmount
      type: int64              # 総件数 / 合計金額（USD セント整数）。購入 0 件でも 0 を返しエラーにしない
    - name: StatusBreakdown
      type: "[]StatusCountView"  # { StatusID, StatusName, Count, TotalAmount }。購入に出現したステータスのみ・マスタ表示順（sort_key 昇順）

dependencies:
  - query.PurchaseSummaryQueryService   # SummarizeByUserID（ステータス単位の COUNT / SUM・所有権 SQL 述語）
  - observability.TracerFactory

workflow:
  tx_required: false               # read-only
  steps:
    - authn == nil なら ErrUnauthenticated（401）
    - authn.UserID() を取得（未解決はエラー伝播）
    - qs.SummarizeByUserID(userID) でステータス別の件数・金額を 1 クエリで取得
    - ステータス別の集計値を総件数・合計金額へ畳み込み SummaryView へ写像（総計と内訳が同一スナップショットで整合する）
  errors:
    - ErrUnauthenticated → 401（Authn 不在）
```

キャンセル済みの購入も総件数・合計金額に含める。キャンセルはステータス別内訳の 1 要素として件数・金額とも返るため、
キャンセルを除いた値が必要なクライアントは内訳から差し引ける。総計から除外する設計は、内訳の母数と総計が食い違ううえ、
「何を純額とするか」という業務方針を API に焼き込むことになるため採らない（会計・決済 API の慣行どおり、
キャンセルを負値で表現することもしない。金額は非負のまま状態で区別する）。

## PATCH キャンセル (purchase cancel)

`PATCH /v1/purchases/{purchaseId}/cancel`。本人の購入をキャンセルする状態遷移経路。状態機械の source of truth は
`status_id`（現在状態）で、timestamps（`canceled_at` / `shipped_at` / `delivered_at`）はイベント発生の監査記録として併用する。
在庫復元は `POST /v1/purchases` の在庫減算と対称な同一 tx 強整合で、[ADR-0027] の CommandService に対称実装する（原子性方式は [ADR-0029]）。
キャンセル後の状態名解決は詳細読み取りモデル（`purchase.Detail`、GET 詳細で再利用可能な Repository read）で JOIN 解決する。

```yaml
input:
  struct: CancelPurchaseParams
  fields:
    - name: PurchaseID           # キャンセル対象の購入 ID
      type: uuid.UUID
    - name: UserID               # 認証済みの内部ユーザー ID（所有権検証）
      type: uuid.UUID

output:
  struct: CancelPurchaseView
  fields:
    - name: ID / Code / UserID
      type: uuid.UUID / string / uuid.UUID
    - name: StatusID / StatusName   # 購入ステータスマスタで解決済み（キャンセル）
      type: uuid.UUID / string
    - name: SubtotalAmount / TaxAmount / ShippingFee / TotalAmount
      type: int                     # USD セント整数
    - name: Details
      type: "[]PurchaseDetailView"
    - name: OrderedAt
      type: time.Time
    - name: CanceledAt
      type: "*time.Time"            # キャンセル日時

dependencies:
  - clock.Clock                     # Cancel(now) へ供給する時刻境界（ドメインの時刻直依存を避ける）
  - tx.Manager                      # nested で最外 idempotency tx に乗る
  - command.CommandService          # LockPurchase（FOR UPDATE）/ CancelPurchase（在庫加算 + status/canceled_at 更新）
  - purchase.Repository             # FindDetailByID（書き込み後の状態名解決・DTO 取得元）
  - outbox.EmitUsecase              # purchase.canceled.v1 の emit（同一 tx）

workflow:
  tx_required: true                 # nested（最外は idempotency.Run が所有）
  steps:
    - "txm.Do(nested) 内で:"
    - "  ① cmd.LockPurchase で購入行を FOR UPDATE ロックし明細込みで再構築（並行キャンセルを直列化）"
    - "  ② purchase.UserID() != params.UserID なら NotFound へ畳む（存在秘匿）"
    - "  ③ purchase.Cancel(now) で遷移可否検証 + status/canceled_at を同時更新（ドメイン不変条件）"
    - "  ④ cmd.CancelPurchase で明細分の在庫加算 + purchases の status_id/canceled_at 更新"
    - "  ⑤ emit.Emit(purchase.canceled.v1) を同一 tx で発行する"
    - "  ⑥ repo.FindDetailByID で状態名を解決しレスポンスの取得元とする"
    - CancelPurchaseView へ写像して返す（ドメインエンティティを外へ出さない）
  errors:
    - ErrAlreadyCanceled → 409（二重キャンセル）
    - ErrCancelNotAllowed → 409（完了・発送済み・配達済みからの不正遷移）
    - ErrNotFound → 404（不存在・他人の購入の存在秘匿）
    - 未認証は controller で 401（Authn 不在）
```

## PATCH 支払い (purchase pay)

`PATCH /v1/purchases/{purchaseId}/pay`。本人の購入を支払い済みへ遷移させる状態遷移経路（擬似決済。決済 seam の除外は nextjs-boilerplate ADR-0076）。
決済 SDK / PSP 連携・金額検証は行わず、`paid_at` のセットと `status_id` の「支払い済み」への更新のみを担う。在庫操作は伴わない。
**単一集約（`purchases`）のみを更新するため、複数集約の原子性を要する CommandService（[ADR-0029]）ではなく通常 usecase + Repository で完結する**
（cancel は在庫復元を伴う複数集約書き込みのため CommandService を用いる。判定軸は「集約を跨ぐ書き込みの原子性が要るか」）。
状態機械の source of truth はキャンセルと統一で `status_id`（現在状態）、timestamps は監査記録として併用する。二重支払いは
購入行ロック（`repo.LockByID` の FOR UPDATE・並行制御であって集約横断ではない）+ ドメインの状態チェック（`ErrAlreadyPaid`）で防ぐ。
支払い後の状態名解決は詳細読み取りモデル（`purchase.Detail`）で JOIN 解決する。

```yaml
input:
  struct: PayPurchaseParams
  fields:
    - name: PurchaseID           # 支払い対象の購入 ID
      type: uuid.UUID
    - name: UserID               # 認証済みの内部ユーザー ID（所有権検証）
      type: uuid.UUID

output:
  struct: PayPurchaseView
  fields:
    - name: ID / Code / UserID
      type: uuid.UUID / string / uuid.UUID
    - name: StatusID / StatusName   # 購入ステータスマスタで解決済み（支払い済み）
      type: uuid.UUID / string
    - name: SubtotalAmount / TaxAmount / ShippingFee / TotalAmount
      type: int                     # USD セント整数
    - name: Details
      type: "[]PurchaseDetailView"
    - name: OrderedAt
      type: time.Time
    - name: PaidAt
      type: "*time.Time"            # 支払い日時

dependencies:
  - clock.Clock                     # Pay(now) へ供給する時刻境界（ドメインの時刻直依存を避ける）
  - tx.Manager                      # 最外 tx（本経路は idempotency を配線しない）
  - purchase.Repository             # LockByID（FOR UPDATE）/ UpdatePaid（status/paid_at 更新・単一集約）/ FindDetailByID（状態名解決・DTO 取得元）
  - outbox.EmitUsecase              # purchase.paid.v1 の emit（同一 tx）

workflow:
  tx_required: true
  steps:
    - "txm.Do 内で:"
    - "  ① repo.LockByID で購入行を FOR UPDATE ロックし明細込みで再構築（並行支払いを直列化）"
    - "  ② purchase.UserID() != params.UserID なら NotFound へ畳む（存在秘匿）"
    - "  ③ purchase.Pay(now) で遷移可否検証 + status/paid_at を同時更新（ドメイン不変条件）"
    - "  ④ repo.UpdatePaid で purchases の status_id/paid_at を更新（単一集約・在庫操作なし）"
    - "  ⑤ emit.Emit(purchase.paid.v1) を同一 tx で発行する"
    - "  ⑥ repo.FindDetailByID で状態名を解決しレスポンスの取得元とする"
    - PayPurchaseView へ写像して返す（ドメインエンティティを外へ出さない）
  errors:
    - ErrAlreadyPaid → 409（二重支払い）
    - ErrPayNotAllowed → 409（キャンセル済み・完了・発送済み・配達済みからの不正遷移）
    - ErrNotFound → 404（不存在・他人の購入の存在秘匿）
    - 未認証は controller で 401（Authn 不在）
```

## PATCH 発送 (purchase ship)

`PATCH /v1/purchases/{purchaseId}/ship`。購入を発送済みへ遷移させる状態遷移経路。`shipped_at` のセットと `status_id` の
「発送済み」への更新のみを担い、配送追跡（追跡番号 / 配送業者 / 追跡 URL）と在庫操作は伴わない。支払いと同じく
**単一集約（`purchases`）のみを更新するため CommandService ではなく Repository で完結する**（[ADR-0029] の判定軸）。
二重発送は購入行ロック（`repo.LockByID` の FOR UPDATE）+ ドメインの状態チェック（`ErrAlreadyShipped`）で防ぐ。

支払い・キャンセルと異なり **admin 専用の運用操作**であり、認可の扱いが 3 点で異なる:

- **所有権を問わない**（admin は任意の購入を発送できる）。よって `UserID` を入力に取らず、`Authn` をそのまま受けて認可へ渡す。
- **認可は Authorizer へ委譲する**。usecase は role を検査せず、`ActionPurchaseShip` と所有者なしリソース
  （`authz.NewResource("purchase", nil)`）を宣言するのみで、「admin なら許可 / 非 admin は所有者一致時のみ許可」というポリシーは
  Authorizer（`internal/infrastructure/authz/userrole`）が持つ。`ownerID` を nil にすると所有者フォールバックが働かず、実質 admin 限定になる。
- **不存在を 404 で秘匿しない**（非 admin は認可で先に 403 となり購入の存在を知り得ないため）。

```yaml
input:
  args:
    - name: authn                # 認証主体（認可の判定材料。usecase は role を検査せず Authorizer へ委譲）
      type: "*auth.Authn"
    - name: purchaseID           # 発送対象の購入 ID
      type: uuid.UUID

output:
  struct: ShipPurchaseView
  fields:
    - name: ID / Code / UserID
      type: uuid.UUID / string / uuid.UUID
    - name: StatusID / StatusName   # 購入ステータスマスタで解決済み（発送済み）
      type: uuid.UUID / string
    - name: SubtotalAmount / TaxAmount / ShippingFee / TotalAmount
      type: int                     # USD セント整数
    - name: Details
      type: "[]PurchaseDetailView"
    - name: OrderedAt
      type: time.Time
    - name: ShippedAt
      type: "*time.Time"            # 発送日時

dependencies:
  - authz.Authorizer                # ActionPurchaseShip の認可（admin 判定は内部 DB role が SoT）
  - clock.Clock                     # Ship(now) へ供給する時刻境界（ドメインの時刻直依存を避ける）
  - tx.Manager                      # 最外 tx（本経路は idempotency を配線しない）
  - purchase.Repository             # LockByID（FOR UPDATE）/ UpdateShipped（status/shipped_at 更新・単一集約）/ FindDetailByID（状態名解決・DTO 取得元）
  - outbox.EmitUsecase              # purchase.shipped.v1 の emit（同一 tx）

workflow:
  tx_required: true
  steps:
    - "① authn が nil なら ErrUnauthenticated（401）"
    - "② authorizer.Authorize(ActionPurchaseShip, Resource{kind: purchase, ownerID: nil}) で認可（tx の外）"
    - "txm.Do 内で:"
    - "  ③ repo.LockByID で購入行を FOR UPDATE ロックし明細込みで再構築（並行発送を直列化）"
    - "  ④ purchase.Ship(now) で遷移可否検証 + status/shipped_at を同時更新（ドメイン不変条件）"
    - "  ⑤ repo.UpdateShipped で purchases の status_id/shipped_at を更新（単一集約・在庫操作なし）"
    - "  ⑥ emit.Emit(purchase.shipped.v1) を同一 tx で発行する"
    - "  ⑦ repo.FindDetailByID で状態名を解決しレスポンスの取得元とする"
    - ShipPurchaseView へ写像して返す（ドメインエンティティを外へ出さない）
  errors:
    - ErrForbidden → 403（非 admin）
    - ErrAlreadyShipped → 409（二重発送）
    - ErrShipNotAllowed → 409（未払い相当・完了・キャンセル済み・配達済みからの不正遷移）
    - ErrNotFound → 404（不存在。所有権による秘匿はしない）
    - 未認証は controller で 401（Authn 不在）
```

## PATCH 配達完了 (purchase deliver)

`PATCH /v1/purchases/{purchaseId}/deliver`。購入を配達済みへ遷移させる状態遷移経路。`delivered_at` のセットと `status_id` の
「配達済み」への更新のみを担い、配達確認の証跡（署名 / 受領写真 / GPS 位置）と在庫操作は伴わない。発送と同じく
**単一集約（`purchases`）のみを更新するため CommandService ではなく Repository で完結する**（[ADR-0029] の判定軸）。
二重配達は購入行ロック（`repo.LockByID` の FOR UPDATE）+ ドメインの状態チェック（`ErrAlreadyDelivered`）で防ぐ。

発送と同じ **admin 専用の運用操作**であり、認可の扱いも同じ 3 点で支払い・キャンセルと異なる:

- **所有権を問わない**（admin は任意の購入を配達済みにできる）。よって `UserID` を入力に取らず、`Authn` をそのまま受けて認可へ渡す。
- **認可は Authorizer へ委譲する**。usecase は role を検査せず、`ActionPurchaseDeliver` と所有者なしリソース
  （`authz.NewResource("purchase", nil)`）を宣言するのみで、ポリシーは Authorizer（`internal/infrastructure/authz/userrole`）が持つ。
- **不存在を 404 で秘匿しない**（非 admin は認可で先に 403 となり購入の存在を知り得ないため）。

```yaml
input:
  args:
    - name: authn                # 認証主体（認可の判定材料。usecase は role を検査せず Authorizer へ委譲）
      type: "*auth.Authn"
    - name: purchaseID           # 配達完了対象の購入 ID
      type: uuid.UUID

output:
  struct: DeliverPurchaseView
  fields:
    - name: ID / Code / UserID
      type: uuid.UUID / string / uuid.UUID
    - name: StatusID / StatusName   # 購入ステータスマスタで解決済み（配達済み）
      type: uuid.UUID / string
    - name: SubtotalAmount / TaxAmount / ShippingFee / TotalAmount
      type: int                     # USD セント整数
    - name: Details
      type: "[]PurchaseDetailView"
    - name: OrderedAt
      type: time.Time
    - name: DeliveredAt
      type: "*time.Time"            # 配達日時

dependencies:
  - authz.Authorizer                # ActionPurchaseDeliver の認可（admin 判定は内部 DB role が SoT）
  - clock.Clock                     # Deliver(now) へ供給する時刻境界（ドメインの時刻直依存を避ける）
  - tx.Manager                      # 最外 tx（本経路は idempotency を配線しない）
  - purchase.Repository             # LockByID（FOR UPDATE）/ UpdateDelivered（status/delivered_at 更新・単一集約）/ FindDetailByID（状態名解決・DTO 取得元）
  - outbox.EmitUsecase              # purchase.delivered.v1 の emit（同一 tx）

workflow:
  tx_required: true
  steps:
    - "① authn が nil なら ErrUnauthenticated（401）"
    - "② authorizer.Authorize(ActionPurchaseDeliver, Resource{kind: purchase, ownerID: nil}) で認可（tx の外）"
    - "txm.Do 内で:"
    - "  ③ repo.LockByID で購入行を FOR UPDATE ロックし明細込みで再構築（並行配達を直列化）"
    - "  ④ purchase.Deliver(now) で遷移可否検証 + status/delivered_at を同時更新（ドメイン不変条件）"
    - "  ⑤ repo.UpdateDelivered で purchases の status_id/delivered_at を更新（単一集約・在庫操作なし）"
    - "  ⑥ emit.Emit(purchase.delivered.v1) を同一 tx で発行する"
    - "  ⑦ repo.FindDetailByID で状態名を解決しレスポンスの取得元とする"
    - DeliverPurchaseView へ写像して返す（ドメインエンティティを外へ出さない）
  errors:
    - ErrForbidden → 403（非 admin）
    - ErrAlreadyDelivered → 409（二重配達）
    - ErrDeliverNotAllowed → 409（未払い相当・支払い済み・完了・キャンセル済みからの不正遷移）
    - ErrNotFound → 404（不存在。所有権による秘匿はしない）
    - 未認証は controller で 401（Authn 不在）
```

## Notes

- 冪等スコープは内部 UserID（#581 の確定に追随）。middleware が Scope を設定し、本 usecase 側の固有作業はない。
- `referenceAmount` は非永続・参考表示専用。丸めは half-up（[ADR-0099]）で、ドメインの切り捨て金額とは目的が異なるため規則が分かれる（[ADR-0100]）。
- 購入集計（`GET /v1/users/me/purchases/summary`）の `WHERE user_id = $1` は、購入履歴一覧用の複合インデックス
  `purchases (user_id, ordered_at DESC, id DESC)`（migration 000012）の先頭列で解決できるため、集計専用のインデックス追加は不要。

[ADR-0027]: ../../adr/0027-lightweight-cqrs.md
[ADR-0028]: ../../adr/0028-system-cqrs-dml-category.md
[ADR-0029]: ../../adr/0029-commandservice-atomicity-criterion.md
[ADR-0099]: ../../adr/0099-reference-amount-half-up-rounding.md
[ADR-0100]: ../../adr/0100-purchase-stock-lock-and-amount-contract.md
[ADR-0107]: ../../adr/0107-withdrawal-purchase-row-lock-serialization.md
