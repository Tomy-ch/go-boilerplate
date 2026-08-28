# usecase/realtime

Realtime Delivery（[`docs/design/realtime-delivery.ja.md`](../../../docs/design/realtime-delivery.ja.md)）の機構側 usecase。
stream 接続を commit する前に決めるべきこと — 提示された cursor からまだ replay できるか、提示された ticket が
有効か — と、ticket の発行側を持ちます。依存は `boundary/realtime` と `boundary/clock` だけで、feature の語彙は
持ちません。

| usecase | 決めること | エラー |
| --- | --- | --- |
| `CursorValidator.Validate(streamID, cursor)` | replay floor。保存せず EventLog から**導出**する（[ADR-0072](../../../docs/adr/0072-postgres-state-dynamodb-eventlog.ja.md)）: `cursor` より後ろに現存する最初の event を強い一貫性で 1 回読む: それが `cursor+1` なら `realtime.EventLogRetention` より古くない限り replay 可、`cursor+1` より後ろなら gap、何も無く初期位置でない cursor 自身の event も無ければ消失。1 回で読むのは、`cursor+1` と最新を別々に読むと、その間に挟まった普通の append が gap に見えるため | `ErrCursorExpired`（client は canonical recovery path — History — へ戻る）。store の失敗は `apperror.ErrUnavailable` のまま通し、呼び出し側が `503 + Retry-After` を返せるようにする |
| `TicketIssuer.Issue(in)` → `TicketView` | `SecretGenerator` の新しい 256 bit の値。SHA-256 の hash を subject / destination / scope / initial cursor に束ねて保存し、`TicketTTL`（5 分）だけ有効 | store の失敗はそのまま通す |
| `TicketVerifier.Verify(value, destination)` → `VerifiedTicketView` | 値の hash が存在し、`clock.Now()` で期限内で、この destination に束ねられていること | すべての失敗が `ErrTicketInvalid`（`apperror.ErrUnauthenticated` を包む）— 未知・期限切れ・destination 違いはわざと区別しない |
| `AccessRevoker.Revoke(subject, destination)` | feature が subject の destination への権利を取り下げるときに呼ぶ失効の seam（[ADR-0074](../../../docs/adr/0074-query-ticket-stream-authentication.ja.md)）。その組の ticket を**先に**すべて無効にし（`StreamTicketStore.Invalidate` — 無効化された ticket は `Verify` を通らないので、`STOP` を無視した client も再接続できない）、そのうえで `RevocationNotifier` を通じて各 serve instance に該当接続を閉じさせる。通知に失敗しても無効化は成立している | store と notifier の失敗はそのまま通す |

`ErrCursorExpired` を `apperror` に足さず package の sentinel にしているのは、本 package が HTTP を知らないため。
`410` への写像は stream handler（`internal/controller/stream`）が `apperror.ErrGone` を重ねて行う。

本 package の外: replay の読み取りと catch-up（stream handler が接続状態と一緒に持つ）、lease の heartbeat
（serve lifecycle）、orphan cleanup の引き受け（cleanup job）。

## テスト戦略

boundary は生成 mock（`boundary/realtime/mock`、`boundary/clock/mock`）で、store は開きません。各判定は境界の
両側で固定します — 保持期間ちょうどとその 1 秒後、初期位置での空 stream と先頭 event が消えた stream — また
store の失敗はすべて、失効や無効 ticket に読み替えられずに*そのまま*通ることを検証します。
