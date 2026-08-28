---
status: accepted
date: 2026-08-28
deciders: [maintainers]
tags: [architecture, persistence, async, realtime, reliability]
---

# ADR-0072: 現在状態は PostgreSQL が持ち、DynamoDB EventLog は有限の replay store であって event-sourced log ではない

## ステータス

accepted

## 背景

切断した SSE 接続から復帰したクライアントは、取りこぼした event を順序どおりに、重複なく受け取らなければ
ならない。そのためには「この stream の cursor N 以降をすべて」を、再接続の burst の下でも安価に答えられ、
かつ通常の REST traffic を捌く transactional database に触れない store が要る——PostgreSQL の pool に
落ちる reconnect storm は API 全体を道連れにする。

state store の隣に event store を置くと、よく知られた 2 つの過ちを招く。1 つは、それを source of truth に
昇格させて state を再構築すること（event sourcing）。これは既存の全 aggregate を書き換え、state の住処を
2 倍にする。もう 1 つは、それを監査ログへ流れさせること。単なる配送バッファに保持期間・不変性・法的要件が
付いてくる。

順序には固有の罠がある。書き込み時に割り当てる stream ごとの sequence だけでは足りない。sequence 5 の
publish が失敗して retry 待ちの間に sequence 6 が publish されクライアントに読まれると、クライアントの
cursor は 6 へ進み、sequence 5 は永久に配送されない——catch-up query は cursor の *後* しか見ず、
機構はクライアントが穴に気づける合図を意図的に持たない。

## 決定

**PostgreSQL が現在のドメイン状態の唯一の正本であり続ける。** DynamoDB の table——**EventLog**——は
stream の replay と resume のためだけに、有限期間（7 日）配送 event を保持する。そこから何も再構築しない。
監査ログではない。

- partition key = `streamId`、sort key = `sequence`。replay の読み出しは `ConsistentRead=true`。
- History をはじめ feature が提供する読み出しは PostgreSQL の projection であり、EventLog の scan ではない。

### ordering chain は 1 つの不変条件である

correctness は「sequence が正しく採番されること」ではない。**feature の commit 順 → outbox → EventLog
可視化 → client cursor** という chain が決して壊れないことである。3 つの規則がそれを 1 つの不変条件にする。

1. **sequence に gap は無い。** feature は stream を所有する行を `UPDATE … RETURNING` で更新して
   stream-local sequence を業務 transaction の中で採番し、その行ロックを commit まで保持する。したがって
   採番順 = commit 順であり、rollback した transaction は増分も戻す。1 つの stream への書き込みはその行で
   直列化する——DynamoDB の partition が読み出し側に課すのと同じ、単一 stream の天井である。
2. **client-visible な sequence は連続した prefix を成す。** outbox relay は、同一 ordering key で
   より小さい sequence が未 published のうちは行を claim しない——既存の `FOR UPDATE SKIP LOCKED`
   （[ADR-0056]）と並ぶ claim 述語で表現する stream-local な head-of-line blocking。publish 中の行はまだ
   `pending` なので、述語は追加の状態なしに後続を除外する。
3. **terminal failure は stream を止める。sequence を飛ばすことはない。** 先頭が dead になった stream は
   metric（`realtime_blocked_streams`）で可視化し、dead 行を replay したときに再開する。failure marker、
   tombstone、「sequence を消費して先へ進む」は存在しない。

head-of-line 規則を保つのが安価なのは、realtime channel の失敗 class が 3 つしかないからである:
substrate に到達できない（全 event が同時に失敗する）、conditional write が衝突する（冪等な成功）、
payload が不正（outbox 行を書く前に拒否済み）。「1 件だけ恒久的に失敗し次は成功する」には発生源が無い。
だから規則が効くことはほぼ無く、効いたときの原因は系統的で、停止こそが正しい合図である。

### 不変条件が不要にするもの

gap が無く prefix が連続なら、store に replay metadata は要らない——stream ごとの `latest` / `floor` /
`version` item も、それを進める job も:

- latest sequence は降順読み出しの先頭 item である。
- cursor が失効しているのは、`cursor + 1` の item が無いのに後続が存在するとき、または存在しても
  `occurredAt` が保持期間より古いとき——DynamoDB の非同期 TTL 削除を「失効」の正本にはしない。
