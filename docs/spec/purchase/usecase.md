# Purchase — Usecase Spec

> `POST /v1/purchases`（購入作成）の usecase spec。本リポジトリ初の CommandService（[ADR-0031 (lightweight-cqrs)]）を消費し、
> 在庫減算・購入作成・明細作成・outbox 発行を単一トランザクションで原子的に行う。最外 tx は `idempotency.Run` が所有し、
> 本 usecase は nested（`tx.Manager.Do`）で同一 tx に乗る（[ADR-0033 (commandservice-atomicity-criterion)]）。

## Overview

購入作成ユースケースは、認証済みの内部ユーザー ID と購入明細（`productID` + 数量）を受け取り、購入を作成して DTO を返す。CommandService（infra）は「決定済みの書き込みの実行」のみを担い（Repository の write 側対称物）、outbox 発行（`purchase.created.v1`）は usecase の責務（[ADR-0032 (system-cqrs-dml-category)] の system_cqrs 区分）。`displayCurrency` 指定時のみ合計金額の参考換算額（`referenceAmount`）を tx 外で付与し、為替障害時は null で degrade する。

## Interface

```yaml
package: internal/usecase/purchase
interface: Usecase
methods:
  - name: CreatePurchase
    signature: CreatePurchase(ctx context.Context, params CreatePurchaseParams) (PurchaseView, error)
  - name: GetPurchases   # GET /v1/purchases（購入履歴一覧・cursor）。詳細は「## GET 一覧（購入履歴）」
    signature: GetPurchases(ctx context.Context, userID uuid.UUID, cursor *paging.Cursor, spec period.Spec) (*PurchaseListView, error)
  - name: GetPurchaseDetail # GET /v1/purchases/{purchaseCode}（購入詳細・集約跨ぎ QS）。詳細は「## GET 詳細（購入詳細）」
    signature: GetPurchaseDetail(ctx context.Context, authn *auth.Authn, purchaseCode string) (PurchaseGetDetailView, error)
  - name: CancelPurchase # PATCH /v1/purchases/{purchaseCode}/cancel。詳細は「## PATCH キャンセル」
    signature: CancelPurchase(ctx context.Context, params CancelPurchaseParams) (CancelPurchaseView, error)
  - name: PayPurchase   # PATCH /v1/purchases/{purchaseCode}/pay。詳細は「## PATCH 支払い」
    signature: PayPurchase(ctx context.Context, params PayPurchaseParams) (PayPurchaseView, error)
  - name: ShipPurchase  # PATCH /v1/purchases/{purchaseCode}/ship（admin のみ）。詳細は「## PATCH 発送」
    signature: ShipPurchase(ctx context.Context, authn *auth.Authn, purchaseCode string) (ShipPurchaseView, error)
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
- name: user.LockRepository              # LockShareByID（購入者の共有ロック取得。退会との直列化。[ADR-0035 (ordered-pessimistic-row-locks)]）
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
    - "  ⓪ userLock.LockShareByID で購入者を共有ロック付きで読み出し、membership.EnsurePurchasable で在籍を判定する（退会と直列化。[ADR-0035 (ordered-pessimistic-row-locks)]）"
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
避ける（順序固定・取得位置・ロックモードの規律は [ADR-0035 (ordered-pessimistic-row-locks)]）。

購入者の在籍判定は、退会（`DELETE /v1/users/{userId}`）の「進行中の購入が残っていれば拒否」と
**対になる 1 つの業務ルール**である。片方だけを読んでも全体は分からないため、もう一方は
[`docs/spec/user/usecase.md`](../user/usecase.md) の `DeleteUser` に記述してある。購入側は共有ロックで
在籍を観測し、退会側は同じ行を排他ロックで押さえる。共有ロック同士は衝突しないため、同一ユーザーの
並行購入は互いに直列化されず、退会とだけ直列化される。退会済みユーザーによる購入は 409（`ErrConflict`）で、
退会側の拒否と同じステータスに揃えてある。

### ガードを置かない集約横断条件（商品の非公開化）

在籍ガードと同じ形をしていながら、**意図的にガードを置かない**条件が 1 つある。商品の非公開化
（`PATCH /v1/products/{productId}` で `publishedAt` を null クリア）と、その商品を参照する進行中の購入である。

購入は作成時に単価スナップショット（`purchase_details.unit_price`）を記録し、以後のライフサイクル
（支払い・発送・配達・キャンセル）は商品の公開状態を一度も読まない。金額の権威は購入側のスナップショットに
あり、購入詳細の取得も `products` へは商品名の解決のためだけに結合する。したがって商品が非公開になっても
成立済みの購入が誤りになることはなく、この条件は判定後に陳腐化してよい。商品更新は購入行をロックせず、
進行中の購入の有無も見ない（商品側は自集約の楽観ロックだけで並行編集から守る）。

これは在籍ガードの対照例であり、[ADR-0033 (commandservice-atomicity-criterion)] の判定手順でいえば在籍ガードが分岐 2（同期ロック）、
こちらが分岐 1（既定の分解）に当たる。同じ「一方の集約の書き込みと他方の集約の状態」という形から
答えが分かれるのは、条件が陳腐化したときに業務上の誤りが生じるかどうかが違うためである。

要件が変われば分類も変わる。「非公開になった商品は以後発送してはならない」が要件として立てば、
その判定は発送時点の公開状態を読むことになり、判定と commit の間に非公開化が入り得るため分岐 2 へ移り、
商品行の共有ロックが必要になる。現時点でそれは要件ではない。

## GET 一覧（購入履歴・cursor）

`GET /v1/purchases`。認証主体（`userID`）の購入履歴を注文日時降順で cursor（keyset）ページネーション取得する読み取り経路。
一覧は概要のみ（明細の配列は含まない）だが、行を見分けるための要約 2 項目（先頭商品名・明細の点数）を持つ。
先頭商品名は独立集約である商品（`products`）との結合で解決するため、この経路は子参照マスタ例外に収まらず
**QueryService**（`internal/usecase/purchase/query`）に置く（[ADR-0031 (lightweight-cqrs)]。購入詳細と同じ扱い）。
ステータスの業務キーと名称は購入ステータスマスタとの JOIN で解決する。ページを先に閉じてから要約を LATERAL で
結合するため、1 クエリで解決し N+1 にならない。

```yaml
input:
  - userID: uuid.UUID          # #583 が解決する認証主体の内部ユーザー ID（所有権フィルタ）
  - cursor: "*paging.Cursor"   # first（件数上限）+ after（不透明カーソル）
  - spec: period.Spec          # 注文日時の対象期間（all / month / range / recent）。ゼロ値は全期間

