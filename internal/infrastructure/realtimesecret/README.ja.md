# realtimesecret

`boundary/realtime.SecretGenerator` — Realtime Delivery の不透明な ticket 生値 — を OS の暗号論的乱数源から
実装します。32 byte の乱数を base64url（パディング無し、43 文字）にしたもので、構造も claim も持たず、
store は hash しか持たないため、提示する以外に照合の手段はありません。

[`token`](../token/README.ja.md) と同じ形をわざと重複させ、依存しません: `boundary/token` は cart の
セッション追跡のためのもので sample feature と一緒に削除され、Realtime Delivery はその後も compile / test
できなければならないからです。

## 補足

- tracer span は無い — 実 I/O が無いため（[Observability](../README.ja.md)）。
- `math/rand` ではなく `crypto/rand`。推測できる ticket は credential ではない。
- 短い読み出しは短い値ではなくエラー。

## テスト戦略

単体テストのみ: 実出力に対する encoding と幅、繰り返し呼び出しでの一意性、失敗する reader を注入した
エラー経路。contract test すべき substrate は無い。
