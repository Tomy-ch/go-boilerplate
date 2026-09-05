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

可視性には、これと対をなす第 2 の罠がある。feature が渡す cursor——History の `streamCursor`——は
PostgreSQL で commit された stream の位置だが、それを検証する store は relay が非同期に埋める。
**commit 済みと追記済みは別である**。「ここに無い」を「もう無い」と読む検証は、relay がまだ到達して
いないだけの cursor を拒み、クライアントを「同じ cursor を返す」回復経路へ送り返す——出口の無いループ
になる。同じループは反対側からも開く。idle な stream の最後の event が保持期間で消えた後、その event を
見たクライアントは何も失っていないのに、cursor の位置の item が無い。

## 決定

**PostgreSQL が現在のドメイン状態の唯一の正本であり続ける。** DynamoDB の table——**EventLog**——は
stream の replay と resume のためだけに、有限期間（7 日）配送 event を保持する。そこから何も再構築しない。
監査ログではない。

- partition key = `streamId`、sort key = `sequence`。replay の読み出しは `ConsistentRead=true`。
- History をはじめ feature が提供する読み出しは PostgreSQL の projection であり、EventLog の scan ではない。

### ordering chain は 1 つの不変条件である

correctness は「sequence が正しく採番されること」ではない。**feature の commit 順 → outbox → EventLog
可視化 → client cursor** という chain が決して壊れないことである。3 つの規則がそれを 1 つの不変条件にする。

1. **sequence に gap は無い。** feature の adapter は、機構の **sequence allocator** を通じて業務
   transaction の中で stream-local sequence を採番する: Realtime Delivery が所有する `system_cqrs` の
   table（[ADR-0033]）に stream ごと 1 行を持ち、`UPDATE … RETURNING` で更新して commit まで行ロックを
   保持する。この行は outbox 行と同じ機構の状態であり、sequence を field として持つ aggregate も、採番する
   Repository も存在しない。したがって採番順 = commit 順であり、rollback した transaction は増分も戻す。
   1 つの stream への書き込みはその行で直列化する——DynamoDB の partition が読み出し側に課すのと同じ、
   単一 stream の天井である。
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

### 不変条件が不要にするもの、そして 1 つだけそうでないもの

gap が無く連続した prefix であれば、store に replay の *floor* は要らない——stream ごとの
`floor` / `version` item も、それを進める job も不要である。まだ replay できる範囲についての事実は、
すべてログ自身から導出できる。

**ログから導出できない**のは、ログが*どこで終わっているか*である。保持期間は痕跡を残さず item を消す
ため、位置 `n` の item が無いという事実は、「`n` は追記され、その後に期限切れした」のか「relay がまだ
`n` を書いていない」のかを区別しない——そして cursor は後者を正当に指し得る。そこで EventLog は
stream ごとに metadata を 1 つだけ持つ。**append watermark**、すなわちこれまでに追記した最大の
sequence であり、追記そのものが進め、後戻りせず、TTL の対象外とする。これはログについての事実を
ログ自身が記録したものであって、outbox の status の写しではない。また、保持期間を持つログが例外なく
公開している形（partition の end offset、stream の last generated id）でもあり、trim の後も
「先頭より前」と「終端より先」を区別できるようにする。**これは何も clamp せず、順序も付けない**。
その 2 つは引き続き claim 述語が担う。

cursor の有効性は、cursor より後ろを強い一貫性で 1 回読み、それが空なら watermark を 1 回読んで決まる。

- `cursor` の次の item が `cursor + 1` で保持期間内 ⇒ replay 可能
- それが `cursor + 1` より後、または保持期間より古い ⇒ 失効
- `cursor` より後ろに何も無く `cursor >= watermark` ⇒ replay 可能。クライアントがまだ見ていないもの
  は 1 つも追記されていない——追いついているか、relay がまだ `cursor` に到達しておらず接続がそれを
  待つかのいずれかである
- `cursor` より後ろに何も無く `cursor < watermark` ⇒ 失効。cursor より後ろの event は追記され、その後
  期限切れした——idle になって丸ごと期限切れした stream が、そうでなければ「追いついている」と偽って
  見せてしまう場合である
- DynamoDB の非同期な TTL 削除が「失効」の権威になることは決してなく、読み取れない EventLog は
  retryable なサーバエラーであって cursor についての推測ではない

cursor *自身*の item が残っているかは判定に一切関与せず、初期位置も特例ではない——最初の追記までは
`0` である watermark に対する、cursor `0` にすぎない。

## 影響

### ポジティブな影響

- reconnect storm は key-range 読み出しのために作られた store が吸収し、REST を捌く PostgreSQL pool は
  それを見ない。
