# dynamodbclient/testkit

Realtime Delivery の contract test 用の helper。`NewTestClient(t)` は `config.NewRealtimeTestConnection`
（`REALTIME_TEST_*` で AWS DynamoDB へ向け直さない限り `localhost:8000` の DynamoDB Local）へ繋ぎ、
`TableName(t, base)` は実行ごとに一意な小文字の table 名を返し、`DeleteOnCleanup(t, client, table)` はテスト
終了時に table を削除します。各テストは adapter の `TableSpec` + `dynamodbclient.EnsureTable` で自分の table を
作るので、複数 checkout の並行実行が同じ table に触れることはありません。
