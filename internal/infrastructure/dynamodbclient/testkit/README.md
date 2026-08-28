# dynamodbclient/testkit

Test helpers for the Realtime Delivery contract tests. `NewTestClient(t)` connects to
`config.NewRealtimeTestConnection` (DynamoDB Local on `localhost:8000` unless `REALTIME_TEST_*`
redirects it to AWS DynamoDB), `TableName(t, base)` returns a per-run unique lowercase table name,
and `DeleteOnCleanup(t, client, table)` drops the table when the test finishes. Each test creates its
own table through the adapter's `TableSpec` + `dynamodbclient.EnsureTable`, so parallel runs from
several checkouts never touch the same table.
