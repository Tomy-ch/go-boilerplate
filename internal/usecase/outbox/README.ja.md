# outbox

トランザクショナル outbox のユースケース群です。**emit**（ドメイン変更と同一
トランザクションで outbox へ 1 件記録する）、**relay**（pending のエントリを claim
して publish する）、**GC**（古い published のエントリを刈り取る）、**replay**（dead のエントリを
pending へ戻す）を提供します。永続化はすべて `Store` 境界
（`internal/usecase/boundary/outbox`）を経由し、具体的な RDB 実装は
`internal/infrastructure/rdb/system_cqrs/outbox/` にあります。

## なぜ outbox か

外部ブローカーへのイベント publish は DB トランザクションの一部ではありません。
ドメイン変更は commit したのに publish が失敗（あるいはその逆）すると、両者が
乖離し、**lost event**（イベント欠落）や phantom event（幻イベント）が発生し
ます。outbox はこのギャップを埋めます。イベントはドメイン変更と *同一*
トランザクション内で記録され、別プロセスの relay が後から publish
します（at-least-once）。そのため consumer は冪等でなければなりません。
`MessageID` が `Idempotency-Key` へ伝搬される安定した dedup キーです。

## エントリのライフサイクル

```mermaid
stateDiagram-v2
    [*] --> pending: Emit (業務 tx 内)
    pending --> published: relay ClaimPending (FOR UPDATE SKIP LOCKED) + publish 成功
    pending --> pending: 一時失敗, next_attempt_at 以降の poll で再送
    pending --> dead: 恒久失敗（エラー分類。ADR-0058）
    published --> [*]: SweepPublished (GC) でエントリ削除
    dead --> pending: ReplayDead
```

`published` のエントリは **GC**（`SweepPublished`）で刈り取られ、`dead` のエントリは
**replay**（`ReplayDead`）で復帰します。両者は独立した経路です。GC は `dead` に
触れず、replay は `published` に触れません。

## ユースケース

### emit — `EmitUsecase`

`NewEmit(store, tracerFactory) EmitUsecase`

- `Emit(ctx, EmitInput) (uuid.UUID, error)` は outbox へちょうど 1 件記録
  し、採番された `message_id` を返します。**ドメイン変更と同じ `tx.Manager.Do`
  の中で** 呼ぶ必要があります。そうすることで業務トランザクションが巻き戻れば
  outbox のエントリも巻き戻り、lost / phantom event を排除します。
- 現在の trace context をエントリの headers へ `traceparent` として capture し
  （`observability.InjectTraceContextToCarrier` 経由）、後続の
  relay → consumer が同一 trace に繋がります。
- `EmitInput` のフィールド: `AggregateType`・`AggregateID`（観測用）、
  `EventType`（種別 + version）、`Payload`（呼び出し側が marshal 済みのイベント
  本文 JSON）、`Headers`（外部エンドポイントへ伝搬）、`Channel`（配送レーン。既定値は
  無く、既知でない値は行を書く前に拒否される）、および任意の `OrderingKey` /
  `OrderingSequence` の対（エントリをストリームに載せる）。この対は全か無かで、
  両方を 1 以上の位置とともに指定するか、どちらも指定しないかのいずれかです。`Headers` に
  `Authorization` / `Cookie` 等の機微ヘッダを入れてはいけません。そのまま外部
  エンドポイントへ送出されます。
- **`Payload` の構築場所 — usecase 本体には書かない。** marshal は呼び出し側の
  責務ですが、版付きのイベント契約（struct・JSON フィールド名・`EventType` 定数）は
  **専用のイベント単位**（独立したパッケージ / 関数）に定義し、usecase からはそれを
  呼ぶだけにします。ワイヤ表現を usecase メソッドへインライン展開すると fat usecase
  が再発し、orchestration がシリアライズ形式へ結合します。分離すれば usecase は薄い
  orchestrator のままで、イベント契約も単一のテスト可能な置き場所を持てます。

### relay — `RelayUsecase`

`NewRelay(deps RelayDeps, channel) RelayUsecase`

- `RelayDeps` は協力者（トランザクションマネージャ・store・publisher・メトリクス・clock・ロガー・
  tracer の生成元）をまとめたものです。チャネルだけを別の引数に残すのは、それが依存ではなく
  「この relay が何であるか」だからです。構造体の組み立ては DI 側が行います。

