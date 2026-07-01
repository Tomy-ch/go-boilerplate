# backoff

[English](README.md) | 日本語

指数バックオフの待機時間を attempt 回数のみから算出する純関数を提供し、時刻や乱数に依存しません。

## 補足

- `Duration` は `Initial * Multiplier^attempt` を計算し、`Max` が正なら `Max` で上限を設けます。
- `Initial <= 0` の場合は `0` を返し、負の `attempt` は `0` として扱います。
- `1` 未満の `Multiplier` は `1` として扱います。
- 上限なし（`Max <= 0`）の場合は、オーバーフローによる負の待機時間を避けるため `math.MaxInt64` で頭打ちにします。

## ラップ対象

標準ライブラリ `time` および `math` パッケージ
