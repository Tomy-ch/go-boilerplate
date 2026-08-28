# infrastructure/streamticket

stream ticket の seam（`internal/usecase/boundary/realtime.StreamTicketStore`）— client が stream を開くときに提示する、
hash で保存される短命の credential（[ADR-0074](../../../docs/adr/0074-query-ticket-stream-authentication.ja.md)）— を
実装する adapter 群です。

|パス|役割|
|---|---|
|`streamticket.go`|実装を選ぶ唯一の場所。DI module が共有の `*dynamodb.Client` を渡す|
|`dynamodb/table.go`|`TableSpec` — partition key `ticket_hash`、`expires_at` の TTL、GSI `by_subject_destination`（partition `subject`、sort `destination`、`KEYS_ONLY`）|
|`dynamodb/stream_ticket.go`|adapter 本体|

## seam の写像

| seam | DynamoDB |
| --- | --- |
| item | `ticket_hash`（S、partition key）、`subject`、`destination`、`scope`、`initial_cursor`（N）、`issued_at`（RFC 3339 nano）、`expires_at`（N、epoch **秒** — TTL 属性でもあるので期限は秒精度） |
| `Save` | `PutItem` — 同じ hash の再保存は上書き。hash が空なら `ErrInvalidArgument` |
| `Find(hash, asOf)` | partition key への `ConsistentRead` の `GetItem`。接続経路が結果整合の index を読むことはない。無い、または `asOf >= expires_at` は `ok=false` — 期限は呼び出し側の時計が決め、TTL は掃除にすぎない |
| `Invalidate(subject, destination)` | GSI を `subject = :s AND destination = :d` で `Query` し（key 属性が 2 つなので区切り文字で別の組と衝突しない）、hash ごとに `DeleteItem`、ページ送り。GSI は結果整合なので直前に保存した ticket が 1 回の呼び出しでは残ることがある。revocation の主機構は fan-out による接続の close（`STOP`）で、ここはそれを無視する client への保険 |

key を `subject` + `destination` でなく hash にする理由: hot path は接続時の照合で、それを partition key に置けば
強い一貫性のまま。index を通るのは稀でベストエフォートな無効化の向きだけ。

## エラー正規化

`dynamodbclient.Normalize` — `ErrUnavailable`、取り消しは `ErrCanceled`。形の違う item は `ErrInternal`。

## テスト戦略

substrate が database ではなく DynamoDB なのでここで宣言します（[`internal/infrastructure/README.ja.md`](../README.ja.md)）:

- 各 method の `TestXxx` は DynamoDB Local（手元は共有の `dynamodb_local`、CI は `go-test` の service container）に
  対する contract test。各テストは一意な名前の table を作り cleanup で落とす。`REALTIME_TEST_*` で同じテストが AWS
  DynamoDB へ向く。
- 期限は境界の両側（`expires_at` の 1 秒前とちょうど）、reuse は `Find` の繰り返し、無効化は subject × destination
  だけが消え他の subject / destination の ticket が残ることで検証する。無効化のテストは index が追いつくのを待ってから
  検証する — adapter が文書化している結果整合そのもの。
- item の写像（`toItem` / `fromItem`）は接続無しの単体テスト。
