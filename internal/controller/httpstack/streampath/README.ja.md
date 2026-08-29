# streampath

`Is(path)` — そのリクエストが SSE の stream endpoint（`GET /v1/streams/{destination}`）かどうかを、
`/v1/streams/` prefix で判定します。リクエスト / レスポンス型のトラフィックを前提に書かれたミドルウェアが、
1 時間開いたままになるレスポンスに対して身を引けるようにするために存在します。

## 何が参照し、なぜ参照するのか

| 利用側 | 除外しない場合 |
| --- | --- |
| `timeout` | `SERVER_REQUEST_TIMEOUT`（60 秒）がリクエスト context をキャンセルするため、すべての stream が 1 分で切られる |
| `logging` | レスポンス完了時にアクセスログを 1 行書く — 1 時間遅れで、しかも所要時間はリクエストではなく接続を測った値になる |
| `redmetrics` | RED の histogram がその 1 時間をリクエストのレイテンシとして取り込み、API を評価するすべての percentile を歪める |

代わりに置かれるのは接続自身の境界です: 書き込みごとの deadline（10 秒）、heartbeat、そして接続の最大寿命。
いずれも `controller/stream` が持ちます。

## 参照してはならないもの

**`oapi/skipper`**、および OpenAPI 検証全般です。stream endpoint の `ticket` を検証するのは `StreamTicket`
security scheme であり、それを走らせるのは OpenAPI validator です。この path で検証をスキップすれば、
すべての接続が未認証のまま通ります。この除外を `ops.IsOpsPath` にもう 1 項目足すのではなく別の述語にしている
のはこのためです。`ops` のほうは `oapi/skipper` も参照しますし、ops パスが検証を免除されているのは、そもそも
OpenAPI の定義を持たないからです。stream endpoint は定義を持ち、それを効かせる必要があります。