- 既存の全 aggregate は形を保つ。feature に realtime 配送を足すのは adapter と outbox 行であって、
  書き換えではない。
- resume は正確である。`Last-Event-ID` や `after` は gap の無い連続 sequence 上の 1 点を指し、先行 event
  が in-flight のうちに後続 event を渡されることはない。
- replay metadata は同じ table の stream ごと 1 item だけであり、それを真にする追記がその場で進める——
  別 table も job も無く、outbox が既に記録している事実の 2 つ目の記録も無い。

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

### head-of-line blocking の代わりに contiguous watermark を使う

stream ごとの「連続して append 済みの最大 sequence」を EventLog に持ち、**クライアントに見せてよい範囲を
clamp する**のに使い、先行が詰まっている間も後続を append できるようにする案。**順序機構としては却下**:
append 済みで不可視な event に価値は無く、可視性の clamp は outbox 自身の status 列と食い違い得る第 2 の
状態になる。この決定が保持する append watermark はそれではない——何も clamp せず、順序は引き続き
claim 述語が担い、保持期間が item を消した後に「無い」を分類できるようにするため、ログの終端だけを記録する。

### append 時に先行を確認する（publisher 側の順序制御）

却下。先行がまだ無いという理由で失敗を重ねる後続は attempts を消費し、回数基準の dead 規則の下では自分を
dead にし、どの規則の下でも stream 長ぶんの DynamoDB read を無駄に繰り返す。claim 述語なら後続はそもそも
claim されない。

### sequence を消費する failure marker

却下。存在しないケースを解いている（payload 固有の恒久失敗は emit 前に拒否される）。存在するケース
（substrate が落ちている）では marker 自身の append も同じく失敗する。

### cleanup job で floor を進める replay metadata

sequence が gap 無しになった時点で冗長として却下: metadata が記録するものはすべて log 自身から導出でき、
floor を進めるために全 stream を走査する job は情報を持たないコストである。ログの終端だけは、保持期間が
走った後にログ自身から導出できない唯一の値であり、それを進めるのは job ではなく追記そのものである。

### History の cursor を EventLog から導く

History が commit 済みの位置ではなく relay の位置を報告するようにし、cursor が構造的に必ず replay 可能に
なるようにする案。却下。feature の正本の読み取り経路を配信バッファの可用性に載せることになり、
「History は PostgreSQL の projection である」と衝突する。さらに行と cursor が食い違う——History が
commit 済みの行を伏せて呼び出し元が自分の書き込みを次の読み取りで見られなくなるか、History が既に返した
ものを stream が再配送するかのどちらかになる。

### 終端より先の cursor をすべて受け入れる

cursor より後ろに何も無ければ受け入れ、gap の判定は配信時点の連続性チェック（`RESYNC`）に委ねる案。却下。
そのチェックは次の event が届いたときにしか発火しない。idle になって丸ごと期限切れした stream では、
古い cursor を持つクライアントは何も告げられず、追いついていると信じたままになる。保持期間内に
再同期する義務が、紙の上にしか存在しなくなる。

### relay の位置を outbox から読む

接続時に outbox から「relay 済みの位置」を導出する案（その key の未 publish の最小 sequence、無ければ
allocator の現在位置）。却下。idle な stream への一斉再接続がまさに通る分岐に PostgreSQL を置くことになり、
さらに cursor の検証を claim 述語の head-of-line 意味論と結合させる——この決定の他の部分が引き離している
2 つである。

## 備考

- 設計正本: `docs/design/realtime-delivery.md` §2（ordering chain を状態機械として）と §5（`stream`、
  `sequence`、`cursor`、`replay floor`）。
- 関連: [ADR-0071]（機構）、[ADR-0054]（event は業務 transaction の中で emit する）、[ADR-0056]
  （本決定が拡張する claim 述語）、[ADR-0058]（outbox 行が dead になる条件——head-of-line blocking が
  dead な先頭で stream を止める理由）、[ADR-0037]（UUIDv7 の event 識別子）、[ADR-0033]（sequence table
  が属する `system_cqrs` 区分——feature の集約ではなく outbox と並ぶ機構の状態）。
- sequence 採番と claim 述語は親 issue の別フェーズ（feature adapter と outbox routing）で入る。各フェーズ
  の test が chain の自分の半分を固定する。

[ADR-0033]: 0033-system-cqrs-dml-category.ja.md
[ADR-0037]: 0037-uuidv7-identifiers.ja.md
[ADR-0054]: 0054-transactional-outbox.ja.md
[ADR-0056]: 0056-skip-locked-outbox-relay.ja.md
[ADR-0058]: 0058-outbox-dead-on-permanent-error.ja.md
[ADR-0071]: 0071-realtime-delivery-driving-mechanism.ja.md
