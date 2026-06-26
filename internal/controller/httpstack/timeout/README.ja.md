# timeout

[English](README.md) | 日本語

リクエスト context に per-request の deadline budget（`SERVER_REQUEST_TIMEOUT`）を設定します。

## 役割

deadline budget の入口です。per-request の deadline をここで1点だけ設定し、`ctx` 経由で下流の全層へ伝播させます——後続の `Use` ミドルウェア、OpenAPI 検証、ハンドラ、DB クエリ（pgx が `ctx` deadline でキャンセル）、外部 HTTP（`httpclient` は既に `ctx.Deadline()` で残予算を尊重）。各境界に独立した timeout ノブを置くのではなく、全層がこの単一予算から deadline を導出します。`statement_timeout` / `lock_timeout`（将来）は `ctx` を無視するクエリ向けの粗い backstop であり、主機構ではありません。

response writer のデータ競合を避けるため、echo 標準の race-free な `middleware.ContextTimeout` を基底とします（非推奨の `middleware.Timeout` は競合を抱える）。deadline 超過時は `apperror.ErrUnavailable` を返し、echo 中央の統一 `HTTPErrorHandler` が他のエラーと同じボディ形（HTTP 503）を生成します。

## 補足

- **Pre** ミドルウェア（priority 2、`uri`=1 の直後）として登録し、deadline が全 `Use`・検証・ハンドラ・DB を覆うようにします（priority が小さいほど先に実行されるため `uri`=1 が先行、`timeout`=2 が直後に続く。`internal/di/server/extension/inbound` 参照）。
- timeout 値は `ServerConfig.RequestTimeout()` から注入されます。本パッケージ自体は duration を引数で受け取り、フレームワーク設定に非依存です。
- `ContextTimeout` は deadline 付き `ctx` を供給するだけで、`ctx` を無視するハンドラを強制中断はしません。ハンドラ／ドライバ（pgx, `httpclient`）が `ctx` を尊重することで予算が協調的に enforce されます。
