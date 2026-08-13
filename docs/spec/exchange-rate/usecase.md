# Exchange Rate — Usecase Spec

> exchange-rate は外部の為替レート配信サービスを参照するだけの非永続 API であり、固有の domain
> エンティティ・テーブルを持たない。そのため lean-a-spec（[ADR-0092 (lean-a-spec-scaffold)]）の「原則 domain + usecase」から
> domain.md を省略し、本 usecase.md のみで表現する。外部レート取得への意味的 gateway は
> `internal/usecase/boundary/exchangerate.Gateway`（boundary IF）として Dependencies 節に明示する。

## Overview

為替レート換算ユースケース（`GET /v1/exchange-rates`）は、基軸通貨 `base` から `quote` への現在レートを
外部サービス（gateway 経由）で取得し、`original` を換算した `converted` を返す thin orchestrator。
`displayCurrency` が指定されたときのみ、`base → displayCurrency` のレートを追加取得して
**参考換算額（`referenceAmount`）**を添える。外部レスポンスの型・エラーは gateway 内で完結しており、
本 usecase は boundary DTO（`exchangerate.Rate`）のみを扱う（外部型の内層非漏洩）。

本 API の実証目的は「外部依存を持つ read 経路の、キャッシュ・degrade・正確な十進の扱い」であり、
address（外部不通でも `200 + 空候補 + IsFallback: true` へ倒す）とは degrade の方針が**正反対**である。
応答は 3 状態に分離する:

1. 主レート（`base → quote`）の取得失敗 → **error を返す**（gateway が `apperror.ErrUnavailable` を返し、
   usecase・handler はそのまま伝播して 503）。`converted` は本 API の存在理由そのものであり、これを欠いた
   200 は「成功した」と嘘をつくことになるため degrade しない。
2. 主レートは取得できたが `displayCurrency` のレートが取得できない → `converted` は返し
   `referenceAmount: null`（200）。degrade したことは `Warn` ログに残す
   （`reference amount degraded: display rate unavailable`）。
3. `referenceAmount` の丸め結果が `int64` を超える → 同じく `referenceAmount: null`（200）＋ `Warn` ログ
   （`reference amount degraded: amount exceeds settlement range`）。

`referenceAmount` は**参考表示専用の非永続な値**である。決済されるわけではないので会計・税務の制約を
受けず、丸めは半額切り上げ（half-up / half-away-from-zero）を採る。銀行家丸めも検討したが、偏りの議論は
繰り返し集計される権威的な金額に対するものであり、1 回きりの表示値には当てはまらない。丸めは
`money.ApplyRateHalfUp` の **1 箇所だけ**で行い、呼び出し側は丸めない（後続の購入 API も同じ関数を使うため、
方針がエンドポイント間でドリフトしない）。

表示通貨の最小単位桁数（`minorUnitDigits`）は **usecase が持つ policy** であり、汎用の decimal 機構
（`pkg/decimal.ToScaledInt64`）には焼き込まない（[ADR-0036 (two-scale-quantity-model)]）。現状の表示通貨は JPY のみで
`displayMinorUnitDigits = 0`（1 円 = 小数 0 桁）。JPY はあくまで**参考表示**として現れるだけで、決済通貨は
USD のみである（決済スケールの仕様は [`docs/spec/purchase/domain.md`](../purchase/domain.md)）。多通貨決済は
スコープ外。

`original` / `converted` / `rate` はいずれもワイヤ上で**十進文字列**として扱う。JSON number は一般的な
パーサが IEEE754 double として復号するため、そこで精度が落ちる（[ADR-0036 (two-scale-quantity-model)]）。

## Interface

```yaml
package: internal/usecase/exchangerate
name: Usecase
methods:
  - name: Convert
    signature: Convert(ctx context.Context, in ConvertInput) (*ConvertResult, error)
```

## DTOs

```yaml
- name: ConvertInput
  description: 換算入力。DisplayCurrency 指定時のみ referenceAmount を組み立てる。
  fields:
    - name: Base
      type: string
    - name: Quote
      type: string
    - name: Amount
      type: decimal.Decimal      # pkg/decimal。float64 は使わない
    - name: DisplayCurrency
      type: "*string"            # nil なら referenceAmount を返さない
- name: ConvertResult
  description: 換算結果の usecase 出力 DTO。
  fields:
    - name: Converted
      type: decimal.Decimal      # Amount × rate（丸めなし・正確な十進）
    - name: Reference
      type: "*ReferenceAmount"   # degrade 時 nil
- name: ReferenceAmount
  description: 表示通貨での参考換算額。最小単位の整数（JPY なら円）で保持する。
  fields:
    - name: Currency
      type: string
    - name: Amount
      type: int64                # 最小単位整数。half-up で 1 回だけ丸めた結果
    - name: Rate
      type: decimal.Decimal      # base -> displayCurrency のレート
    - name: RateDate
      type: string               # 例 "2026-07-21"。外部応答に日付が無ければ空文字
```

## Dependencies

