---
status: accepted
date: 2026-08-28
deciders: [maintainers]
tags: [security, http, realtime, contract]
---

# ADR-0074: SSE stream は subject / destination / scope / expiry に bind した opaque な query ticket で認証する

## ステータス

accepted

## 背景

ブラウザの `EventSource` API は request header を設定できない。他の全 endpoint が使う bearer token 方式
（[ADR-0016]、[ADR-0021]）は、SSE 接続で最も重要な瞬間——接続時——に使えない。代替はそれぞれ、
無視ではなく秤にかけるべきコストを持つ。cookie session は、それ以外は stateless な機構に CSRF、
`SameSite`、CORS の設計を引き込む。query string の bearer token は長寿命の credential を URL、access log、
`Referer` header に置く。one-time token はブラウザ自身の再接続——同じ URL を送り直す——を壊す。

stream の認可には、request 単位の検査には無かった寿命の問題もある。REST request は一度認可されれば
ミリ秒で終わる。stream は 1 時間開いたままで、その間に subject の destination へのアクセスが取り消され得る
——この service によって（membership の無効化、会話へのアクセス剥奪）、あるいは identity provider に
よって（account の無効化）。接続時を認証するものは、その後に何が起きるかを言わなければならない。

## 決定

feature の endpoint が自身の認可を行った後に **opaque な 256-bit stream ticket** を発行する。ticket は
hash としてのみ保存し、4 つのものに bind する: **subject**（feature 非依存の principal 識別子——stream
runtime はそれが user か operator かを知らない）、**destination**（読んでよい stream）、**scope**、
**expiry**。クライアントは stream を開くときに query parameter として提示する。

- **TTL の間は再利用可、one-time ではない。** 発行後 5 分間は新規接続を受け付けるので、ブラウザの自動
  再接続——同じ URL、`Last-Event-ID` 付き——はそのまま動く。
- **ticket TTL と connection lifetime は別の量である。** TTL は *新しい* 接続を開始できる期限を定める。
  確立済みの接続は独自の **maximum lifetime 1 時間** を持ち、到達するとサーバーは `REAUTHENTICATE`
  control event を送って閉じる。クライアントは feature の認可経路で新しい ticket を得て再接続する。
  どちらの値も固定であり、deployment の設定ではない。
- **log にも trace にも出さない。** 生の ticket は request URI、query parameter、error / recovery log、
  span attribute から、それらが emit される前に除去する。これは HTTP stack で強制し、handler 任せに
  しない。
- **この service 内での失効は即時である。** この service がアクセスを取り消すとき——REST では identity
  resolver が request ごとに強制している membership の soft-delete（[ADR-0021]）、または feature による
  subject の destination アクセス剥奪——feature は失効 seam（`usecase/realtime.AccessRevoker`。feature が既に使う ticket 発行の seam の隣）を呼ぶ。seam は
  先にその subject がその destination に持つ全 ticket を無効化し、次に infrastructure が実装する
  `boundary/realtime.RevocationNotifier` を通じ既存の fan-out（[ADR-0073]）で全 serve instance
  に通知する。各 instance は該当する接続を `STOP` control event で閉じる。connection registry はこのために
  subject で索引する。ticket も無効化されるので、`STOP` を無視するクライアントはそれで再接続できない。
- **identity provider での失効は観測しない。** IdP で無効化された account の発行済み JWT は `exp` まで
  有効なままで、この service は IdP を polling しない——それは既存の認証姿勢であって新しいものではない。
  したがって開いている stream がそのような変更に収束するのは 1 時間の maximum lifetime を通じてだけで
  あり、REST の bearer token が `exp` を通じて収束するのと同じである。IdP から service への薄い ingress
  adapter が追加されたときは、同じ失効 seam を呼び、即時 close に変わる。

## 影響

### ポジティブな影響

- platform の `EventSource` とその組み込み再接続で動く。クライアントに polyfill も独自 transport も要らない。
- URL 上の credential は短命で、1 つの stream に限定され、hash で保存され、失効後は無用——query string の
  bearer token に欠けている性質を持つ。
- 認可は今ある場所に留まる: ticket 発行時の feature usecase と、request ごとの identity resolver。stream
  runtime は独自の認可を行わず、行うための feature 語彙も持たない。
- この service 内でのアクセス取り消しは、次の再接続時ではなく数秒で開いている stream に届く。

### ネガティブな影響

- stream する feature はそれぞれ ticket 発行 endpoint を持ち、scope を自分で決めなければならない。汎用の
  「ticket をくれ」操作は無い。
- 接続は IdP 側の無効化より最大 1 時間長く生きられる。上限は明示されており REST token の上限と一致するが、
  ゼロではなく上限である。
- ticket の log / trace からの除去は共有 middleware に掛ける規則であり、忘れた新しい logging 経路は
  credential を漏らす。除去を固定する test だけが守りである。
- ticket store は TTL 付きの table を 1 つ増やし、失効は wakeup topic に message type を 1 つ足す。

## 検討した代替案

### stream request での `Authorization: Bearer`

使えない: `EventSource` は header を設定しない。fetch ベースのクライアントに header を要求するのは、
全 consumer に独自 transport を押し付けることになる。

### stream endpoint 用の cookie session

却下。SSE 固有の CSRF、`SameSite`、CORS の credential 扱いを増やし、この service の前段の BFF が既に
持っている session と二重になる。

### one-time ticket

却下。ブラウザは同じ URL で再接続する。消費済み ticket は自動再接続をすべて `401` にし、`Last-Event-ID`
による resume を無効にする。

### JWT そのものを query parameter に

却下。長寿命で replay 可能な credential が URL、access log、`Referer` header に載り、鍵の rotation 以外に
取り消す手段が無い。

### 失効を connection lifetime だけに任せる

却下。失効がまさにこの service で起きた場合でも、取り消された subject が最大 1 時間 event を受け取り続ける。
即時 close のコストは fan-out message 1 つである。

### stream runtime から identity provider を polling する

却下。認証設計が既に約束している local 評価モデルに反し、接続ごとに IdP の可用性への依存を足し、それでも
polling の間隔は取りこぼす。

## 備考

- 設計正本: `docs/design/realtime-delivery.md` §2（ticket と connection の lifecycle）と §4
  （`REAUTHENTICATE` と `STOP` のクライアント契約）。本決定が継承する 2 つの失効軸は
  `docs/design/auth.md` §2、log 除去が効く場所は `docs/design/security.md`。
- 関連: [ADR-0071]、[ADR-0073]、[ADR-0016]（spec 駆動の認証: query-ticket security scheme は OpenAPI で
  宣言し bearer scheme と並べて dispatch する）、[ADR-0021]（認証の失敗は request を拒否する——不正または
  失効した ticket は response commit 前に拒否する）、[ADR-0108]（再接続の *頻度* は edge の関心。ticket が
  縛るのは *誰か* であって *何回か* ではない）。

[ADR-0016]: 0016-spec-driven-request-validation.ja.md
[ADR-0021]: 0021-optional-authentication-fail-closed.ja.md
[ADR-0071]: 0071-realtime-delivery-driving-mechanism.ja.md
[ADR-0073]: 0073-sns-sqs-instance-fanout.ja.md
[ADR-0108]: 0108-no-in-app-rate-limiter.ja.md