output:
  struct: PurchaseListView
  fields:
    - name: Items
      type: "[]PurchaseSummaryView"   # { Code string; TotalAmount int(USD セント); StatusID/StatusCode/StatusName; FirstItemName string; ItemCount int(明細の行数); OrderedAt time.Time }
    - name: NextCursor
      type: "*string"                 # 最終ページは nil

cursor:
  boundary: purchaseCursor        # (orderedAt, id) の複合 keyset。usecase 層 private（読み取りモデルは id を持つが応答へは出さない）
  keys: [ordered_at(RFC3339Nano), id(UUID)]

dependencies:
  - query.PurchaseFeedQueryService  # FindFeedByUserID（所有権フィルタ + 期間絞り込み + ステータス / 商品の結合、[]PurchaseFeedReadModel を返す）
  - purchase/period                # Spec の暦日境界への解決（clock + *time.Location に依存）
  - tools/paging                   # Cursor（decode/encode・件数ポリシー）

workflow:
  tx_required: false               # read-only
  steps:
    - cursor を decode し keyset 境界（ordered_at, id）へ解釈（不正は ErrInvalidArgument → 400）
    - period.Resolve(spec, clock.Now(), loc) で対象期間を暦日へ解決し、半開区間 [after, before) を params へ載せる
    - feedQS.FindFeedByUserID(userID, Limit+1) で所有者の購入を注文日時降順に取得（要約 2 項目込み）
    - 取得件数 > limit なら hasNext=true とし末尾を切り詰め、末尾行から nextCursor を encode
    - PurchaseSummaryView へ写像（他ユーザーの購入は SQL の所有権フィルタで空扱い）
  errors:
    - ErrInvalidArgument → 400（不正 cursor / 区分ごとの必須指定の欠落 / to < from）
    - 未認証は controller で 401（Authn 不在）
