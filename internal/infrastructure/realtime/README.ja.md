# infrastructure/realtime

Realtime Delivery の fan-out 基盤（[ADR-0073](../../../docs/adr/0073-sns-sqs-instance-fanout.ja.md)）の
adapter 群です。event を EventLog へ append してから全 serve インスタンスを起こす publish 側と、1 つの
インスタンスの queue と subscription をそのインスタンスが生きている間だけ所有する receive 側を持ちます。

## 役割

`realtime.go` が実装を選ぶ唯一の場所で、`aws/` が SNS / SQS 実装を持ちます。SDK の語彙（topic と queue の
ARN、queue URL、receipt handle、message attribute）はここで止まり、上の port が語るのは instance
subscription・不透明な receipt を伴う通知・wakeup・revocation だけです。

|パス|役割|
|---|---|
|`realtime.go`|実装を選び、DI module・CLI・テストが必要とするもの（`NewClients`・`NewPublisher`・`NewRevocationNotifier`・`NewInstanceSubscription`・`EnsureTopic`・2 つの `AttributesBuilder` builder、キー付き入力 `QueueAttributesInput` / `SubscriptionTarget`）を再 export する|
|`aws/client.go`|`NewClients` — 資格情報の解決は 1 回（`awsclient.Resolve`）、1 つの endpoint 上に 2 つのサービスクライアント（SNS と SQS は endpoint を共有する。GoAWS と AWS の違いは endpoint と資格情報だけ）|
|`aws/publisher.go`|`realtime` outbox チャネル向けの `boundary/publisher.Publisher`: payload → `DeliveryEvent` → `EventLogStore.Append` → wakeup の SNS `Publish`|
|`aws/revocation.go`|`realtime.RevocationNotifier`: 同じ topic への revocation。message attribute で区別する|
|`aws/subscription.go`|`realtime.InstanceSubscription`: queue 作成 → ARN 解決 → 属性設定 → subscribe → `RawMessageDelivery=true`。long poll の receive、delete、unsubscribe → queue 削除|
|`aws/attributes.go`|`AttributesBuilder` — インスタンス queue を作るときの属性集合。production の builder（`NewQueueAttributes(QueueAttributesInput{TopicARN, DLQARN})`）は全部を返す|
|`aws/topic.go`|`EnsureTopic` — ARN を返す冪等な `CreateTopic`。`realtime-init` と contract test 用（アプリケーションの起動時には決して呼ばない）|
|`aws/policy.go`|queue の access policy: `aws:SourceArn` が wakeup topic のときに限り `sns.amazonaws.com` からの `sqs:SendMessage` を許可する|
|`aws/message.go`|wire 形式: ADR-0073 が定める wakeup の body `{eventId, streamId, sequence}`、revocation の body `{subject, destination}`、および両者を分ける `type` message attribute|
|`local/attributes.go`|emulator 用の属性集合 — GoAWS が受け付けるものだけ（後述）|
|`testkit/`|contract test 用のクライアントと実行ごとの topic。ARN は常に API 応答から取り、組み立てない|

## port の写像

| seam | SNS / SQS |
| --- | --- |
| wakeup | topic へ body `{"eventId","streamId","sequence"}`（sequence は 10 進文字列）と message attribute `type=wakeup` で `Publish`。body は payload を運ばない: client は必ず EventLog から供給されるので、wakeup が重複しても同じ catch-up 読み出しになる |
| revocation | 同じ topic へ body `{"subject","destination"}` と `type=revocation` で `Publish` |
| `Provision(id)` | `CreateQueue(<prefix>-<id>)` → `GetQueueAttributes(QueueArn)` → `SetQueueAttributes`（`AttributesBuilder` の集合）→ `Subscribe(protocol=sqs, endpoint=queue ARN)` → `SetSubscriptionAttributes(RawMessageDelivery=true)`。同じ `id` に対して冪等で、2 つ目の `id` は `ErrConflict`。途中で失敗した場合は作った分を teardown するので、起動に失敗しても何も残らない。queue 名は決定的なので、orphan-cleanup job は lease だけから queue に辿り着ける。subscription は `ListSubscriptionsByTopic` から queue ARN で探す |
| `Receive(limit)` | `WaitTimeSeconds=20`・`MaxNumberOfMessages=min(limit,10)`・`MessageAttributeNames=[All]` の `ReceiveMessage`。`type` を読めないメッセージは `Kind` が空のまま receipt 付きで返るので、consumer は永遠に再配送させる代わりに削除できる |
| `Delete(n)` | receipt handle による `DeleteMessage` |
| `Teardown()` | `Unsubscribe` → `DeleteQueue`。前段が失敗しても各段を試み、失敗は束ねる — 残ったものは orphan-cleanup job が回収する |
| `Reclaim(id)` | `ListSubscriptionsByTopic`（ページを辿る）→ `Unsubscribe` → `GetQueueUrl` → `DeleteQueue`。順序が `Teardown` と同じ理由も同じで、違うのは instance 自身の状態ではなく識別子から辿る点。削除する queue の名前は、一覧が返した endpoint が持つ実際の名前と、現在の `QueuePrefix` から導ける名前の両方。lease は queue への参照を持たないため、設定から導くだけでは prefix を変えた後に旧世代の残骸へ届かず、しかも「無いものは成功」なので回収は成功を報告し、残骸を辿る唯一の索引である lease が消えてしまう。既に無いものは成功として扱うので、繰り返しても同じ状態に収束する。確認待ちの subscription は ARN の代わりに `PendingConfirmation` を持つため触らない |

