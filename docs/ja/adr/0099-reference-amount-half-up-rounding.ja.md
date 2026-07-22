---
status: accepted
date: 2026-07-22
deciders: [maintainers]
tags: [architecture]
---

# ADR-0099: referenceAmount は整数演算で計算し、丸めは 1 箇所で half-up する

English canonical: [0099-reference-amount-half-up-rounding.md](../../adr/0099-reference-amount-half-up-rounding.md)

## ステータス

accepted

## 背景

`GET /v1/exchange-rates` は、`displayCurrency` 要求時に `referenceAmount`——base 金額とレートから導く表示通貨（JPY）額——を返す。マネー計算は `float` 誤差を累積してはならないため、`referenceAmount` の内部計算は整数ベースでなければならない。決めるべきは、最終整数の丸め方式と、float→整数の正規化を許す単一地点である（query `amount` は `float64` の倍率であり、*何らかの*正規化点が存在する）。

## 決定

`referenceAmount.amount` フィールド（JPY 円の整数）は、**最終除算で 1 回だけ half-up 丸め**して計算する。計算は `internal/usecase/tools/money.ApplyRateHalfUp(amountMinor int64, rate float64, scale int64) int64` に集約する。この関数はレートを 10^6 固定小数整数へ変換したうえで整数演算のみを行い、呼び出し側は丸めない。

**単一の入力正規化点**は usecase に置く: `amountMinor = int64(math.Round(amount * baseMinorUnitScale))`（USD セントは `baseMinorUnitScale = 100`）を `ApplyRateHalfUp` の前で 1 回だけ行う。この点より下流の reference 計算は `float` マネー値を持たない純整数演算である。

query `amount` は OpenAPI 上 `type: number, format: double` のまま維持する: これは換算の入力倍率であって保存されるマネー値ではない。「float マネー禁止」の適用範囲は**保存値と referenceAmount 計算パスのみ**であり、この入力倍率や汎用の `converted` 出力には及ばない。

half-up を採るのは、`referenceAmount` が参考・非永続の表示額であり会計・税務制約を受けないためである。将来そうした要件が生じた場合は本 ADR を改訂する。

## 結果

### ポジティブな結果

- マネーパス上に `float` 累積がなく、参考額は決定的である。
- 丸めが 1 関数に集約され、後続 API（`POST /v1/purchases` 等）が再利用するため、方針がエンドポイント間でドリフトしない。
- float 例外（query `amount`、`converted`）が明示的かつ限定的である。

### ネガティブな結果

- `amount` は丸め前にセント精度（小数 2 桁、`amount * 100`）へ量子化されるため、base 金額が小数 3 桁以上の有効数字を持つ場合、参考額でサブセント精度が失われる。ただし換算の桁（order of magnitude）は影響を受けない——`×100` と `ApplyRateHalfUp` 内の `÷100` は相殺し、base 通貨の小数指数に関わらず結果は `≈ round(amount * rate)` となる。これは scale の誤りではなく軽微な精度アーティファクトである。
- half-up は銀行丸めと異なり、将来の会計要件では改訂が必要になる。

## 検討した代替案

### query `amount` を API 境界で整数化（USD セント）する

却下: 本エンドポイントは汎用の任意通貨ペア換算デモであり、最小単位整数 query は通貨ごとの指数表（JPY=0 / USD=2 / BHD=3 …）を要する。referenceAmount 整数化に対し過剰装備で、後続 purchases は query ではなく自前の保存 USD セント値を持ち込む。「保存値・reference パスは整数」という非交渉条件は本案なしでも完全に満たせる。

### 銀行丸め（round half to even）

会計中立性のため検討したが、`referenceAmount` は参考・非永続でありバイアス論は当たらない。表示額としては half-up の方が推論が単純である。

## 備考

- ヘルパ: `internal/usecase/tools/money`（`ApplyRateHalfUp`）。
- reference DTO の組み立て: `exchangerate.BuildReferenceAmount`（1 箇所。後続 API が再利用）。
- レートを供給するキャッシュ: [ADR-0098](0098-exchange-rate-cache-gateway-decorator.ja.md)。
