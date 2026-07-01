# outbox

[English](README.md) | 日本語

トランザクショナル outbox のユースケース群です。**emit**（ドメイン変更と同一
トランザクションで outbox 行を書き込む）、**relay**（pending 行を claim して
publish する）、**GC**（古い published 行を刈り取る）、**replay**（dead 行を
pending へ戻す）を提供します。永続化はすべて `Store` 境界
（`internal/usecase/boundary/outbox`）を経由し、具体的な RDB 実装は
`internal/infrastructure/rdb/system_query/outbox/` にあります。

## なぜ outbox か

外部ブローカーへのイベント publish は DB トランザクションの一部ではありません。
ドメイン変更は commit したのに publish が失敗（あるいはその逆）すると、両者が
乖離し、**lost event**（イベント欠落）や phantom event（幻イベント）が発生し
ます。outbox はこのギャップを埋めます。イベントはドメイン変更と *同一*
トランザクション内で DB 行として書き込まれ、別プロセスの relay が後から publish
します（at-least-once）。そのため consumer は冪等でなければなりません。
`MessageID` が `Idempotency-Key` へ伝搬される安定した dedup キーです。

## 行のライフサイクル

```mermaid
stateDiagram-v2
    [*] --> pending: Emit (業務 tx 内)
    pending --> published: relay ClaimPending (FOR UPDATE SKIP LOCKED) + publish 成功
    pending --> pending: publish 失敗 (attempts++), 次 poll で再送
    pending --> dead: publish 失敗, attempts ≥ maxAttempts
    published --> [*]: SweepPublished (GC) で行削除
    dead --> pending: ReplayDead
```

`published` 行は **GC**（`SweepPublished`）で刈り取られ、`dead` 行は
**replay**（`ReplayDead`）で復帰します。両者は独立した経路です。GC は `dead` に
触れず、replay は `published` に触れません。

## ユースケース

### emit — `EmitUsecase`

`NewEmit(store, tracerFactory) EmitUsecase`

- `Emit(ctx, EmitInput) (uuid.UUID, error)` は outbox 行をちょうど 1 行 INSERT
  し、採番された `message_id` を返します。**ドメイン変更と同じ `tx.Manager.Do`
  の中で** 呼ぶ必要があります。そうすることで業務トランザクションが巻き戻れば
  outbox 行も巻き戻り、lost / phantom event を排除します。
- 現在の trace context を行の headers へ `traceparent` として capture し
  （`observability.InjectTraceContextToCarrier` 経由）、後続の
  relay → consumer が同一 trace に繋がります。
- `EmitInput` のフィールド: `AggregateType`・`AggregateID`（観測用）、
  `EventType`（種別 + version）、`Payload`（呼び出し側が marshal 済みのイベント
  本文 JSON）、`Headers`（外部エンドポイントへ伝搬）。`Headers` に
  `Authorization` / `Cookie` 等の機微ヘッダを入れてはいけません。そのまま外部
  エンドポイントへ送出されます。

### relay — `RelayUsecase`

`NewRelay(txm, store, publisher, metrics, clock, logger, tracerFactory) RelayUsecase`

- `RelayBatch(ctx, batchSize) (RelayResult, error)` は最大 `batchSize` 件の
  pending 行を claim して publish します。すべて **1 トランザクション** 内で
  行うため、複数の relay インスタンスが同一行を二重 publish しません。
  `batchSize <= 0` は `DefaultBatchSize`（100）にフォールバックします。
  `RelayResult` は `Claimed` と `Published` を報告します。
  - **publish 失敗はトランザクションを巻き戻しません**。行は failed
    （`attempts++`）としてマークされ、次 poll の再送に委ねられます。`attempts`
    が `DefaultMaxAttempts`（10）に達すると行は `dead` にマークされ、
    `Metrics.IncDead` が計上され、warning がログ出力されます。
  - **DB アクセス失敗**（claim / mark）のみ、トランザクションを巻き戻すエラー
    として返します。
- `RecordLag(ctx) error` は最古 pending 行の経過時間を outbox lag SLI として
  `Metrics.SetLagSeconds` に記録します。pending 行が無ければ `0` を記録します。
- `Metrics` は outbox 固有の o11y シンクです: `SetLagSeconds(ctx, seconds)` と
  `IncDead(ctx)`。

### GC — `GCUsecase`

`NewGC(store, clock) GCUsecase`

- `SweepPublished(ctx, batchSize) (int64, error)` は `DefaultRetention`（7 日）
  より古い `published` 行を `batchSize` 件ずつ削除し、合計削除件数を返します。
  `batchSize <= 0` は `DefaultGCBatchSize`（10,000）にフォールバックします。
  バッチが満たなくなり対象行が無くなるまでループします。

### replay — `ReplayUsecase`

`NewReplay(store, tracerFactory) ReplayUsecase`

- `ReplayDead(ctx, messageID *uuid.UUID) (int64, error)` は `dead` 行を
  `pending` へ戻し、戻した件数を返します。`messageID == nil` は **全** dead 行を
  replay し、非 nil の場合は当該 `message_id` のみを対象とします。

## レイアウト

| 関心事 | パス |
| --- | --- |
| boundary（`Store`） | `internal/usecase/boundary/outbox/` |
| usecase | `internal/usecase/outbox/`（本パッケージ） |
| infrastructure | `internal/infrastructure/rdb/system_query/outbox/` |
| sqlc DML | `database/dml/system_query/outbox/` |
