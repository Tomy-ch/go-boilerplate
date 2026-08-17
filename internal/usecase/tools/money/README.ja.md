# money

[English](README.md) | 日本語

Usecase 層のマネー決済 **policy** を提供します。正確な算術・丸め・最小単位スケーリングの機構は
[`pkg/decimal`](../../../../pkg/decimal/README.ja.md) にあり、本パッケージは決済額を *どの* 最小単位桁へ
*どの* 丸めモードで落とすかだけを選択します。

## 公開 API

- `ApplyRateHalfUp(amount, rate decimal.Decimal, minorUnitDigits int32) (int64, error)` — `amount × rate`
  を正確に計算し、`minorUnitDigits` 桁へ **half-away-from-zero（0 から遠い方向への四捨五入）** で丸めて
  決済スケールの `int64` へ変換する（`decimal.ToScaledInt64`）。丸めは決済境界のこの 1 点でのみ行い、経路に
  `float` を持たないため累積誤差が生じない。最小単位整数が `int64` を超える場合はエラーを返す。

## 設計方針

- 丸めをここに集約し、呼び出し側では丸めないことで方針のドリフトを防ぐ。
- half-up 方式と丸め 1 箇所ルールは
  [ADR-0037 (two-scale-quantity-model)](../../../../docs/adr/0037-two-scale-quantity-model.md) に、具体的な方針はそれを適用する機能とともに（`docs/spec/exchange-rate/`。サンプル側の内容）記録している。
- 汎用の十進機構は [`pkg/decimal`](../../../../pkg/decimal/README.ja.md)。本パッケージは policy
  （`minorUnitDigits`・half-up）のみを持ち、infra 依存を持たない。
