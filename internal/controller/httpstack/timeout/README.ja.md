# timeout

リクエスト context に per-request の deadline budget（`SERVER_REQUEST_TIMEOUT`）を設定します。

## 役割

deadline budget の入口です。per-request の deadline をここで1点だけ設定し、`ctx` 経由で下流の全層へ伝播させます——後続の `Use` ミドルウェア、OpenAPI 検証、ハンドラ、DB クエリ（pgx が `ctx` deadline でキャンセル）、外部 HTTP（`httpclient` は既に `ctx.Deadline()` で残予算を尊重）。各境界に独立した timeout ノブを置くのではなく、全層がこの単一予算から deadline を導出します。`statement_timeout` / `lock_timeout` は `ctx` を無視するクエリ向けの粗い backstop であり、主機構ではありません。

response writer のデータ競合を避けるため、echo 標準の race-free な `middleware.ContextTimeout` を基底とします（非推奨の `middleware.Timeout` は競合を抱える）。deadline 超過時は `apperror.ErrUnavailable` を返し、echo 中央の統一 `HTTPErrorHandler` が他のエラーと同じボディ形（HTTP 503）を生成します。

## 予算を張らない唯一の path

SSE の stream endpoint（`/v1/streams/{destination}`）は `streampath.Is` で除外します。stream は数分単位で
開いたままになることが前提で、上限は 1 時間です。そこへ 60 秒の予算を 1 つ張っても制限にはならず、client
も server も決めていない地点で接続を終わらせるだけになります。

除外しても接続が無制限になるわけではありません。`controller/stream` が、長寿命のレスポンスに合った境界で
予算を置き換えます: 書き込みごとに張り直す write deadline、死んだ peer を検知する heartbeat、そして接続の
最大寿命です。この endpoint が拒否するものはすべて、レスポンスを確定する前に拒否します。つまり予算が効いた
であろう範囲には、そもそも意味のあるものが無いということです。

## 補足

- **Pre** ミドルウェア（priority 2、`uri`=1 の直後）として登録し、deadline が全 `Use`・検証・ハンドラ・DB を覆うようにします（priority が小さいほど先に実行されるため `uri`=1 が先行、`timeout`=2 が直後に続く。`internal/di/server/extension/inbound` 参照）。
- timeout 値は `ServerConfig.RequestTimeout()` から注入されます。本パッケージ自体は duration を引数で受け取り、フレームワーク設定に非依存です。
- `ContextTimeout` は deadline 付き `ctx` を供給するだけで、`ctx` を無視するハンドラを強制中断はしません。ハンドラ／ドライバ（pgx, `httpclient`）が `ctx` を尊重することで予算が協調的に enforce されます。
