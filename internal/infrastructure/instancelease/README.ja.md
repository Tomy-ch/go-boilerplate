# infrastructure/instancelease

instance lease の seam（`internal/usecase/boundary/realtime.InstanceLeaseStore`）— serve instance が crash した後に
fan-out の resource を回収できるよう持つ生存記録（[ADR-0073](../../../docs/adr/0073-sns-sqs-instance-fanout.ja.md)）—
を実装する adapter 群です。lock でも leader election でもありません。

|パス|役割|
|---|---|
|`instancelease.go`|実装を選ぶ唯一の場所。DI module が共有の `*dynamodb.Client` を渡す|
|`dynamodb/table.go`|`TableSpec` — partition key `instance_id`、**TTL 無し**: DynamoDB が勝手に消す lease は orphan の証跡ごと消える|
|`dynamodb/instance_lease.go`|adapter 本体|

## seam の写像

時刻は条件式で比較できるよう epoch ナノ秒（`N`）で持ちます。

| seam | DynamoDB |
| --- | --- |
| `Heartbeat` | `UpdateItem` `SET heartbeat_at, expires_at` — 無ければ作り、回収の項目には触れないので、heartbeat と回収の引き受けが競合しても引き受けを消さない |
| `Delete` | `DeleteItem`。無い lease は成功 |
| `ListExpired(asOf)` | `expires_at < :asOf` の `Scan`、`ConsistentRead`、ページ送り。母数は serve instance の数なので scan が正しい形 |
| `AcquireCleanup(claim)` | `attribute_exists(instance_id) AND expires_at < :before AND (attribute_not_exists(cleanup_owner) OR cleanup_owner_until < :now)` の下で `UpdateItem` `SET cleanup_owner, cleanup_owner_until`。条件不成立は `false, nil` — 他者が引き受け済みか回収対象が無い — でエラーではない。expiry と `:before` の margin は呼び出し側（`internal/usecase/realtime`）のもの |

## エラー正規化

`dynamodbclient.Normalize` — `ErrUnavailable`、取り消しは `ErrCanceled`。形の違う item は `ErrInternal`。

## テスト戦略

substrate が database ではなく DynamoDB なのでここで宣言します（[`internal/infrastructure/README.ja.md`](../README.ja.md)）:

- 各 method の `TestXxx` は DynamoDB Local（手元は共有の `dynamodb_local`、CI は `go-test` の service container）に
  対する contract test。各テストは一意な名前の table を作り cleanup で落とす。`REALTIME_TEST_*` で同じテストが AWS
  DynamoDB へ向く。
- 引き受けの競合をそのまま検証する: 1 つの期限切れ lease に 2 者が claim して最初だけ `true`、safety margin の
  内側の claim は `false`、無い lease への claim は何も作らない。
- item の写像（`nano` / `fromNano` / `fromItem`）は接続無しの単体テスト。
