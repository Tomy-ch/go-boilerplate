# realtimeinit

`realtime-init` コマンドのコア。`REALTIME_*` / `ENDPOINT_REALTIME` が指す DynamoDB 互換 store に、Realtime
Delivery の 3 table — EventLog / StreamTicket / InstanceLease — を作り、続いて SNS 互換エンドポイント
`ENDPOINT_REALTIME_PUBSUB` 上に `REALTIME_TOPIC` が名指す fan-out topic を作ります。冪等な one-shot で、
`dynamodbclient.EnsureTable` は無い table だけを作り、`ACTIVE` を待ち、TTL は未設定のときだけ有効にし、
`CreateTopic` は既存の topic をそのまま返すので、再実行しても同じ状態に収束します。application は起動時に
table も topic も作りません
（[`docs/design/realtime-delivery.ja.md`](../../../docs/design/realtime-delivery.ja.md) §3.1）。application 自身が
作る resource は instance ごとの queue だけです。

|関数|役割|
|---|---|
|`TableNames(cfg)`|`RealtimeConfig` が組み立てた名前（`realtime_<kind>_<suffix>`）を作成順に並べた 3 table 名|
|`Run(ctx, cfg, ensure, logger)`|table を順に実在させ、最初の失敗で table 名を添えて止まる。残りは次の実行に任せる — 各段が冪等なので安全|
|`TopicName(arn)`|topic 名、すなわち `REALTIME_TOPIC` を `:` で区切った最後の要素。無ければ `ErrTopicARNInvalid`|
|`RunTopic(ctx, topicARN, ensure, logger)`|topic を実在させ、基盤が返した ARN が設定値と一致することを要求する（違えば `ErrTopicARNMismatch`）— 設定した ARN と account や region が食い違う emulator では、publisher が誰も subscribe していない topic を指したままになってしまうため|

`Ensurer` と `TopicEnsurer` が seam で、*名前*を受けます。`cmd/realtime_init.go`（composition root。infrastructure を
import してよい層）が table 名を各 adapter package の `TableSpec` に写して実クライアントに束ねた
`dynamodbclient.EnsureTable` を渡し、topic 名を `infrastructure/realtime` の `realtime.EnsureTopic` に
束ねます。テストは記録用の関数を渡します。コア自身は `internal/cli/README.md` の規約どおり infrastructure
package を import しません。

## テスト戦略

判断ロジック — 順序、最初の失敗で止まること、seam に渡す名前、ARN の比較 — は fake の `Ensurer` /
`TopicEnsurer` で単体テストし、接続は開きません。`EnsureTable` が本当に DynamoDB に対して収束するかは
`internal/infrastructure/dynamodbclient` の契約で、そちらで DynamoDB Local に対して検証します。
