# streampath

述語は 2 つあります。リクエスト / レスポンス型のトラフィックを前提に書かれたミドルウェアが、1 時間開いたままに
なるレスポンスに対して身を引けるようにするため — そして、ついに接続にならなかった拒否に対してまで身を引いて
しまわないためです。

- `Is(path)` — そのリクエストが SSE の stream endpoint（`GET /v1/streams/{destination}`）を狙ったものかどうかを、
  `/v1/streams/` prefix で判定します。handler が走る**前**に答えられます。
- `IsCommittedStream(header)` — そのレスポンスが実際に stream として確定したかどうかを、`text/event-stream` の
  content type で判定します。答えられるのは**後**だけです。

## 何がどちらを参照し、なぜ参照するのか

| 利用側 | 述語 | 除外しない場合 |
| --- | --- | --- |
| `timeout` | `Is` | `SERVER_REQUEST_TIMEOUT`（60 秒）がリクエスト context をキャンセルするため、すべての stream が 1 分で切られる。予算は handler が走る前に決まるので、これだけは path で判定するしかない |
| `logging` | `IsCommittedStream` | 接続が閉じたときにレスポンスのログ行を書く — 1 時間遅れで、しかも所要時間はリクエストではなく接続を測った値になる。*リクエスト* のログ行は接続時にそのまま出る |
| `redmetrics` | `IsCommittedStream` | RED の histogram がその 1 時間をリクエストのレイテンシとして取り込み、API を評価するすべての percentile を歪める |

リクエストの予算の代わりに置かれるのは接続自身の境界です: 書き込みごとの deadline（10 秒）、heartbeat、そして
接続の最大寿命。いずれも `controller/stream` が持ちます。そこから出る store への呼び出しは
`dynamodbclient.CallTimeout` で上限が付きます。リクエストの予算がもうそれらを覆わないためです。

**観測系の 2 つが path ではなくレスポンスを見る理由。** stream endpoint は 401・400・410・503 で接続を拒否し、
それらはミリ秒で終わる通常のレスポンスです。path で除外すれば、まさに見るべき信号が消えます — ticket の
総当たりは 401 のバーストとして、容量の枯渇は 503 のバーストとして現れるのに、そのどちらもログにもメトリクスにも
出ません。除外の理由（「接続の長さはリクエストのレイテンシではない」）が当てはまるのは、接続が存在してからです。

## 参照してはならないもの

**`oapi/skipper`**、および OpenAPI 検証全般です。どちらの述語もそこには置けません。stream endpoint の `ticket` を検証するのは `StreamTicket`
security scheme であり、それを走らせるのは OpenAPI validator です。この path で検証をスキップすれば、
すべての接続が未認証のまま通ります。この除外を `ops.IsOpsPath` にもう 1 項目足すのではなく別の述語にしている
のはこのためです。`ops` のほうは `oapi/skipper` も参照しますし、ops パスが検証を免除されているのは、そもそも
OpenAPI の定義を持たないからです。stream endpoint は定義を持ち、それを効かせる必要があります。