```yaml
- tracer                  # observability.TracerFactory -> LayerTracer
- exchangerate_gateway    # internal/usecase/boundary/exchangerate.Gateway（外部レート取得。外部型は gateway 内で消える）
- logging                 # logging.Logger（referenceAmount の degrade を Warn で残す）
```

`money.ApplyRateHalfUp`（`internal/usecase/tools/money`）は依存注入ではなく純関数として呼ぶ。

## Workflow

### Convert

```yaml
tx_required: false
steps:
  - exchangerate_gateway.GetRate(base, quote) で主レートを取得する
  - 取得に失敗した場合は degrade せず error を伝播する（状態1。ErrUnavailable -> 503）
  - Converted = Amount × rate.Value を正確な十進で計算する（ここでは丸めない）
  - DisplayCurrency が nil ならここで ConvertResult{Converted} を返す
  - exchangerate_gateway.GetRate(base, displayCurrency) で表示用レートを取得する
  - 取得に失敗した場合は Warn ログを出し Reference: nil で返す（状態2。error は返さない）
  - BuildReferenceAmount で money.ApplyRateHalfUp(amount, rate, displayMinorUnitDigits) を呼び最小単位整数へ丸める
  - int64 を超える場合は Warn ログを出し Reference: nil で返す（状態3）
calls:
  - exchangerate_gateway.GetRate
  - money.ApplyRateHalfUp
  - logging.Warn
errors:
  - 主レート取得の失敗は伝播（503）
  - 表示用レート取得の失敗・オーバーフローは referenceAmount のみを degrade（200 + null）
```

## Caching

レート配信元（Frankfurter / ECB 由来）は 1 日 1 回しか更新されないため、リクエストごとに外部 HTTP を
叩くのは無駄であり、リクエストのレイテンシを外部依存に結びつけてしまう。キャッシュは
**`boundary.Gateway` を満たす TTL decorator**（`internal/infrastructure/webapi/exchangerate/cache.go`）として
infra 層に置き、DI で生の gateway を包む。usecase・domain はキャッシュの存在を知らない。汎用の Cache 抽象を
作らず既存の継ぎ目を decorate するという判断そのものは [ADR-0104 (no-generic-cache-abstraction)]。

- **TTL は `rateTTL = 24h` の定数**であり、config 値にはしない。環境ごとに変える理由を述べられないため。
  配信元が日次更新であることが 24h の根拠である。
- ECB の公表時刻（およそ 16:00 CET）をまたぐと最大でおよそ 24 時間古いレートを返し得る。これを許容するのは、
  この値が**非永続の参考表示**にしか使われないためであり、応答に `rateDate` を含めることで呼び出し側が
  鮮度を自分で判断できるようにしている。
- **エラーはキャッシュしない。** 取得に失敗した場合は次のリクエストで再び配信元を叩くので、一過性の障害が
  キャッシュを汚染しない。
- キャッシュ件数は `maxCacheEntries = 512` で上限を設ける。未認証エンドポイントで `base` / `quote` が
  自由入力であるため、無制限の map はメモリ枯渇の攻撃面になる。上限到達時は期限切れを掃除し、それでも
  埋まっている場合は保存を諦める（取得したレート自体は呼び出し元へ返す）。
- キャッシュキーは `base` / `quote` の struct であって文字列連結ではない。区切り文字が値に混入したときに
  別ペアと衝突し得るため。
- 現在時刻は `boundary/clock.Clock` から取る（TTL の時刻依存を infra decorator の内側に閉じ込め、テストで
  固定時計を差せるようにするため）。

## API Contract

```yaml
operation: GetExchangeRates       # GET /v1/exchange-rates（security: [] = 未認証）
query:
  - name: base
    type: string
    required: true
  - name: quote
    type: string
    required: true
  - name: original
    type: string                  # 十進文字列。pattern ^\d{1,20}(\.\d{1,18})?$ / maxLength 40
    required: true                # 桁数を有界にし、巨大入力による十進解析を使った未認証 DoS を防ぐ
  - name: displayCurrency
    type: string
    required: false
    enum: [JPY]
response_200:
  - base: string
  - quote: string
  - original: string              # 十進文字列
  - converted: string             # 十進文字列（符号付きパターン）
  - referenceAmount: object|null  # { currency, amount(int64), rate(string), rateDate(string) }
responses: [400, 405, 500, 503]
```

## Notes

- 換算ヘルパ: `internal/usecase/tools/money`（`ApplyRateHalfUp`）。汎用の十進機構は `pkg/decimal`。
- 参考額の組み立ては `exchangerate.BuildReferenceAmount` の 1 箇所に集約し、後続 API（購入作成の
  `referenceAmount`）も同じ関数を再利用する。
- 決済スケール（USD セント整数）の仕様は [`docs/spec/purchase/domain.md`](../purchase/domain.md)。

[ADR-0036 (two-scale-quantity-model)]: ../../adr/0036-two-scale-quantity-model.md
[ADR-0092 (lean-a-spec-scaffold)]: ../../adr/0092-lean-a-spec-scaffold.md
[ADR-0104 (no-generic-cache-abstraction)]: ../../adr/0104-no-generic-cache-abstraction.md
