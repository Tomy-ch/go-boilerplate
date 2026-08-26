# retry

[English](README.md) | 日本語

失敗分類を*消費する*有限リトライの行動層を提供します。`classify → bounded attempts → backoff + full jitter → deadline-aware` を 1 度だけ実装し、自身の失敗をリトライ可能と分類する任意の呼び出し側が共有します。

## このパッケージの意図

`pkg/backoff` は待機時間を attempt 回数のみから算出する純関数で、時刻や乱数に依存しません。full jitter の付与には `math/rand/v2` が必要なため、その責務を本パッケージへ閉じます。これにより `backoff` の純粋性を保ちつつ、乱数依存を `retry` 側に局所化します。

`pkg/` 配下は相互独立（pkg → pkg の import は `pkg/xerrors` のみ例外的に許可）のため、`Policy.Backoff` は `backoff.Exponential` 型ではなく**関数値**（`attempt → 基本待機時間`）で受け取り、呼び出し側（`internal/`）が `backoff.Exponential.Duration` 等を結線します。

## API

|シンボル|説明|
|---|---|
|`Do(ctx, sleeper, policy, isRetryable, fn)`|`fn` を有限リトライで実行。`isRetryable(err)` が真の間、試行ごとに `policy.Backoff`（+ full jitter）だけ待機して再試行|
|`Full(d)`|`[0, d]` の一様乱数（full jitter）を返す。`d <= 0` なら `0`|
|`Policy`（構造体）|`MaxAttempts` ＋ `Backoff`（`func(attempt int) time.Duration`。`nil` は待機なし）|
|`Sleeper`（インターフェース）|`Sleep(ctx, d) error` — 待機の抽象。呼び出し側が持つ sleeper 型が構造的に充足|

## 補足

- `Do` は `fn` を最低 1 回実行し（`MaxAttempts < 1` は `1` 扱い）、最後に観測した error（成功時は `nil`）を返します。
- `isRetryable` は `fn` の返した非 nil error に対してのみ呼ばれます。
- `sleeper.Sleep` が error を返した（ctx 打ち切り / deadline）場合、`Do` は sleep の error ではなく**直前の `fn` の error** を返します。リトライ対象だった元の失敗を呼び出し側へ伝えるためです。
- `Sleeper` は `pkg/` が `internal/` に依存しないようローカル定義です。呼び出し側が持つ sleeper 型が、本パッケージのインターフェースを import せずに構造的に充足します。

## ラップ対象

標準ライブラリ `context` / `time` / `math/rand/v2`。他パッケージへの依存はありません。

## テストカバレッジ例外

以下の未被覆分岐は**構造上到達不能**として、ほぼ 100% の被覆期待の対象外とする。これを塗る
ための contrived テストや追加実装は行わない:

- `retry.go` `Do` — 試行ループ後の末尾 `return err`。最終試行は必ずループ内で return する
  （`attempt == attempts`）ため、末尾 return はコンパイラ充足のためだけに存在し実行されない。

**ガバナンス:** カバレッジ例外は**任意に追加しない**。新規エントリはアーキテクト等の適切な
承認者の承認を要する。
