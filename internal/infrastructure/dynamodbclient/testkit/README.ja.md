# dynamodbclient/testkit

Realtime Delivery の contract test 用の helper。`NewTestClient(t)` は `config.NewRealtimeTestConnection`
（`REALTIME_TEST_*` で AWS DynamoDB へ向け直さない限り `localhost:8000` の DynamoDB Local）へ繋ぎ、
`TableName(t, base)` は実行ごとに一意な小文字の table 名を返し、`DeleteOnCleanup(t, client, table)` はテスト
終了時に table を削除します。各テストは adapter の `TableSpec` + `dynamodbclient.EnsureTable` で自分の table を
作るので、複数 checkout の並行実行が同じ table に触れることはありません。

## contract test を AWS に対して実行する

`make realtime-contract-test` は Realtime Delivery の contract test を実行します。既定の接続先は
DynamoDB Local と GoAWS なので、その場合は先に `make infra-up` が要ります。同じテストを AWS へ
向け直すことが、受入基準「DynamoDB Local と production DynamoDB で同じ application contract test が
通る」の裏付けになります。これは CI ジョブではなく人が実行するものです。ターゲットを用意したのは、
自動で走らせるためではなく、実行を再現可能にするためです。

`config.NewRealtimeTestConnection` は 5 つの変数を `os.LookupEnv` で読むため、**空文字**は未設定とは
別物です。endpoint と資格情報の変数を空文字にすることで解決が AWS SDK の既定 chain に委ねられ、
未設定のままにすればエミュレータ向けの既定値が使われます。

```sh
REALTIME_TEST_ENDPOINT= REALTIME_TEST_PUBSUB_ENDPOINT= \
REALTIME_TEST_ACCESS_KEY_ID= REALTIME_TEST_SECRET_ACCESS_KEY= \
REALTIME_TEST_REGION=ap-northeast-1 make realtime-contract-test
```

コンテナではなく host で走らせるのは、SDK の既定 chain（環境変数・共有プロファイル・SSO）が host 側に
あるためです。各テストは一意な名前のテーブルを自分で作って落とすので実行後に何も残りませんが、実資源を
作って課金は発生するため、この用途に切り分けたアカウントへ向けてください。