- EventLog が読めないのは retry 可能な server error であり、cursor の推測ではない。

## 影響

### ポジティブな影響

- reconnect storm は key-range 読み出しのために作られた store が吸収し、REST を捌く PostgreSQL pool は
  それを見ない。
- 既存の全 aggregate は形を保つ。feature に realtime 配送を足すのは adapter と outbox 行であって、
  書き換えではない。
- resume は正確である。`Last-Event-ID` や `after` は gap の無い連続 sequence 上の 1 点を指し、先行 event
  が in-flight のうちに後続 event を渡されることはない。
- replay metadata が無いことで、table 1 つ、job 1 つ、同じ事実の 2 つの記録が食い違うという一群の不整合が
  消える。

### ネガティブな影響

- 1 つの stream への書き込みは PostgreSQL の 1 行で直列化する。最初の consumer（user ごとに 1 つの
  active な会話）では見えないが、高頻度で書かれる stream は他のどの限界より先に行ロックの天井に当たり、
  機構はそのための sharding を提供しない。
- dead な先頭行は operator が replay するまで stream を止める。停止は意図的であり、metric が監視されて
  いることに依存する。
- 保持期間は配送の性質であって業務の性質ではない。7 日より長く不在だったクライアントは feature の正規
  読み出し経路（History）から再同期しなければならず、feature はそれを提供しなければならない。
- 2 つの store が最大 7 日間、同じ payload の複製を持つ。したがって EventLog は primary database と同じ
  at-rest encryption の義務を負う。

## 検討した代替案

### event sourcing——EventLog を source of truth にする

却下。同じ事実に対して 2 つの store が権威になり、全 aggregate を event から再構築させ、配送の関心を
アプリケーション全体の永続化モデルにしてしまう。

### PostgreSQL に event table を置く

却下。REST と pool を共有する。機構が生き残らねばならない reconnect storm は、まさにその pool を枯らす
負荷である。

### head-of-line blocking の代わりに contiguous watermark

stream ごとの「連続して append 済みの最大 sequence」を EventLog に持ち、クライアントに見せてよい範囲を
それで抑える案。却下: 先行が詰まっていても後続を append できるが、append 済みで不可視な event に価値は
無く、watermark は outbox 自身の status 列と食い違い得る第 2 の状態になる。

### append 時に先行を確認する（publisher 側の順序制御）

却下。先行がまだ無いという理由で失敗を重ねる後続は attempts を消費し、回数基準の dead 規則の下では自分を
dead にし、どの規則の下でも stream 長ぶんの DynamoDB read を無駄に繰り返す。claim 述語なら後続はそもそも
claim されない。

### sequence を消費する failure marker

却下。存在しないケースを解いている（payload 固有の恒久失敗は emit 前に拒否される）。存在するケース
（substrate が落ちている）では marker 自身の append も同じく失敗する。

### cleanup job で floor を進める replay metadata

sequence が gap 無しになった時点で冗長として却下: metadata が記録するものはすべて log 自身から導出でき、
floor を進めるために全 stream を走査する job は情報を持たないコストである。

## 備考

- 設計正本: `docs/design/realtime-delivery.md` §2（ordering chain を状態機械として）と §5（`stream`、
  `sequence`、`cursor`、`replay floor`）。
- 関連: [ADR-0071]（機構）、[ADR-0054]（event は業務 transaction の中で emit する）、[ADR-0056]
  （本決定が拡張する claim 述語）、[ADR-0058]（outbox 行が dead になる条件——head-of-line blocking が
  dead な先頭で stream を止める理由）、[ADR-0037]（UUIDv7 の event 識別子）。
- sequence 採番と claim 述語は親 issue の別フェーズ（feature adapter と outbox routing）で入る。各フェーズ
  の test が chain の自分の半分を固定する。

[ADR-0037]: 0037-uuidv7-identifiers.ja.md
[ADR-0054]: 0054-transactional-outbox.ja.md
[ADR-0056]: 0056-skip-locked-outbox-relay.ja.md
[ADR-0058]: 0058-outbox-dead-after-max-attempts.ja.md
[ADR-0071]: 0071-realtime-delivery-driving-mechanism.ja.md