```

期間の絞り込みは keyset ページネーションと直交する。絞り込み条件はページ間で不変である前提で、
呼び出し側はページ送りの間も同じ期間を渡す（条件が変われば keyset 境界の意味も変わるため連続性は保証しない）。
一覧は対象期間をレスポンスに含めない（相対指定の解決結果を必要とするのは集計側だけで、一覧はカーソルが継続の
責務を負っているため）。

## GET 詳細（購入詳細・集約跨ぎ QS）

`GET /v1/purchases/{purchaseCode}`。本人の購入 1 件を明細（商品名込み）とともに取得する読み取り経路。購入（`purchases` / `purchase_details`）と
商品（`products`）は独立集約であり、明細に商品名を含む集約跨ぎの read 投影のため **QueryService**（`internal/usecase/purchase/query`）で取得する
（[ADR-0031 (lightweight-cqrs)]。子参照マスタへの JOIN で済む cancel/pay/ship/deliver の `purchase.Detail`（Repository read）とは経路を分ける）。商品名は `products` との
JOIN でサーバー解決した現在名（live・非スナップショット）、ステータス名は購入ステータスマスタとの JOIN で解決する。所有権は QS 本体クエリの
`WHERE p.code = @code AND p.user_id = @user_id` で担保し、他人の購入・不存在はいずれも 0 行 → NotFound（404）で存在を秘匿する（403 は用いない）。
固定 2 クエリ（本体 + 明細 JOIN products）で N+1 を避ける。書き込みを伴わないため tx / authorizer は不要。

```yaml
input:
  - authn: "*auth.Authn"       # 認証主体。nil は Unauthenticated（401）。UserID() を QS の所有権述語へ渡す
  - purchaseCode: string       # 公開識別子。内部 UUID は受け取らない

output:
  struct: PurchaseGetDetailView
  fields:
    - name: Code / UserID
      type: string / uuid.UUID
    - name: StatusID / StatusCode / StatusName   # 購入ステータスマスタで解決済み
      type: uuid.UUID / int / string
    - name: SubtotalAmount / TaxAmount / ShippingFee / TotalAmount
      type: int64                   # USD セント整数
    - name: Details
      type: "[]PurchaseDetailItemView"   # { ProductID, ProductName(products JOIN の現在名), Quantity, UnitPrice(価格スケール decimal) }
    - name: OrderedAt
      type: time.Time
    - name: PaidAt / CanceledAt
      type: "*time.Time"            # 未確定なら nil

dependencies:
  - query.PurchaseDetailQueryService   # FindDetailByUserAndCode（集約跨ぎ read 投影・所有権 SQL 述語・0 行は NotFound）
  - observability.TracerFactory

workflow:
  tx_required: false               # read-only
  steps:
    - authn == nil なら ErrUnauthenticated（401）
    - authn.UserID() を取得（未解決はエラー伝播）
    - qs.FindDetailByUserAndCode(userID, purchaseCode) で本体 + 明細（商品名 JOIN）を取得
    - PurchaseGetDetailView へ写像して返す（読み取りモデルを外へ出さない）
  errors:
    - ErrNotFound → 404（不存在・他人の購入の存在秘匿）
    - ErrUnauthenticated → 401（Authn 不在）
```

## GET 集計（購入サマリ・me）

`GET /v1/users/me/purchases/summary`。認証主体自身の購入の件数・支払金額・明細金額・ステータス別内訳と、
要求されたグループ化単位の内訳を返す集計読み取り経路（マイページの集計カード用。一覧・明細は返さない）。
`COUNT` / `SUM` / `GROUP BY` の結果は購入集約を再構成できない
**派生投影**であり、[ADR-0031 (lightweight-cqrs)] が Repository から集計を明示除外しているため **QueryService**（`internal/usecase/purchase/query`）に置く
（ステータス名解決だけで済む一覧の Repository read とは経路を分ける。集計は購入集約側に置き、user 配下には置かない）。
配置判断は ADR-0031 (lightweight-cqrs) + `docs/rules.md` § Repository / QueryService Rules で既に固定化されているため**新規 ADR は発行しない**。
所有権は QS の `WHERE p.user_id = @user_id` で担保し、パスに他者識別子を持たないため所有権チェックは不要。書き込みを伴わないため tx は不要。

ユースケースは `internal/usecase/purchase/purchase_usecase.go`（tx / CommandService / outbox を持つ書き込み中心の集約）ではなく、
読み取り専用の別パッケージ `internal/usecase/purchase/summary` に置く（`product/ranking` と同じ集計 read の形。
`purchase.PurchaseSummaryView` は一覧要素の DTO 名として既に使われており、集計 DTO と名前空間が衝突する点も理由）。

```yaml
input:
  - authn: "*auth.Authn"       # 認証主体。nil は Unauthenticated（401）。UserID() を QS の所有権述語へ渡す
  - params: GetSummaryParams   # { Period period.Spec; GroupBy []GroupKind }