`Receive` と `Delete` は「受信先が無い」を通常の失敗と区別します。キャッシュした識別子を捨てて
`ErrReceivingEndGone` を返し（`currentQueueURL` も未 provision のとき同じ sentinel を返します）、**作り直しはしません**。
作り直しは受信先より先に lease を書く順序でなければならず（lease に指されない queue は誰にも回収できない。設計 §2.5）、
その順序を持つのはこの package ではなく両者を合成する側だからです。`internal/controller/realtime` の受信ループがこの
エラーで `Reprovisioner` に依頼し、DI module がそれを `LeaseKeeper.Beat` → `Provision` の順で組み立てます。
「まだ作っていない」を同じ sentinel に畳んでいるのが、ループを修復可能に保つ要点です。作り直しに失敗すると
（AWS は queue の削除から同名 queue の作成まで 60 秒を要求します）手元の状態は空のまま残るので、次の周回が別のエラーへ
落ちて誰も対処しない形にならないよう、同じ修復へ届く必要があります。

固定値は `aws/attributes.go` にあります: visibility timeout 30 秒、long polling 20 秒、
`maxReceiveCount` 5。topic・queue prefix・（任意の）DLQ はデプロイ先依存で、`RealtimeConfig`
（`REALTIME_TOPIC` / `REALTIME_QUEUE_PREFIX` / `REALTIME_DLQ`。最初と最後は AWS では ARN）から来ます。

## エラーの分類

publisher は outbox publisher なので、そのエラーは relay の dead-letter 規則
（[ADR-0058](../../../docs/adr/0058-outbox-dead-on-permanent-error.ja.md)）が読むものです。

| 失敗 | 分類 | 効果 |
| --- | --- | --- |
| payload が `DeliveryEvent` でない、封筒が不正、`eventId` ≠ outbox の `message_id`（`ErrEventIDMismatch`） | `apperror.ErrPermanent` | 行は dead になり、その stream は先頭で止まる（ordering chain、[ADR-0072](../../../docs/adr/0072-postgres-state-dynamodb-eventlog.ja.md)） |
| `realtime.ErrSequenceConflict`（その位置に別の event が既にある） | `ErrPermanent` | 同上 — retry してもその位置は空かない |
| EventLog に到達できない / SNS の `Publish` が失敗 | `ErrRetryable`（+ `ErrUnavailable`） | `next_attempt_at` が先へ進む。retry は冪等に再 append し（同じ `eventId` なら成功）改めて publish する — wakeup が 2 度出るが無害 |

subscription と notifier は、SDK の失敗を `apperror.ErrUnavailable` に正規化します（context の cancel は
`ErrCanceled`）。

## emulator 互換性（`local/`）

GoAWS v0.5.4 に対する `make realtime-smoke`（`scripts/realtime-smoke`）で、fan-out・lifecycle API・
`type` attribute・`ListSubscriptionsByTopic` は wire 互換であり、queue の 4 属性はそうでないことが
分かりました: `Policy` は拒否され（`InvalidParameterValue`）、`RedrivePolicy` は受理されるものの以後
受信したメッセージを削除できなくなり、`SqsManagedSseEnabled` / `KmsMasterKeyId` は受理されるが保存
されません。そのため `local.NewQueueAttributes` は timing の属性だけを設定します。production の builder
は何も落としません — production で policy が欠けているのは吸収すべきものではなく失敗すべき設定ミス
だからです — そして DI module が環境で builder を選びます。

## テスト戦略

この adapter の基盤はデータベースではなく SNS / SQS なので、戦略は
[`internal/infrastructure/README.ja.md`](../README.ja.md) から継承せずここで宣言します。

- すべてのメソッドを、生成した `SNSAPI` / `SQSAPI` の mock に対して単体テストする — `Provision` と
  `Teardown` の呼び出し順序、途中失敗時の rollback、receive のパラメータ、エラー分類の全分岐。これらに
  emulator は要らない。
- `TestInstanceSubscriptionContract` は provision → publish → receive → delete → teardown の往復を GoAWS
  （ローカルでは共有の `goaws` サービス、CI では `go-test` の service container）に対して、`testkit` の
  実行ごとの topic と queue prefix で走らせる。AWS へ向けるのは `REALTIME_TEST_*`（この基盤では
  `REALTIME_TEST_PUBSUB_ENDPOINT`）の問題でしかない。skip は無い: emulator が居なければテストは失敗する
  — DynamoDB の store と同じ規則。
- N 個の subscriber への fan-out と「publish 後の mark 失敗で二重配送しない」シナリオも contract test
  （`aws/fanout_contract_test.go`）にある。publisher と EventLog store の両方が要るため DynamoDB Local と
  GoAWS を並べて走らせる。skip しない規則は同じ。
- 回収も同じ理由で contract test（`aws/cleanup_contract_test.go`）。fan-out の隣に lease store が要るため、
  ここでも DynamoDB Local と GoAWS を並べて走らせる。`TestOrphanCleanupContract` が crash した instance の
  往復・2 つの掃除役が見る競合・繰り返し実行の収束を、`TestReceivingEndGoneContract` が作り直しの経路が
  依存するエラー分類を固定する — mock は `QueueDoesNotExist` を返すよう指示できてしまうので、実際に返すか
  どうかは emulator にしか答えられない。
