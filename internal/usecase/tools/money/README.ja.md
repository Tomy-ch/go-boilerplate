# money

[English](README.md) | 日本語

Usecase 層向けのマネー計算ヘルパを提供します。マネー値は `float` 誤差の累積を避けるため
**最小単位の整数**（例: USD セント、JPY 円）で保持し、丸め方式は ADR で確定したアプリ規約に従います。

## 公開 API

- `ApplyRateHalfUp(amountMinor int64, rate float64, scale int64) int64` — 最小単位整数 `amountMinor`
  に `rate` を適用し、`scale`（`amountMinor` を得た固定小数スケール。セント化なら 100）で除して、最終除算で
  **half-away-from-zero（0 から遠い方向への四捨五入）** する。`rate` は 10^6 固定小数整数へ写像し、中間積は
  `math/big` で計算するため `int64` 乗算はオーバーフローしない。丸めは 1 回だけ。`amountMinor` / `rate` は
  任意符号可（負値は 0 から遠い方向へ丸め）、`scale` は正の整数を前提とする（`<= 0` はゼロ除算で panic）。

## 設計方針

- 丸めをここに集約し、呼び出し側では丸めないことで方針のドリフトを防ぐ。
- half-up 方式と丸め 1 箇所ルールは
  [ADR-0099](../../../../docs/adr/0099-reference-amount-half-up-rounding.md) に記録している。
- infra 依存を持たない機械的変換のみ。