output:
  struct: SummaryView          # package summary
  fields:
    - name: Period
      type: period.Window      # 集計に実際に用いた対象期間（解決済みの暦日）。全期間なら絞り込みなしを表す
    - name: TotalCount / TotalAmount
      type: int64              # 購入件数 / 支払金額（小計 + 税額 + 送料、USD セント整数）。対象 0 件でも 0 を返しエラーにしない
    - name: ItemsTotal
      type: decimal.Decimal    # 明細金額（単価 × 数量）の合計。価格スケールの正確な decimal（丸めない）
    - name: StatusBreakdown
      type: "[]StatusCountView"  # { StatusID, StatusCode, StatusName, Count, TotalAmount }。対象期間に出現したステータスのみ・マスタ表示順（sort_key 昇順）
    - name: Groups
      type: "map[string]GroupNodeView"  # GroupBy 指定時のみ。{ Name; ItemsTotal; Groups }（再帰）。未指定なら nil

dependencies:
  - query.PurchaseSummaryQueryService   # SummarizeByUserID / SumItemsByUserID / SummarizeItemsByProductByUserID
  - purchase/period                     # Spec の暦日境界への解決（clock + *time.Location に依存）
  - observability.TracerFactory

workflow:
  tx_required: false               # read-only
  steps:
    - authn == nil なら ErrUnauthenticated（401）
    - authn.UserID() を取得（未解決はエラー伝播）
    - period.Resolve(params.Period, clock.Now(), loc) で対象期間を暦日へ解決（レスポンスにも載せるため usecase で確定させる）
    - GroupBy を検証（未知の単位・同一単位の重複は ErrInvalidArgument → 400）
    - qs.SummarizeByUserID(userID, window) でステータス別の件数・金額を 1 クエリで取得
    - GroupBy が空なら qs.SumItemsByUserID で明細金額の合計だけを取得（商品単位の行は読まない）
    - GroupBy があれば qs.SummarizeItemsByProductByUserID の行を GroupBy の順に入れ子へ畳み込み、同じ行から合計も導く
    - ステータス別の集計値を購入件数・支払金額へ畳み込み SummaryView へ写像（総計と内訳が同一スナップショットで整合する）
  errors:
    - ErrUnauthenticated → 401（Authn 不在）
    - ErrInvalidArgument → 400（区分ごとの必須指定の欠落 / to < from / 不正な GroupBy）
