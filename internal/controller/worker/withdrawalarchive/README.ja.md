# withdrawal-archive worker（サンプル）

[English](README.md) | 日本語

outbox 経路の**消費側**を一通り動かして見せる実例です。ユーザーが退会すると、outbox が同一トランザ
クションで `user.withdrawn.v1` を emit し、relay がそれを broker へ publish し、この worker が消費して
退会証跡をオブジェクトストレージへ書き出します。

削除可能なサンプル一式の一部です。`make setup-remove-sample-api` はこのパッケージと、ここへ到達する
2 行の登録を削除し、`provideWorkers()` を再び空へ戻します。

## 何を証跡として残すか、なぜそれなのか

保存するオブジェクトは**イベント payload そのもの**（`{userId, deletedAt}`）で、
`withdrawals/{userID}.json` に置きます。

退会したユーザーのレコード一式を書き出す案は自明な代替でしたが、成立しません。退会済みユーザーは
ユーザー検索クエリの `deleted_at IS NULL` で除外されるため、消費側から読み戻せないからです。payload を
保存する形ならそれを丸ごと回避できますし、そもそも設計として筋が通ります — 残るのは「誰がいつ退会
したか」であり、`user-purge` がユーザーを物理削除した後も監査証跡として意味を保ち、それ自体は
purge の対象になるような個人情報を含みません。

## 冪等性

at-least-once 配信である以上、この Handler は 1 回の退会に対して 2 回走ることがあります。再実行を
検出するのではなく、**操作自体を冪等に**しています。キーはユーザー ID だけから決まり、本文は payload を
加工しないため、2 回目の実行は同一のバイト列で上書きします。

だからここでは冪等性ストアを参照しません。HTTP の idempotency サブシステムは HTTP レスポンスの
再生のために作られており、そのレコードは method / path / レスポンスステータスを持ちます。ここで
借りると、それらにダミー値を詰めることになります。

同じ性質が、Handler の処理が長引いたことによる再配送も引き受けます。実行が
`CONSUMER_QUEUE_VISIBILITY_TIMEOUT`（既定 30 秒、`WORKER_EXTEND_INTERVAL` のハートビートは既定で
無し）を超えるとメッセージは再配送されて再度処理されますが、結果は変わりません。

## メッセージの選別

1 つのキューには outbox が publish する全種別が流れるため、Handler はまず `event_type` 属性を確認し、
それ以外には成功を返します — engine がそれを ack し、メッセージはキューから消えます。他人のイベントを
永久失敗として扱うと、購入イベントが軒並み DLQ へ送られます。`event_type` 属性を持たないメッセージも
同じ扱いにするため、属性が存在する前に publish されたメッセージが DLQ を埋めることもありません。

これが成立するのは、サンプルが `gobp-events` の唯一の consumer だからです。1 つのキューに複数の
consumer がいるデプロイでは、代わりにサブスクリプションフィルタ（またはイベント種別ごとのキュー）を
使い、各 consumer が自分の扱う分だけを受け取るようにします。

## エラーの分類

| 状況 | 分類 | 効果 |
| --- | --- | --- |
| 他種別 / `event_type` 欠落 | 成功 | ack され、証跡は書かれない |
| payload を復元できない | `ErrPermanent` | DLQ へ退避してから ack |
| ユースケースが入力を拒否（`ErrValidation`） | `ErrPermanent` | DLQ へ退避してから ack |
| ストレージが利用不能 | 未分類 | engine 既定（retryable）→ backoff 付きで再配送 |

なお broker 側にも redrive policy があり（`docker/elasticmq/elasticmq.conf` の
`maxReceiveCount = 5`）、*retryable* な理由で失敗し続けたメッセージも受信回数を使い切れば DLQ へ
入ります。これは `FailureHandler` の下にあるもう一段の broker レベルの網であって、上の表と矛盾する
ものではありません。

## 一気通貫で動かす

以下は worktree で DB スロットを取得済み（`make slot-acquire`）である前提です。メイン checkout では
既定のポートになります。

```bash
# 1. 共有インフラ + API（elasticmq と garage も一緒に上がる）
make serve

# 2. 別端末: outbox 行をキューへ publish する relay
make outbox-relay

# 3. 別端末: この worker
make worker NAME=withdrawal-archive

# 4. ユーザーを退会させる（トークンの取得は docs/get-started を参照）
curl -X DELETE "http://localhost:${API_HOST_PORT:-8080}/v1/users/<userId>" \
  -H "Authorization: Bearer <token>"
```

追跡するポイントは順に:

1. `api_server` — 退会リクエストが完了し、同一トランザクションで outbox 行が書かれる
2. `outbox-relay` — `user.withdrawn.v1` の publish ログが出て、行が `published` へ遷移する
3. この worker — publisher の `traceparent` から継続されたトレースで Handler が走る
4. オブジェクトストレージ — 証跡オブジェクトが現れる:

   ```bash
   docker run --rm --network host amazon/aws-cli s3 ls s3://gobp-local/withdrawals/ \
     --endpoint-url http://localhost:3900 --region us-east-1
   ```

   （資格情報は `env/.env` の `OBJECT_STORAGE_ACCESS_KEY_ID` / `..._SECRET_ACCESS_KEY`）

冪等性を確かめるには同じメッセージをもう一度 publish します — 同じバイト列でオブジェクトが書き直され
ます。DLQ 経路を確かめるには、`event_type` 属性を `user.withdrawn.v1` にしたまま JSON として不正な
本文を送ります。`failure_reason=permanent` 付きで `gobp-events-dlq` に入ります。

> キューは全 checkout で共有され、worktree 単位では分離できません
> （[`docs/ja/maintenance/db-worktree-pool.ja.md`](../../../../docs/ja/maintenance/db-worktree-pool.ja.md) を参照）。
> 2 つの worktree で同時にこの worker を動かすと、どちらがメッセージを取るかは決まりません。

## 構成

| ファイル | 内容 |
| --- | --- |
| `withdrawal_archive_worker.go` | `worker.Worker` — 名前 / consumer / handler / failure handler を束ねる |
| `withdrawal_archive_handler.go` | `worker.Handler` — 選別・復元・エラー分類 |

broker adapter 自体はここではなく `internal/di/module/withdrawalarchive.go` で組み立てられます。
controller 層は infrastructure を import できないため、このパッケージはどの broker から消費して
いるかを最後まで知りません。
