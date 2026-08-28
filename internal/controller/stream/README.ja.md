# stream

generic な SSE endpoint `GET /v1/streams/{destination}` の handler です。Realtime Delivery の transport のうち
**レスポンスを確定する前**に「この接続を始めてよいか」を決める部分を持ちます。確定後の部分（connection registry、
admission、replay、heartbeat、control event）は、この package が宣言し Phase 6 が実装する `Streamer` seam の担当です。

設計の正本: `docs/design/realtime-delivery.md` §2.3（接続の lifecycle）と §3.1（package の配置）、
ADR-0074 (query-ticket-stream-authentication)。

## 接続時に起きること

| 順 | 段 | 場所 | 拒否 |
| --- | --- | --- | --- |
| 1 | query の `ticket` を path の `destination` に対して検証し、束縛を `StreamGrant` context スロットへ置く | `auth/` — `StreamTicket` security scheme。OpenAPI validator がパラメータ検証より先に呼ぶ | 401（`UNAUTHORIZED`）— 未知・期限切れ・失効・destination 違いは区別しない |
| 2 | `after` / `Last-Event-ID` を spec の pattern で検査する | OpenAPI validator | 400（`BAD_REQUEST`） |
| 3 | cursor を解決する: `Last-Event-ID` → `after` → ticket の初期位置 | `cursor.go` | 負数・範囲外は 400（`INVALID_STREAM_CURSOR`） |
| 4 | cursor を replay floor と突き合わせる | `usecase/realtime.CursorValidator` | そこから replay を始められなければ 410（`STREAM_CURSOR_EXPIRED`）、EventLog を読めなければ 503 |
| 5 | 検証済みの `StreamRequest` を `Streamer` へ渡す | Phase 6 | — |

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
禁じています。`BindHandler` は `Streamer` が揃った時点（Phase 6）で Realtime の DI module が登録し、
`di/module/controller.go` には載せません。

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
- `internal/integration` は本物の validator → scheme → handler の連鎖を HTTP 越しに 401 / 400 / 410 / 200 で通す。
