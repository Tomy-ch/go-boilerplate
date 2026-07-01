# worker/testkit

[English](README.md) | 日本語

`worker` seam（`Consumer` / `FailureHandler`）の in-memory テストダブルです。
実 broker 無しで engine の receive → process → ack/nack 不変条件テストを green に
するために用います（broker 非依存の 2nd 実装、SDK 非依存）。

## `Fake`

`NewFake() *Fake` は `worker.Consumer` と `worker.FailureHandler` の両方を実装
します。in-memory のキュー・in-flight 集合・ID ごとの配送回数カウンタを保持し、
すべての Ack / Nack / Extend / Fail 呼び出しを検証用に記録します。

### seam メソッド（`Consumer` / `FailureHandler`）

- `Receive(ctx, limit) ([]worker.Message, error)` — 最大 `limit` 件を返します。
  キューが空の場合は投入 / 再配送または `ctx` 完了まで **ブロック** します。
  配送のたびにメッセージの `ReceiveCount` を加算します。
- `Ack(ctx, m) error` — メッセージを in-flight から除去し、記録します。
- `Nack(ctx, m) error` — 即時に再配送（キュー末尾へ戻す）し、記録します。
- `NackWithBackoff(ctx, m, d) error` — 要求された遅延 `d` を記録してから再配送
  します（fake は実時間の遅延を模しません）。
- `Extend(ctx, m, d) error` — Extend の呼び出し回数を記録します。`SetExtendErr`
  でエラーが設定されている場合はそのエラーを返します。
- `Fail(ctx, m, cause) error` — dead-letter（`FailureHandler`）の呼び出しを記録
  します。

### テスト操作・検証用ヘルパー

- `Enqueue(msgs ...worker.Message)` — メッセージを投入します。
- `FailReceiveOnce(err error)` — 次回以降の `Receive` で返すエラーを 1 件
  キューイングします（複数回呼べば順に消費）。
- `SetExtendErr(err error)` — 以降の `Extend` が常に `err` を返すようにします。
- `AckedIDs() []string` / `NackedIDs() []string` — 呼び出し順の ID 一覧。
- `ExtendCount(id string) int` — 指定 ID の Extend 呼び出し回数。
- `NackBackoffOf(id string) time.Duration` — 指定 ID に対し最後に要求された
  backoff（未記録は 0）。
- `NackBackoffApplied(id string) bool` — 指定 ID に対し `NackWithBackoff` が
  呼ばれたか（full jitter で遅延が 0 になり得るため、有無で判定）。
- `Failed() []FailedRecord` — 呼び出し順の `Fail` 記録。`FailedRecord` は
  `Message` と `Cause` を保持します。
- `QueueLen() int` — 受信待ちのメッセージ数。
- `InflightLen() int` — 受信済み・未 Ack/Nack のメッセージ数。

`Fake` は mutex で保護されており、並行利用は安全です。