```

**キャンセル済みの購入はすべての集計値から除外する。** ステータス別内訳も同じ母集団に揃えるため、内訳に
キャンセルのステータスは現れない。母集団を 1 つに保つことで `totalCount = Σ statusBreakdown.count` が常に成立する。
代償として「内訳からキャンセル分を差し引く」使い方はできなくなるが、期間を指定した集計の主用途は支出額の把握であり、
成立しなかった取引を含む合計は支出として読めない（会計・決済 API の慣行どおり、キャンセルを負値で表現することもしない。
金額は非負のまま状態で区別する）。

**金額は 2 つの尺度で返す。** `TotalAmount` は購入の支払金額（小計 + 税額 + 送料）の合計で決済スケールの整数セント、
`ItemsTotal` は明細金額（単価 × 数量）の合計で価格スケールの正確な decimal。税額・送料は明細へ按分できないため
両者は一致しない。`ItemsTotal` を決済スケールへ丸めないのは、[ADR-0037 (two-scale-quantity-model)] が丸めを
「正確な量が決済確定値になる 1 箇所」に限っているためで、参照系の集計はその 1 箇所ではない。丸めをグループごとに
行えば `Σ groups ≠ ItemsTotal` となり、内訳が合計を説明しなくなる。

**グループ化のキーは単位ごとに異なる。** カテゴリは `product_categories.name` が UNIQUE なので名称をそのままキーにでき、
商品は `products.name` に一意制約が無いため ID をキーにする（同名の別商品を 1 グループへ畳み込まないため）。
どちらのノードも表示名は `Name` で返すので、クライアントはキーの形を意識せず表示できる。

**ユースケースの `GroupNodeView` は再帰型だが、wire の応答スキーマは階層 2 段を別の型で表す**
（`PurchaseGroupResponse` → `PurchaseSubGroupResponse`）。グループ化単位の上限が 2 である以上、
再帰は契約として過剰なうえ、自己参照する component は `oapi-codegen` の tag 絞り込みで刈り取れず、
購入と無関係な全ハンドラの生成物へ同じ型が複製される（実測 20 パッケージ超）。単位を 3 つに増やすときは
スキーマを 1 段足す。

## PATCH キャンセル (purchase cancel)

`PATCH /v1/purchases/{purchaseCode}/cancel`。本人の購入をキャンセルする状態遷移経路。状態機械の source of truth は
`status_id`（現在状態）で、timestamps（`canceled_at` / `shipped_at` / `delivered_at`）はイベント発生の監査記録として併用する。
在庫復元は `POST /v1/purchases` の在庫減算と対称な同一 tx 強整合で、[ADR-0031 (lightweight-cqrs)] の CommandService に対称実装する（原子性方式は [ADR-0033 (commandservice-atomicity-criterion)]）。
キャンセル後の状態名解決は詳細読み取りモデル（`purchase.Detail`、GET 詳細で再利用可能な Repository read）で JOIN 解決する。

```yaml
input:
  struct: CancelPurchaseParams
  fields:
    - name: PurchaseCode         # キャンセル対象の購入コード（公開識別子）
      type: uuid.UUID
    - name: UserID               # 認証済みの内部ユーザー ID（所有権検証）
      type: uuid.UUID

output:
  struct: CancelPurchaseView
  fields:
    - name: Code / UserID
      type: string / uuid.UUID
    - name: StatusID / StatusCode / StatusName   # 購入ステータスマスタで解決済み（キャンセル）
      type: uuid.UUID / int / string
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
  - tx.Manager                      # 最外 tx（本経路は Idempotency-Key 冪等化を配線しない）
  - command.CommandService          # LockPurchase（FOR UPDATE）/ CancelPurchase（在庫加算 + status/canceled_at 更新）
  - purchase.Repository             # FindDetailByID（書き込み後の状態名解決・DTO 取得元）
  - outbox.EmitUsecase              # purchase.canceled.v1 の emit（同一 tx）

workflow:
  tx_required: true
  steps:
    - "txm.Do 内で:"
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

`PATCH /v1/purchases/{purchaseCode}/pay`。本人の購入を支払い済みへ遷移させる状態遷移経路（擬似決済）。
決済 SDK / PSP 連携・金額検証は行わず、`paid_at` のセットと `status_id` の「支払い済み」への更新のみを担う。在庫操作は伴わない。
**単一集約（`purchases`）のみを更新するため、複数集約の原子性を要する CommandService（[ADR-0033 (commandservice-atomicity-criterion)]）ではなく通常 usecase + Repository で完結する**
（cancel は在庫復元を伴う複数集約書き込みのため CommandService を用いる。判定軸は「集約を跨ぐ書き込みの原子性が要るか」）。
状態機械の source of truth はキャンセルと統一で `status_id`（現在状態）、timestamps は監査記録として併用する。二重支払いは
購入行ロック（`repo.LockByCode` の FOR UPDATE・並行制御であって集約横断ではない）+ ドメインの状態チェック（`ErrAlreadyPaid`）で防ぐ。
支払い後の状態名解決は詳細読み取りモデル（`purchase.Detail`）で JOIN 解決する。

