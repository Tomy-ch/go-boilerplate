# Realtime consumer エンジンガイド（`internal/controller/realtime`）

## オニオンアーキテクチャでの役割

- outbox relay や worker engine と同格の **consume 駆動 driving adapter**。新しい層ではなく、機構への
  もう 1 つの入口。fan-out の受信側にあたる
  （[ADR-0073](../../../docs/adr/0073-sns-sqs-instance-fanout.ja.md)）: serve インスタンス自身の queue
  （`realtime.InstanceSubscription`）をここで drain し、すべての通知を接続側へ渡す。
- engine が担うのは **loop と batch ごとの coalescing と待機制御だけ**。wakeup が*何を意味するか* —
  どの接続を起こし、どれを閉じるか — は sink の責務であり、`controller/stream` の connection registry が
  実装する。本 package が宣言するのは必要な sink だけで、接続についてそれ以上は知らない。registry へは
  sink 越しにしか届かず、その結線は realtime の DI module が行う。
- 依存は usecase 層の port のみ: `realtime.InstanceSubscription`・`ucrealtime.LeaseKeeper`・
  `clock.Sleeper`・`logging.Logger`・`observability.LayerTracer`（`TracerFactory.Controller()` 経由）。
  `internal/infrastructure/*` は import せず（depguard `maintain_a_sound_controller`）、
  `InstanceLeaseStore` も名指ししない（architecture test の allowlist）。

## 公開 API

- `Engine` — 常駐 consumer。`NewEngine(sub, reprovision, fanout, wakeups, revocations, sleeper, log, tf, metrics, set)`。
  `Run(ctx) error` が loop 本体。
- `Settings` — `BatchSize`（既定 10。queue 自身の上限）と `ErrorBackoff`（既定 5 秒）。ゼロ値と負値は既定へ
  フォールバックする。receive が long poll であり、どちらの値もデプロイ先で変わらないため config は無い。
- `Waker.Wake(ctx, streamID, upTo)` / `Revoker.Revoke(ctx, subject, destination)` — 受け手。どちらも
  loop 上で同期的に呼ばれるので、実装は印を付けるか通知するだけで、replay を待ってはならない。重複は
  正常であり、冪等でなければならない。
- `FanoutObserver.ObserveFanout(err)` — 受信の成否を毎回伝える受け口。通知が届いているかどうかは
  受信を試みた者にしか分からないので、loop が外へ報告する。readiness の側が queue を突いて確かめると、
  この loop とメッセージを取り合うことになる。
- `Reprovisioner.Reprovision(ctx) error` — 受信が `realtime.ErrReceivingEndGone` で失敗したときに loop が
  依頼する受け手。受信先を作り直す前に lease を書き直す順序が要り、その順序を知っているのは両者を合成する
  側だけなので、loop は自分で作り直さず委ねる
  （[`docs/design/realtime-delivery.ja.md`](../../../docs/design/realtime-delivery.ja.md) §2.5）。
  `ReprovisionFunc` が関数をこれに適合させる。
- `Heartbeat` — `NewHeartbeat(keeper, id, sleeper, log, tf, metrics)`。`Run(ctx)` は instance lease を直ちに書き、
  以後 `ucrealtime.LeaseHeartbeatInterval` ごとに書く。単発の失敗はログに出して次の tick で再試行する。
  instance が orphan になるのは `LeaseExpiry` より長く沈黙したときだけ。

## loop の意味論（`Run`）

| `Receive` の結果 | 次の動作 | 理由 |
| --- | --- | --- |
| 通知あり | coalesce → sink → それぞれ `Delete` → 再び receive | receive は long poll（20 秒）であり、成功時に待つものは無い |
| 通知なし | 再び receive | 同上 |
| エラー | ログに出し `ErrorBackoff` だけ待つ | 一時的な障害に hot loop でぶつかり続けず、収まるのを待つ |
| `ctx` 完了 | `nil` を返す | loop の先頭、エラーの後、そして sleeper で検査する |

- **coalescing は batch ごと**。同じ stream への wakeup は最大 sequence へ畳む。どれも「cursor の先から
  読み直せ」としか言っていないため。batch を*またぐ* coalescing は registry の責務であって engine の
  ものではない。
- kind を読めなかった通知（`Kind == ""`）は計数し、warn でログに出して削除する。誰も対処できず、放置
  すれば永遠に再配送されるため。
- **削除は受け渡しの後**。削除の失敗はログに出して loop を続ける。通知は戻ってくるが、sink の冪等性が
  吸収する。 停止中に失敗した削除はログに出さない（原因は取り消しであり substrate ではない）。heartbeat も
  停止中に失敗した `Beat` を同じに扱う。より悪い失敗は wakeup を失うことであり、それすら定期 catch-up が補う。

## テスト戦略

他のループ駆動 controller と同じ（[`../README.ja.md`](../README.ja.md)）: instance subscription と lease
keeper は生成 mock、sleeper も mock にしてテストは一切 sleep せず、loop は loop として駆動する — 1 反復の効果
（batch → sink → 削除。coalescing は生成した `Waker` / `Revoker` の mock が受けた呼び出しで検証する）、停止の意味論（loop 先頭・receive 中・
backoff 中での cancel）、反復ごとのエラー経路（backoff して継続、削除失敗のログ）、そして settings の
既定値。作り直しの経路も単独ではなく loop の一部として固定する — 受信が
`realtime.ErrReceivingEndGone` で失敗した周回は `Reprovisioner` に 1 度依頼してから backoff する、という
形なので、harness は既定で作り直しを呼ばせず、必要なテストが個別に宣言する。非公開ヘルパーを直接呼ぶ
テストだけで済ませると、`Run` 側の呼び出し箇所を削除しても気づけない。
