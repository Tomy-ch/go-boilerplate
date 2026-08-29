# stream

generic な SSE endpoint `GET /v1/streams/{destination}` です。Realtime Delivery の transport 全体 — **レスポンスを
確定する前**に「この接続を始めてよいか」を決める handler と、確定後のすべて（admission、replay、catch-up、
heartbeat、control event、drain）を回す connection registry — を持ちます。

設計の正本: `docs/design/realtime-delivery.md` §2.3（接続の lifecycle）と §3.1（package の配置）、
ADR-0074 (query-ticket-stream-authentication)。

## 接続時に起きること

| 順 | 段 | 場所 | 拒否 |
| --- | --- | --- | --- |
| 1 | query の `ticket` を path の `destination` に対して検証し、束縛を `StreamGrant` context スロットへ置く | `auth/` — `StreamTicket` security scheme。OpenAPI validator がパラメータ検証より先に呼ぶ | 401（`UNAUTHORIZED`）— 未知・期限切れ・失効・destination 違いは区別しない |
| 2 | `after` / `Last-Event-ID` を spec の pattern で検査する | OpenAPI validator | 400（`BAD_REQUEST`） |
| 3 | cursor を解決する: `Last-Event-ID` → `after` → ticket の初期位置 | `cursor.go` | 負数・範囲外は 400（`INVALID_STREAM_CURSOR`） |
| 4 | cursor を replay floor と突き合わせる | `usecase/realtime.CursorValidator` | そこから replay を始められなければ 410（`STREAM_CURSOR_EXPIRED`）、EventLog を読めなければ 503 |
| 5 | 検証済みの `StreamRequest` を `Streamer` へ渡す | `registry.go` | instance が接続上限に達している・有限の待ち時間内に初回 replay の枠が取れない・drain 中のいずれかなら 503（`SERVICE_UNAVAILABLE`）+ `Retry-After` |

拒否はすべて共有のエラーハンドラが返す通常の HTTP エラーで、ここはレスポンス本文を書きません。ticket の生値は
エラー・ログのフィールド・span の属性のどこにも出ません — `httpstack/redaction` がログに出す前にリクエスト URI と
query から取り除きます。

## strict-server を使わない理由

ADR-0014 (oapi-codegen-strict-server) は他のすべての handler を `strict-server` で生成し、その glue が呼び出しごとに
1 つの型付きレスポンスを marshal します。SSE のレスポンスは、明示的な flush・deadline・in-band の切断を伴って
event を書き続ける長寿命の stream で、1 つの返り値を持ちません。そのため `v1/streams` tag だけは `echo5-server` のみで
生成し、handler が `echo.Context` を受け取ります。設計の正本が名指しする唯一の例外です。

## 配置

この package は `handler/` の下ではなく隣に置きます。feature のリソースではなく機構の transport であり、
`internal/architest/realtime_isolation_test.go` が `internal/domain/<feature>` と `internal/usecase/<feature>` の import を
禁じています。`BindHandler` は Realtime の DI module（`internal/di/module/realtime.go`）が登録し、`di/module/controller.go` には載せません。
その module はまだ serve graph に入っていません。加わるのは feature adapter が必要としたとき、つまり #1416 と #1417 です。

## 実行時: 確定後に 1 本の接続がすること

`Registry` は instance が持つ live な接続の index です。1 つの値が 4 つの呼び出し側に応えます。4 つとも同じ index を
読むからで、分けると別々のロックを同期させ続けることになるためです: handler が呼ぶ `Streamer`、consumer エンジン
（`controller/realtime`）が通知を渡す `Waker` と `Revoker`、そして serve lifecycle が HTTP shutdown の前に走らせる
`hook.Drainer`。index は 2 つ持ちます — wakeup は stream を名指すので stream 単位、revocation は subject を名指すので
subject 単位。どちらも feature の語ではありません。機構はどの subject が user でどれが operator かを学びません。

commit された接続はそれぞれ、channel 以外は何も共有しない 2 つの goroutine を走らせます:

| goroutine | 持つもの | すること |
| --- | --- | --- |
| **fetcher** | どこまで読んだかの位置 | 初回 replay、その後は wakeup か jitter 付き 30 秒の catch-up での再読み込み。そのたびに共有の replay semaphore から枠を取る |
| **pump**（handler 自身の goroutine） | wire | event・control event・15 秒の heartbeat を書く。書き込みの前に毎回 write deadline を張り直す |

手を入れる前に知っておく価値のある帰結が 3 つあります:

- **初回 replay の枠はレスポンスを確定する前に取り**、fetcher は最初の読み取りが終わるまでそれを持ち続ける。
  確定した後に待つと、開いたのに黙ったままの接続ができ、client には backpressure ではなくハングに見える。
- **送信バッファが埋まったら event を捨てずに接続を閉じる。** バッファは replay 1 ページ分
  （`ucrealtime.ReplayPageLimit`）で、event は EventLog に残っているので、再接続すれば client 自身の
  `Last-Event-ID` から replay できる。捨てる側にすると、順序の連鎖全体が乗っている「連続した prefix」の不変条件が
  壊れる。
- **読み取りの失敗では接続を閉じない。** EventLog に届かないとき fetcher はログを出して次の catch-up を待つ
  （設計 §2.6）。依存先の一時的な不調で全接続を閉じると、復旧が再接続の嵐に化ける。

`pump` はバッファ済みの event より先に queue された control event を取るので、`STOP` が満杯のバッファの後ろで待つ
ことはありません。control event は SSE の `id` を持ちません。これが `Last-Event-ID` を業務 event の stream だけの
関数に保っています。