```yaml
input:
  struct: PayPurchaseParams
  fields:
    - name: PurchaseCode         # 支払い対象の購入コード（公開識別子）
      type: uuid.UUID
    - name: UserID               # 認証済みの内部ユーザー ID（所有権検証）
      type: uuid.UUID

output:
  struct: PayPurchaseView
  fields:
    - name: Code / UserID
      type: string / uuid.UUID
    - name: StatusID / StatusCode / StatusName   # 購入ステータスマスタで解決済み（支払い済み）
      type: uuid.UUID / int / string
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
  - purchase.Repository             # LockByCode（FOR UPDATE）/ UpdatePaid（status/paid_at 更新・単一集約）/ FindDetailByID（状態名解決・DTO 取得元）
  - outbox.EmitUsecase              # purchase.paid.v1 の emit（同一 tx）

workflow:
  tx_required: true
  steps:
    - "txm.Do 内で:"
    - "  ① repo.LockByCode で購入行を FOR UPDATE ロックし明細込みで再構築（並行支払いを直列化）"
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

`PATCH /v1/purchases/{purchaseCode}/ship`。購入を発送済みへ遷移させる状態遷移経路。`shipped_at` のセットと `status_id` の
「発送済み」への更新のみを担い、配送追跡（追跡番号 / 配送業者 / 追跡 URL）と在庫操作は伴わない。支払いと同じく
**単一集約（`purchases`）のみを更新するため CommandService ではなく Repository で完結する**（[ADR-0033 (commandservice-atomicity-criterion)] の判定軸）。
二重発送は購入行ロック（`repo.LockByCode` の FOR UPDATE）+ ドメインの状態チェック（`ErrAlreadyShipped`）で防ぐ。

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
    - name: purchaseCode         # 発送対象の購入コード（公開識別子）
      type: uuid.UUID

output:
  struct: ShipPurchaseView
  fields:
    - name: Code / UserID
      type: string / uuid.UUID
    - name: StatusID / StatusCode / StatusName   # 購入ステータスマスタで解決済み（発送済み）
      type: uuid.UUID / int / string
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
  - purchase.Repository             # LockByCode（FOR UPDATE）/ UpdateShipped（status/shipped_at 更新・単一集約）/ FindDetailByID（状態名解決・DTO 取得元）
  - outbox.EmitUsecase              # purchase.shipped.v1 の emit（同一 tx）

workflow:
  tx_required: true
  steps:
    - "① authn が nil なら ErrUnauthenticated（401）"
    - "② authorizer.Authorize(ActionPurchaseShip, Resource{kind: purchase, ownerID: nil}) で認可（tx の外）"
    - "txm.Do 内で:"
    - "  ③ repo.LockByCode で購入行を FOR UPDATE ロックし明細込みで再構築（並行発送を直列化）"
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

`PATCH /v1/purchases/{purchaseCode}/deliver`。購入を配達済みへ遷移させる状態遷移経路。`delivered_at` のセットと `status_id` の
「配達済み」への更新のみを担い、配達確認の証跡（署名 / 受領写真 / GPS 位置）と在庫操作は伴わない。発送と同じく
**単一集約（`purchases`）のみを更新するため CommandService ではなく Repository で完結する**（[ADR-0033 (commandservice-atomicity-criterion)] の判定軸）。
二重配達は購入行ロック（`repo.LockByCode` の FOR UPDATE）+ ドメインの状態チェック（`ErrAlreadyDelivered`）で防ぐ。

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
    - name: purchaseCode         # 配達完了対象の購入コード（公開識別子）
      type: uuid.UUID

output:
  struct: DeliverPurchaseView
  fields:
    - name: Code / UserID
      type: string / uuid.UUID
    - name: StatusID / StatusCode / StatusName   # 購入ステータスマスタで解決済み（配達済み）
      type: uuid.UUID / int / string
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
  - purchase.Repository             # LockByCode（FOR UPDATE）/ UpdateDelivered（status/delivered_at 更新・単一集約）/ FindDetailByID（状態名解決・DTO 取得元）
  - outbox.EmitUsecase              # purchase.delivered.v1 の emit（同一 tx）

workflow:
  tx_required: true
  steps:
    - "① authn が nil なら ErrUnauthenticated（401）"
    - "② authorizer.Authorize(ActionPurchaseDeliver, Resource{kind: purchase, ownerID: nil}) で認可（tx の外）"
    - "txm.Do 内で:"
    - "  ③ repo.LockByCode で購入行を FOR UPDATE ロックし明細込みで再構築（並行配達を直列化）"
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

## GET 発送待ち一覧（まとめ発送・admin）

`GET /v1/purchases/shippable`。発送可能な購入を、まとめて発送してよい組に分けて返す admin 向けの読み取り経路。
「何を発送すべきか」を admin に浮かび上がらせる一覧で、発送の実行そのものは購入 1 件ずつ
`PATCH /v1/purchases/{purchaseCode}/ship` が担う。cursor ページングを持たない top-N で、`GET /v1/products/low-stock` と同型。

まとめ判定は**単一集約に閉じたドメインサービス** `domain/service/dispatch` が担う。「この購入は発送可能か」は 1 件の
状態だけで決まるため `Purchase.IsShippable`（エンティティのメソッド）だが、「これらのうちどれとどれを 1 便にまとめて
よいか」は集合についての問いで、どの `Purchase` 1 件のメソッドにもなり得ない
（[`internal/domain/README.md`](../../../internal/domain/README.md) § One thing or a set）。

```yaml
input:
  - authn: "*auth.Authn"                   # admin 認可の主体。nil は ErrUnauthenticated
  - params: ListShippablePurchasesParams   # { Limit int }（1 未満は既定 20 / 100 超は 100 へクランプ）