- relay インスタンスは構築時に固定された **1 つの配送チャネル**を担当します。claim も
  失敗時の進行もその中で閉じるため、下流が停止したチャネルが別のチャネルのエントリを
  止めることはありません。
- `RelayBatch(ctx, batchSize) (RelayResult, error)` は当該チャネルの pending のエントリを
  最大 `batchSize` 件 claim して publish します。すべて **1 トランザクション** 内で
  行うため、複数の relay インスタンスが同一行を二重 publish しません。
  `batchSize <= 0` は `DefaultBatchSize`（100）にフォールバックします。
  `RelayResult` は `Claimed` と `Published` を報告します。
  - **publish 失敗はトランザクションを巻き戻しません**。次に何が起きるかは失敗の分類で
    決まり、試行回数では決まりません（[ADR-0058](../../../docs/adr/0058-outbox-dead-on-permanent-error.md)）:
    恒久失敗（`apperror.ErrPermanent`）は理由を記録してエントリを `dead` にマークし、
    `Metrics.IncDead` を計上して warning をログ出力します。それ以外は — 分類を一切
    持たないエラーも含めて — エントリを `pending` のまま残し、次に claim してよい時刻を
    上限 60 秒の指数バックオフ + full jitter だけ先へ進めます。
  - **DB アクセス失敗**（claim / mark）のみ、トランザクションを巻き戻すエラー
    として返します。
- `RecordLag(ctx) error` は当該チャネルの最古 pending エントリの経過時間を outbox lag SLI
  として `Metrics.SetLagSeconds` に記録します。pending のエントリが無ければ `0` を記録
  します。バックオフ待ちのエントリも未配送であることに変わりはないため除外しません。
- `RecordBlockedStreams(ctx) error` は、`dead` の先頭エントリの後ろで止まっている
  ストリーム数を `Metrics.SetBlockedStreams` に記録します。
- `Metrics` は outbox 固有の o11y シンクで、いずれのメソッドもチャネルを伴います:
  `SetLagSeconds(ctx, channel, seconds)`・`IncDead(ctx, channel)`・
  `SetBlockedStreams(ctx, channel, count)`。

### GC — `GCUsecase`

`NewGC(store, clock) GCUsecase`

- `SweepPublished(ctx, batchSize) (int64, error)` は `DefaultRetention`（7 日）
  より古い `published` のエントリを `batchSize` 件ずつ削除し、合計削除件数を返します。
  `batchSize <= 0` は `DefaultGCBatchSize`（10,000）にフォールバックします。
  バッチが満たなくなり対象のエントリが無くなるまでループします。

### replay — `ReplayUsecase`

`NewReplay(store, tracerFactory) ReplayUsecase`

- `ReplayDead(ctx, messageID *uuid.UUID) (int64, error)` は `dead` のエントリを
  `pending` へ戻し、戻した件数を返します。`messageID == nil` は dead の **すべて** を
  replay し、非 nil の場合は当該 `message_id` のみを対象とします。

## 消費側

このパッケージが担うのは **producing 側**だけです。relay が publish した後にメッセージがどうなるかは
worker サブシステムの関心であり、両端を配線するのは integrator です。何も消費していない outbox は
不完全な状態ではなく、正当な構成の 1 つです。

両端はコードではなく transport で出会います。`relay` が `publisher.Message` を adapter へ渡し、adapter が
payload を本文へ、イベント種別と `message_id` を名前付きのメタデータへ載せ、`worker.Handler` が
`worker.Message` からそれらを読み戻します。どちらの端も相手を import しません。

<!-- sample-api:begin -->
サンプルは、この経路を実際に動かせるよう両端を配線しています。

| 段 | 場所 |
| --- | --- |
| 退会トランザクション内で `user.withdrawn.v1` を emit | `internal/usecase/user` |
| relay → publish | `outbox-relay` + `internal/infrastructure/queue/sqs`（`OUTBOX_PUBLISHER=sqs`） |
| consume → 退会証跡の保存 | [`internal/controller/worker/withdrawalarchive`](../../controller/worker/withdrawalarchive/README.ja.md) |

<!-- sample-api:end -->

## レイアウト

| 関心事 | パス |
| --- | --- |
| boundary（`Store`） | `internal/usecase/boundary/outbox/` |
| usecase | `internal/usecase/outbox/`（本パッケージ） |
| infrastructure | `internal/infrastructure/rdb/system_cqrs/outbox/` |
| sqlc DML | `database/dml/system_cqrs/outbox/` |