### 停止時の予算

`Drain` は新規接続を拒否し、開いている全接続へ `RECONNECT` / `SERVER_DRAINING` を送り、それらを閉じて待ちます —
停止 context と固定の **10 秒**の短いほうまで。待つ対象は 1 接続あたり control frame 1 つと close 1 つで、どちらも
すでに 10 秒の write deadline で抑えられているので、この予算は覆う範囲に対して十分です。設定可能にせず固定なのは、
`SHUTDOWN_TIMEOUT` の残りが後続の段の持ち分だからです: consumer の停止、instance queue と subscription の teardown、
そして HTTP shutdown。drain が予算を使い切ると queue と lease が orphan cleanup job へ残され、それこそ停止順が
避けるために存在する結末になります（設計 §2.5）。これは #1414 から引き継いだ未決の論点への答えです。

### この package が送る reason code

`SERVER_DRAINING`（`RECONNECT`）、`TEMPORARILY_OVERLOADED`（`RETRY_LATER`。バッファが溢れる際の best effort）、
`AUTH_REFRESH_REQUIRED`（`REAUTHENTICATE`。1 時間の寿命で）、`AUTHORIZATION_REVOKED`（`STOP`）、
`CURSOR_TOO_OLD`（`RESYNC`。再読み込みが不連続で返ってきたとき）。

`STREAM_RECOVERY_FAILED` は契約には宣言されていますが**予約**で、この package は送りません。走っている接続は
EventLog に届かなくても失敗せずに生き延びますし、先頭行が死んで詰まった stream は接続ではなく relay から見えるものです。

## まだ別の場所にあるもの

- **型付き config。** `MaxConnections` と `ReplayConcurrency` は `Settings` のフィールドで、ゼロ値は既定値に落ちます。
  consumer エンジンの `realtime.Settings` と同じ形です。これらを環境変数として公開し `env/**` を同期させるのは #1417。
- **メトリクス。** close はすべて理由（`close_reason`）を構造化ログに記録し、分岐はそれぞれに counter を付けられる形に
  分けてあります。`realtime_*` メトリクスの登録・label 規則・その architecture test は #1417 で、span link も同じく
  #1417 が持ちます — 配送の封筒も wakeup の通知も、今は起点の trace を運んでいません。
- **`OBS_TARGET_STATUS_CODES`。** 410 は `env/.env`・`.env.ci`・`.env.dast`・`.env.dev`・`.env.stg` にあり、
  `.env.prd` にだけありません。つまり `STREAM_CURSOR_EXPIRED` の拒否は production を除くすべてでログに出ます。
  production も揃えるかどうかは env のポリシー判断（`env/README.md`）。

## サブパッケージ

| Package | 役割 |
| --- | --- |
| `auth/` | `StreamTicket` security scheme の `SchemeAuthenticator`。scheme が宣言したパラメータを読み、`TicketVerifier.Verify` を呼び、`ctxhelper.SetStreamGrant` へ書く |
| `gen/` | `v1/streams` tag の oapi-codegen 生成物（型 + 非 strict の echo server） |

## テスト戦略

- `stream_handler_test.go` は mock の `CursorValidator` と stub の `Streamer` で確定前の契約を固定する: cursor の優先順位、
  各拒否のエラー分類と `code`、拒否後に `Streamer` へ到達しないこと。
- `cursor_test.go` は 10 進のパースを端で固定する（`0`、`MaxInt64`、overflow、符号、先頭ゼロ）— spec の pattern は 19 桁を
  許すので overflow に到達できる。
- `auth/stream_ticket_test.go` は scheme 名が `openapi.yaml` と一致すること、verifier がリクエストの context と path の
  destination を受け取ること、拒否された ticket がスロットを空のまま残すことを固定する。
- `sse_test.go`・`registry_test.go`・`connection_test.go` は確定後の半分を覆う。これらを形づくる規則が 2 つある。
  **どのテストも実時間を待たない**: heartbeat も寿命も catch-up もすべて `clock.Sleeper` を通し、テストは「どの tick を
  解放するか」をテスト側が指名するまでブロックする Sleeper を渡す。時間を測るのではなくスケジュールの判断を検証する
  ため。**flush と write deadline に触れるものは `httptest.Server` の上で走らせる**: `httptest.ResponseRecorder` は
  どちらも実装していないので、recorder ベースで writer をテストしても、本番経路が使わない writer を検証することになる。
- `internal/integration` は本物の validator → scheme → handler → registry の連鎖を HTTP 越しに通す: 確定前の拒否は
  401 / 400 / 410 / 200、接続そのものについては接続上限、初回 replay の飽和、`after` と `Last-Event-ID` 双方での再開、
  wakeup 無しの catch-up による配送、失効、drain。`sse_client_test.go` は設計が求める Go の reference client —
  テスト専用で、出荷する SDK ではない — であり、client 側の契約を固定するのはこれ: control event は `Last-Event-ID` を
  動かさないこと、`STOP` / `REAUTHENTICATE` / `RESYNC` では server の EOF より先に client 側が閉じること。
- 受け入れ基準 9 は意図的に 2 つに分けている。飽和した接続が**閉じる**ことはバッファを決定的に満たせる
  `connection_test.go` で、**event が失われない**ことは再接続して最初の接続が読めなかった分を受け取る
  `internal/integration` で固定する。loopback 越しに実ソケットのバッファを満たそうとすると、前者がタイミング頼みの
  テストになるため。
