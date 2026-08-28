# infrastructure/dynamodbclient

DynamoDB 互換 store に繋ぐ Realtime Delivery の adapter 群 — `eventlog/dynamodb` / `streamticket/dynamodb` /
`instancelease/dynamodb`、`realtime-init` の initializer、contract test の kit — が共有する substrate です。
boundary interface は持ちません。[`awsclient`](../awsclient/README.ja.md) と同じく、接続形と retry 方針を
1 箇所で定義するために置きます。

## ここで固定するもの

| 関心 | 定義 |
| --- | --- |
| 資格情報 | `awsclient.Resolve` — 鍵 2 つが空なら SDK 既定の chain（IAM ロール）、両方あれば静的、片方だけなら起動エラー |
| endpoint | `Config.Endpoint` が非空なら `BaseEndpoint`（DynamoDB Local）、空なら SDK 既定の解決（AWS DynamoDB） |
| retry | `MaxAttempts = 3`、standard mode、`MaxBackoff = 2s`。SDK 既定とほぼ同じ値だが、ここで宣言することで、不達の store が無制限に retry されず数秒で `ErrUnavailable` として表面化する。次にどうするか（503 + `Retry-After`、outbox の backoff）は呼び出し側が決める |
| エラー正規化 | `Normalize(err, op)` — context の取り消しは `apperror.ErrCanceled`、それ以外は `apperror.ErrUnavailable`。[`objectstorage`](../objectstorage/README.ja.md) と同じく意図的に粗い: 呼び出し側は sentinel で分岐し、SDK の型は見ない。呼び出し側が区別すべき唯一の失敗 — 条件付き書き込みの拒否 — は `IsConditionalCheckFailed` で正規化の前に判定する |
| table 作成 | `EnsureTable(ctx, client, TableSpec)` — 冪等: `ResourceInUseException` は成功、次に `TableExists` waiter、TTL は `DescribeTimeToLive` が同じ属性で有効と報告していないときだけ設定。`TableSpec` は各 adapter package が公開し、table の数と名前は `internal/cli/realtimeinit` が持つ |

retry を `awsclient` の knob にしない理由: その README はサービス固有の振る舞いを各 adapter に置くと定めており、
S3 / SQS の adapter が DynamoDB の上限を共有する理由が無い。

## `testkit/`

`NewTestClient(t)` は `config.NewRealtimeTestConnection` からクライアントを組み立てます — 既定は
`localhost:8000` の DynamoDB Local で、`REALTIME_TEST_*` で AWS DynamoDB へ向け直せるので同じ contract test が
両方に対して走ります。`TableName(t, base)` は実行ごとに一意な小文字の名前を返し、`DeleteOnCleanup` がテスト
終了時に削除するので、複数の checkout が 1 つの DynamoDB Local を共有できます。

## テスト戦略

- `New` / `Normalize` / `IsConditionalCheckFailed` / `isResourceInUse` は接続無しの単体テスト: クライアントは
  `Options()` で、分類器は SDK のエラー型で検証する。
- `EnsureTable` と testkit は DynamoDB Local（手元は共有の `dynamodb_local`、CI は `go-test` の service
  container）に対する contract test。3 つの store adapter の土台なので、契約は fake でなく実物に対して立てる。