output:
  struct: PurchaseShippableListView
  fields:
    - name: Groups
      type: "[]DispatchGroupView"   # { UserID uuid.UUID; Purchases []ShippablePurchaseView }
      # ShippablePurchaseView = { ID uuid.UUID; Code string; TotalAmount int(USD セント); OrderedAt time.Time }

dependencies:
  - authz.Authorizer               # ActionPurchaseListShippable / 所有者なしリソース（admin のみ）
  - purchase.Repository            # FindShippable（発送可能の絞り込み・明細込み）
  - domain/service/dispatch        # GroupForDispatch（まとめ判定。単一集約 Domain Service）
  - tools/paging                   # LimitPolicy（既定 20 / 上限 100）

workflow:
  tx_required: false               # read-only
  steps:
    - authn が nil なら ErrUnauthenticated（401）
    - authorizer.Authorize(ActionPurchaseListShippable, Resource{Kind "purchase", Owner nil}) で admin 認可
    - limit を LimitPolicy で正規化し repo.FindShippable(limit) で発送可能な購入を古い順に取得
    - 返却行を Purchase.IsShippable で検証し、該当しない行があれば ErrInternal（SQL と述語の乖離を表に出す。
      internal/usecase/README.md § Verifying infrastructure against the domain）
    - dispatch.GroupForDispatch(purchases) で購入者ごとの組へ分ける
    - DispatchGroupView / ShippablePurchaseView へ写像（組の順序・組内の順序はドメインサービスの結果を保つ）
  errors:
    - ErrUnauthenticated → 401（Authn 不在）
    - ErrForbidden → 403（非 admin）
    - ErrInternal → 500（発送可能でない行が混じっていた場合＝SQL と述語の乖離）
```

`limit` は**読み出す購入の件数**であり、まとめ判定はその範囲の中で行う。範囲の外にある同一購入者の購入は別の便になる。
組は算出結果であり永続化しないため、組そのものの識別子は返さない。

## Notes

- 冪等スコープは内部 UserID（#581 の確定に追随）。middleware が Scope を設定し、本 usecase 側の固有作業はない。
- `referenceAmount` は非永続・参考表示専用。丸めは half-up で、決済額（切り捨て）とは目的が異なるため規則が分かれる
  （決済額は課金される権威的な値、`referenceAmount` は表示のみの参考値）。丸め方式と最小単位桁数は方式そのものが
  policy であり、汎用の decimal 機構には焼き込まない（[ADR-0037 (two-scale-quantity-model)]）。換算側の仕様は
  [`docs/spec/exchange-rate/usecase.md`](../exchange-rate/usecase.md)。
- 購入集計（`GET /v1/users/me/purchases/summary`）の `WHERE user_id = $1` は、購入履歴一覧用の複合インデックス
  `purchases (user_id, ordered_at DESC, id DESC)`（migration 000012）の先頭列で解決できるため、集計専用のインデックス追加は不要。

[ADR-0031 (lightweight-cqrs)]: ../../adr/0031-lightweight-cqrs.md
[ADR-0032 (system-cqrs-dml-category)]: ../../adr/0032-system-cqrs-dml-category.md
[ADR-0033 (commandservice-atomicity-criterion)]: ../../adr/0033-commandservice-atomicity-criterion.md
[ADR-0035 (ordered-pessimistic-row-locks)]: ../../adr/0035-ordered-pessimistic-row-locks.md
[ADR-0037 (two-scale-quantity-model)]: ../../adr/0037-two-scale-quantity-model.md
