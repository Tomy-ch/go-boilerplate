# Coupon — Domain Spec

> クーポン（廃番の代替として一括発行される）のドメイン spec。値引きと適用範囲という**直交する 2 つの
> 値オブジェクト**を持つのが最大の特徴で、1 枚のクーポンは 1 つの値引きと 1 つの適用範囲を持つ。
> 複数枚の併用も、複数の適用範囲の合成も表さない。

## Overview

クーポン集約（Coupon）は、受給者・値引き・適用範囲・有効期限・使用状態を保持するドメインエンティティ。
**受給者は発行時に確定し、以後移らない。** 譲渡を表さないのは、クーポンが特定の利用者に生じた事情への
補償として発行されるためである。

**種別はマスタ表ではなくドメインが閉じた集合として持ち切る。** 「定額か定率か」「全体かカテゴリか商品か」は
業務の語彙であって、行として編集できることに意味がない。`purchase.Status` と同じ形で、業務キーは `code`、
意味はメソッドが持ち、UUID はドメインに焼き込まない。

**値引きと適用範囲を 1 軸の列挙に畳まない。** 畳むと `定率 × カテゴリ限定` のような組み合わせごとに
メンバーが要り、軸を足すたび積で増える。2 つに割れば、それぞれが 1 つの問い（いくら引くか / どの明細が
対象か）に答えるだけで済む。

**値引き額の計算と適用範囲の判定はここに持たない。** どちらも引き換えの関心であり、その振る舞いは
呼び出し側（checkout）と一緒に足す。発行はクーポンを組み立てて保存するだけで、評価しない。

`id` は UUIDv7（[ADR-0037](../../adr/0037-uuidv7-identifiers.md)）で、生成は usecase 層（廃番では
CommandService）が行いドメインへ渡す。有効期限も時刻境界から供給された値を受け取る。

## Entity

```yaml
package: internal/domain/coupon
struct: Coupon
constructors:
  - name: New          # 発行時。生成直後は未使用
  - name: Reconstruct  # 永続化済みの再構築（usedAt を受け取る）
fields:
  - name: id
    type: uuid.UUID
    required: true          # IsNil の場合は ErrInvalidID
  - name: userID
    type: uuid.UUID
    required: true          # 受給者。IsNil の場合は ErrInvalidUserID
  - name: discount
    type: Discount
    required: true          # ゼロ値の場合は ErrInvalidDiscount
  - name: scope
    type: Scope
    required: true          # ゼロ値の場合は ErrInvalidScope
  - name: expiresAt
    type: time.Time
    required: true          # ゼロ値は ErrInvalidExpiresAt
  - name: usedAt
    type: "*time.Time"      # nil 許容（未使用）。使用済みへの遷移は引き換え（#1473）が持つ
  - name: issuedAt
    type: time.Time
    required: true          # ゼロ値は ErrInvalidIssuedAt
```

## Cross-field Invariants

- `expiresAt > issuedAt`（違反は `ErrInvalidExpiresAt`）。発行した時点で既に使えないクーポンは
  発行の意味を持たないため、同時刻も許さない。`New` / `Reconstruct` が共有する検証ゲートで課す。

## Behavior Methods

```yaml
- name: IsUsed
  signature: IsUsed() bool
  behavior: |
    使用済みかどうかを返す。使用日時が設定されていることを指す。
- name: IsExpired
  signature: IsExpired(now time.Time) bool
  behavior: |
    渡された時点で失効しているかを返す。有効期限ちょうどは失効として扱う。
    失効を一括更新する機構は持たず、判定のたびに現在時刻と突き合わせる（カートの期限切れと同じ形）。
    時刻はドメインの外から渡す。
```

## Value Objects

```yaml
- name: DiscountKind
  underlying_type: struct    # code int / name string
  validation: |
    既知の集合（定額 / 定率）だけを許す。永続化されている code からの解決は NewDiscountKind が行い、
    既知でない code は ErrInvalidDiscountKind（永続化状態の破損を再構築時に弾く）。
  factory: NewDiscountKind
  methods:
    - name: Code
      returns: int
    - name: Name
      returns: string
    - name: IsZero
      returns: bool

- name: Discount
  underlying_type: struct    # kind DiscountKind / value decimal.Decimal
  validation: |
    定額は正の金額、定率は 0 より大きく 1 以下。範囲外は ErrInvalidDiscountValue。
    1 を超える率は対象額より多く差し引くことになり、値引きの意味を失うため許さない。
    適用範囲は関知しない。どの明細が対象かは Scope が答える。
  factory: NewFlatDiscount / NewRateDiscount / ReconstructDiscount
  methods:
    - name: Kind
      returns: DiscountKind
    - name: Value
      returns: decimal.Decimal
    - name: IsZero
      returns: bool

- name: ScopeKind
  underlying_type: struct    # code int / name string
  validation: |
    既知の集合（全体 / カテゴリ限定 / 商品限定）だけを許す。扱いは DiscountKind と同じ。
  factory: NewScopeKind
  methods:
    - name: Code
      returns: int
    - name: Name
      returns: string
    - name: IsZero
      returns: bool

- name: Scope
  underlying_type: struct    # kind ScopeKind / targetID *uuid.UUID
  validation: |
    カテゴリ限定・商品限定は対象 ID を必須とし、全体は対象を持ってはならない（ErrInvalidScopeTarget）。
    対象は識別子だけを持ち、商品集約もカテゴリ集約も参照しない
    （集約をまたぐ参照は識別子に限る。internal/domain/README.md の Aggregate Design）。
  factory: NewAllScope / NewCategoryScope / NewProductScope / ReconstructScope
  methods:
    - name: Kind
      returns: ScopeKind
    - name: TargetID
      returns: "*uuid.UUID"   # 防御コピーを返す
    - name: IsZero
      returns: bool
```

## Repository Methods

```yaml
- name: CountByScopeTargetProductID
  signature: CountByScopeTargetProductID(ctx context.Context, productID uuid.UUID) (int, error)
  behavior: |
    指定商品を適用範囲の対象として発行されたクーポンの枚数を返す。
    廃番を再実行したときに、新たな発行を伴わずに実績を返すために用いる。
```

**廃番に伴う一括発行は Repository に持たない。** 発行対象が述語でしか決まらず件数に上限も無いため、
集約を 1 件ずつ構築して書く形に分解できない。その書き込みは CommandService が担う
（判定基準は [ADR-0034](../../adr/0034-commandservice-atomicity-criterion.md)、
実例は [`docs/spec/usecase/product.md`](../usecase/product.md) の廃番）。

## Notes

- **クーポンの行を消す経路は持たない。** 使用済み・失効済みのいずれも行として残す。控えが値引きの
  理由を結合で解決するため、行が消えると金額の説明が付かなくなる。
- 引き換え・失効判定と、それが読む振る舞い（値引き額の計算・適用範囲の判定）は本 spec の射程外。
