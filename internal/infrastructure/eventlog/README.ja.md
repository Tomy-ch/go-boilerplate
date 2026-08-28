# infrastructure/eventlog

EventLog の seam（`internal/usecase/boundary/realtime.EventLogStore`）— Realtime Delivery の有界 replay store
（[ADR-0072](../../../docs/adr/0072-postgres-state-dynamodb-eventlog.ja.md)）— を実装する adapter 群です。

## 役割

この package の `New` が実装を選び、`dynamodb/` が DynamoDB 実装を持ちます。vendor の語彙（table /
partition key / `ConsistentRead` / `LastEvaluatedKey`）はここで止まり、上の seam は stream / sequence / 封筒
だけを語ります。

|パス|役割|
|---|---|
|`eventlog.go`|実装を選ぶ唯一の場所。DI module が [`dynamodbclient`](../dynamodbclient/README.ja.md) で組み立てた共有の `*dynamodb.Client` を渡す|
|`dynamodb/table.go`|`TableSpec` — `realtime-init` と contract test が作る table 定義|
|`dynamodb/event_log.go`|adapter 本体|

## seam の写像

| seam | DynamoDB |
| --- | --- |
| item | partition key `stream_id`（S）、sort key `sequence`（N、10 進）。`event_id`、`event_type`、`occurred_at`（RFC 3339 nano、UTC）、`schema_version`（N）、`payload`（B、空なら属性無し）、`expires_at`（N、epoch 秒 = `occurred_at` + `realtime.EventLogRetention`） |
| `Append` | `attribute_not_exists(stream_id)` 付きの `PutItem`。`ConditionalCheckFailedException` なら既存 item を `ConsistentRead` で読み戻して `event_id` を比較: 同じなら成功（outbox relay の retry は特別扱い無しで冪等）、違えば `ErrSequenceConflict`。先に `event.Validate()` を通すので不正な封筒は保存されない |
| `ReadAfter` | `Query` `stream_id = :s AND sequence > :after`、`ConsistentRead`、昇順、`Limit`（既定 100、上限 1000）。`HasMore` は `LastEvaluatedKey != nil` — `len == Limit` ではない。DynamoDB は 1 MiB でも打ち切るため。続きは最後の event の sequence から読む。sequence は gap 無しなので不透明な cursor を seam に通さない |
| `Latest` | 降順 `Limit: 1` の `Query`、`ConsistentRead` |
| `Find` | `ConsistentRead` の `GetItem` |

`expires_at` は TTL の掃除にしか使いません。cursor がまだ replay できるかは `internal/usecase/realtime` が
`OccurredAt` と `realtime.EventLogRetention` から判定し、store は age で filter しないので、両者が数値で
食い違うことはありません。

## エラー正規化

SDK の失敗は `dynamodbclient.Normalize` で `apperror.ErrUnavailable`（context の取り消しは `ErrCanceled`）に
なります。上の写像と形が合わない item は `ErrInternal` — この adapter が書いた覚えの無いものが store にある
ことを意味します。

## テスト戦略

substrate が database ではなく DynamoDB なので、[`internal/infrastructure/README.ja.md`](../README.ja.md) を
継承せずここで宣言します:

- **各 method の `TestXxx` はそのまま DynamoDB Local に対する contract test** — 手元は共有の `dynamodb_local`、
  CI は `go-test` の service container。1:1 対応と「Local と本番 DynamoDB で同じテストが通る」が同じテストで
  成立する。AWS へ向けるのは `REALTIME_TEST_*` だけ（[`dynamodbclient/testkit`](../dynamodbclient/testkit/README.ja.md)）。
- 各テストは `TableSpec` から一意な名前の table を作り cleanup で落とすので、複数 checkout が 1 つの
  DynamoDB Local を共有しても互いのデータに触れない。
- item の写像（`toItem` / `fromItem` / 属性の読み取り）は接続無しの単体テストで、形が崩れた item の経路も含む。
- DynamoDB Local は期限切れ item を削除しないので、保持期間のテストは古い event が `OccurredAt` を保ったまま
  返ることを検証する — 失効の判定は usecase の責務で、それがこのテストを両 substrate で意味あるものにする。
