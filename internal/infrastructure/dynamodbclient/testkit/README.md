# dynamodbclient/testkit

Test helpers for the Realtime Delivery contract tests. `NewTestClient(t)` connects to
`config.NewRealtimeTestConnection` (DynamoDB Local on `localhost:8000` unless `REALTIME_TEST_*`
redirects it to AWS DynamoDB), `TableName(t, base)` returns a per-run unique lowercase table name,
and `DeleteOnCleanup(t, client, table)` drops the table when the test finishes. Each test creates its
own table through the adapter's `TableSpec` + `dynamodbclient.EnsureTable`, so parallel runs from
several checkouts never touch the same table.

## Running the contract tests against AWS

`make realtime-contract-test` runs the Realtime Delivery contract tests. It defaults to DynamoDB
Local and GoAWS, which need `make infra-up` first. Redirecting the same tests at AWS is what backs
the acceptance criterion "the same application contract test passes against DynamoDB Local and
production DynamoDB", and it is a manual run rather than a CI job — the target exists so the run is
reproducible, not so it happens automatically.

`config.NewRealtimeTestConnection` reads the five variables with `os.LookupEnv`, so an **empty**
value is distinct from an unset one: emptying the endpoint and credential variables is what hands
resolution back to the AWS SDK's default chain, while leaving them unset keeps the emulator
defaults.

```sh
REALTIME_TEST_ENDPOINT= REALTIME_TEST_PUBSUB_ENDPOINT= \
REALTIME_TEST_ACCESS_KEY_ID= REALTIME_TEST_SECRET_ACCESS_KEY= \
REALTIME_TEST_REGION=ap-northeast-1 make realtime-contract-test
```

The target runs on the host rather than in a container because the SDK's default chain (environment,
shared profile, SSO) lives there. Each test creates and drops its own uniquely named table, so a run
leaves nothing behind — but it does create real resources and incur charges, so point it at an
account set aside for this.
