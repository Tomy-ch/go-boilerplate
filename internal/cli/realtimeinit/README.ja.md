# realtimeinit

`realtime-init` コマンドのコア。`REALTIME_*` / `ENDPOINT_REALTIME` が指す DynamoDB 互換 store に、Realtime
Delivery の 3 table — EventLog / StreamTicket / InstanceLease — を作ります。冪等な one-shot で、
`dynamodbclient.EnsureTable` は無い table だけを作り、`ACTIVE` を待ち、TTL は未設定のときだけ有効にするので、
再実行しても同じ状態に収束します。application は起動時に table を作りません
（[`docs/design/realtime-delivery.ja.md`](../../../docs/design/realtime-delivery.ja.md) §3.1）。

|関数|役割|
|---|---|
|`TableNames(cfg)`|`RealtimeConfig` が組み立てた名前（`realtime_<kind>_<suffix>`）を作成順に並べた 3 table 名|
|`Run(ctx, cfg, ensure, logger)`|table を順に実在させ、最初の失敗で table 名を添えて止まる。残りは次の実行に任せる — 各段が冪等なので安全|

`Ensurer` が seam で、table の*名前*を受けます。`cmd/realtime_init.go`（composition root。infrastructure を import してよい層）が
名前を各 adapter package の `TableSpec` に写し、実クライアントに束ねた `dynamodbclient.EnsureTable` を渡します。テストは
記録用の関数を渡します。コア自身は `internal/cli/README.md` の規約どおり infrastructure package を import しません。

## テスト戦略

判断ロジック — 順序、最初の失敗で止まること、seam に渡す名前 — は fake の `Ensurer` で単体テストし、接続は
開きません。`EnsureTable` が本当に DynamoDB に対して収束するかは `internal/infrastructure/dynamodbclient` の
契約で、そちらで DynamoDB Local に対して検証します。
