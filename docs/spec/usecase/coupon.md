# Coupon — Usecase Spec

> クーポンのユースケース spec。**本 spec の時点で、クーポンを能動的に操作する口は存在しない。**
> 発行は廃番ジャーニーの副作用として起き（[`product.md`](product.md) の廃番）、引き換えは従属 issue
> の射程である。ここが記述するのは、その発行が経由する CommandService の契約だけ。

## Overview

クーポンの発行は、廃番のトランザクションの中で CommandService を通して行う。受給者が述語でしか
決まらないため、usecase が集約を組み立てて Repository へ渡す形に分解できない。

**この spec が Interface / DTO / Workflow を持たないのは意図である。** クーポン自体を主語にする
ユースケースがまだ無く、置くと呼び出し側の無い契約になる。引き換えが入った時点でここへ追記する。

## Command Service

```yaml
- name: IssueDiscontinuationCoupons
  package: internal/usecase/product/command
  signature: IssueDiscontinuationCoupons(ctx context.Context, params IssueDiscontinuationCouponsParams) (IssueDiscontinuationCouponsResult, error)
  behavior: |
    params.ProductID の明細を持つカートの所有者のうち退会していないユーザーへ、同一条件のクーポンを
    1 枚ずつ発行する。渡された ctx のトランザクション内で実行する。

    受給者は述語（cart_items への結合と退会の除外）でしか決まらず件数に上限も無いため、呼び出し側が
    集約を組み立てて渡すことはできない。そのため引数は「決まった集約」ではなく発行条件のテンプレートで、
    個々の Coupon はこのメソッドの中で採番される。

    往復は受給者の取得と挿入の 2 回で、発行枚数に比例して増えない。2 文に分かれるのは主キーの採番を
    ドメイン層に置く ADR-0037 の要請による。集合演算が満たすべき性質は往復が母集団に比例しないことで
    あって、文がちょうど 1 つであることではない。
  invariants:
    - ゲストのカート（所有者未確定）は影響を受けるが受給者にならない
    - 退会済みユーザーは受給者にならない
    - 適用範囲は廃番商品のカテゴリで固定される（商品自身を範囲にすると買えない商品にしか使えない）
```

## Query Service

```yaml
- name: EstimateDiscontinueImpact
  package: internal/usecase/product/query
  signature: EstimateDiscontinueImpact(ctx context.Context, productID uuid.UUID) (DiscontinueImpactReadModel, error)
  behavior: |
    商品を廃番にした場合の影響を件数で返す。カート件数・受給対象の利用者数・進行中の購入件数の 3 つ。

    行をロックしない。返した値は返した瞬間から古くなり、実行時の件数と一致する保証はない。押す前に
    規模を見せるための読み取りであり、可否の判定そのものは実行時のトランザクションが持つ。

    各件数の母集団は CommandService 側の書き込みと 1 対 1 で対応する。片方だけを変えると、見積もりと
    実行が食い違って押す前に見せた数字の意味が失われる。
  invariants:
    - AffectedUserCount <= AffectedCartCount（ゲストのカートと退会済みを除くため）
```

## Notes

- 引き換え・失効判定は本 spec の射程外。値引き額の計算（`Discount.Apply`）と適用範囲の判定
  （`Scope.Covers`）も、それを呼ぶ引き換えと一緒に足す。
