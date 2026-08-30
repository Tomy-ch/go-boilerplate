# usecase/realtime

Realtime Delivery（[`docs/design/realtime-delivery.ja.md`](../../../docs/design/realtime-delivery.ja.md)）の機構側 usecase。
stream 接続を commit する前に決めるべきこと — 提示された cursor からまだ replay できるか、提示された ticket が
有効か — と、ticket の発行側、そして commit 済みの接続が寿命の間ずっと繰り返す読み取りを持ちます。依存は
`boundary/realtime` と `boundary/clock` だけで、feature の語彙は持ちません。

| usecase | 決めること | エラー |
| --- | --- | --- |
| `CursorValidator.Validate(streamID, cursor)` | replay floor。保存せず EventLog から**導出**する（[ADR-0072](../../../docs/adr/0072-postgres-state-dynamodb-eventlog.ja.md)）: `cursor` より後ろに現存する最初の event を強い一貫性で 1 回読む: それが `cursor+1` なら `realtime.EventLogRetention` より古くない限り replay 可、`cursor+1` より後ろなら gap、何も無く初期位置でない cursor 自身の event も無ければ消失。1 回で読むのは、`cursor+1` と最新を別々に読むと、その間に挟まった普通の append が gap に見えるため | `ErrCursorExpired`（client は canonical recovery path — History — へ戻る）。store の失敗は `apperror.ErrUnavailable` のまま通し、呼び出し側が `503 + Retry-After` を返せるようにする |
| `Replayer.ReadPage(streamID, after)` | 接続がまだ見ていないもの。`after` より後ろの event を昇順で、最大 `ReplayPageLimit`（64）件。接続の commit 直後の初回 replay も、その後の再読み込み（wakeup、定期の catch-up）も同じ呼び出しで、違うのは契機だけ。連続性はここでは判定**しない**。返った page が呼び出し側自身の位置の続きかどうかは呼び出し側の比較であり、自分がどこに居るかを知っているのは呼び出し側だけなので | store の失敗は `apperror.ErrUnavailable` のまま通す |
| `TicketIssuer.Issue(in)` → `TicketView` | `SecretGenerator` の新しい 256 bit の値。SHA-256 の hash を subject / destination / scope / initial cursor に束ねて保存し、`TicketTTL`（5 分）だけ有効 | store の失敗はそのまま通す |
| `TicketVerifier.Verify(value, destination)` → `realtime.StreamGrant` | 値の hash が存在し、`clock.Now()` で期限内で、この destination に束ねられていること | すべての失敗が `ErrTicketInvalid`（`apperror.ErrUnauthenticated` を包む）— 未知・期限切れ・destination 違いはわざと区別しない。区別すると、その ticket が存在するかどうかを呼び出し側に教えてしまうため |
| `LeaseKeeper.Beat(id)` / `Release(id)` | instance lease（[ADR-0073](../../../docs/adr/0073-sns-sqs-instance-fanout.ja.md)）: `Beat` は「今生きている」ことを `clock.Now() + LeaseExpiry`（2 分）の期限付きで記録し、`Release` は instance が自分でリソースを片付けたときに記録を削除する。間隔（`LeaseHeartbeatInterval`、30 秒）と cleanup の余裕（`LeaseCleanupMargin`、5 分）は、heartbeat loop と orphan cleanup job がここから読む固定値 | store の失敗はそのまま通す |
| `OrphanSweeper.Sweep()` → `SweepResult` | どの死んだ instance を誰が回収してよいか: 期限切れが `LeaseCleanupMargin`（5 分）を過ぎた lease を条件付き書き込みで引き受け、`OrphanReclaimer` で受信先を片付け、そのうえで lease を閉じる。閉じる側も条件付きなのは、`Heartbeat` が引き受けの記録に触れないため、復帰した instance の生きた lease を消してしまうから。順序は固定で、lease が受信先を辿る唯一の索引である以上、先に閉じると片付けに失敗した残骸を誰も見つけられなくなる。1 件の失敗では止まらず、内訳と束ねたエラーの両方を返す | store と reclaimer の失敗はそのまま通し、束ねて返す |
| `AccessRevoker.Revoke(subject, destination)` | feature が subject の destination への権利を取り下げるときに呼ぶ失効の seam（[ADR-0074](../../../docs/adr/0074-query-ticket-stream-authentication.ja.md)）。その組の ticket を**先に**すべて無効にし（`StreamTicketStore.Invalidate` — 無効化された ticket は `Verify` を通らないので、`STOP` を無視した client も再接続できない）、そのうえで `RevocationNotifier` を通じて各 serve instance に該当接続を閉じさせる。通知に失敗しても無効化は成立している | store と notifier の失敗はそのまま通す |

`ErrCursorExpired` を `apperror` に足さず package の sentinel にしているのは、本 package が HTTP を知らないため。
`410` への写像は stream handler（`internal/controller/stream`）が `apperror.ErrGone` を重ねて行う。

本 package の外: *いつ* replay するか（`internal/controller/stream` の connection registry がスケジュールと接続状態を
持ち、本 package は読み取りを実行するだけ）、heartbeat の*ループ*（`internal/controller/realtime` が serve lifecycle の
スケジュールで `LeaseKeeper.Beat` を駆動する）、*いつ*掃除するか（scheduler が `cmd job orphan-cleanup` を起動する。
`internal/controller/job/orphancleanup` は内訳を報告するだけ）。

## テスト戦略

boundary は生成 mock（`boundary/realtime/mock`、`boundary/clock/mock`）で、store は開きません。各判定は境界の
両側で固定します — 保持期間ちょうどとその 1 秒後、初期位置での空 stream と先頭 event が消えた stream — また
store の失敗はすべて、失効や無効 ticket に読み替えられずに*そのまま*通ることを検証します。
