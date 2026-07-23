# DB スロットプール（worktree 並列開発）

複数の git worktree（および主 checkout）が DB コンテナを衝突なく並列利用するための仕組み。
各 worktree は**スロット**を 1 つリースし、そのスロット専用のホストポートと compose プロジェクトで
DB を起動する。これにより「別 worktree が 5432 を握っていて `make test` が動かせない」を解消する。

## 仕組み

- **スロット N** = compose プロジェクト `gobp-db-slot-N` + ホスト公開ポート `5432+N`（既定 MAX 8 = 5432–5439）。
- **リース** = ホスト上のロックディレクトリ `${GOBP_DB_POOL_DIR:-$TMPDIR/gobp-db-pool}/slot-N.lock`
  を `mkdir` の原子性で確保。`meta`（owner=worktree 絶対パス / heartbeat / branch）を書く。
- **占有情報** = acquire が worktree ルートに `.gobp-db-slot`（gitignore）を書き出す。
  `make` が `-include` して `DB_HOST_PORT` / `COMPOSE_PROJECT_NAME` を全ターゲットへ伝播する。
- **割り当てるホスト公開ポート**（スロット N、既定）:
  - `DB_HOST_PORT` = `5432+N`（DB。host 実行の `go test` 接続先。`internal/config` のテスト設定が参照）
  - `API_HOST_PORT` = `8080+N`（`make serve` の API。ホストから curl する先）
  - `MOCK_AUTH_HOST_PORT` = `4000+N`（`make serve` の mock 認証サーバー）
  - これにより複数 worktree で `make serve` を同時起動し、各 API を別ポートで curl できる。
    o11y / dlv / pprof は現状ずらしていない（並列で o11y も要る場合は別途拡張）。
- **ポートの二重性**（重要）:
  - `*_HOST_PORT`: **ホスト公開ポート**。compose の publish と、ホスト側からの接続先。
  - **コンテナ内部ポート**（DB=`5432` / API=`8080` / mock_auth=`4000` 固定、`env/.env` 等由来）:
    サービス間はこの内部ポート + サービス名で繋ぐ（`database:5432` 等）。`*_HOST_PORT` とは別物。
    内部ポート（例 `DB_PORT`）を export するとコンテナ内アプリがホスト公開ポートへ繋ぎに行き接続不能になる。
- **スキーマ安全性**: acquire は取得後に `db-local-reinit` / `db-test-reinit`（drop→migrate→seed）で
  自ブランチのスキーマへ作り直す。別ブランチが使ったスロットを引き継いでも安全。
- **解放**: `db-release` で lease と `.gobp-db-slot` を削除。コンテナは warm 保持で次に貸す。
- **stale 回収**: heartbeat が TTL（既定 1800 秒、`GOBP_DB_POOL_TTL`）超過したリースは acquire 時に
  別 worktree が再取得できる（crash した worktree がスロットを握り続けない）。

## 使い方

```sh
make db-acquire      # 空きスロットをリースし DB 起動 + スキーマ再構築（この後 test/serve は同一スロット）
make test            # ホストから DB_HOST_PORT 経由で接続
make serve           # API_HOST_PORT / MOCK_AUTH_HOST_PORT で起動 → curl localhost:$API_HOST_PORT
make db-pool-status  # スロット占有状況（DB / API ポート）を表示
make db-release      # スロットを解放（コンテナは warm 保持）
```

`make db-acquire` を実行しなければ、従来どおりホスト既定ポート（DB 5432 / API 8080 / mock_auth 4000）と
ディレクトリ由来プロジェクトで単独動作する（プールは opt-in）。

## 環境変数

| 変数 | 既定 | 意味 |
| --- | --- | --- |
| `GOBP_DB_POOL_DIR` | `$TMPDIR/gobp-db-pool` | リースレジストリの置き場所 |
| `GOBP_DB_POOL_BASE` | `5432` | スロット 0 のホストポート |
| `GOBP_DB_POOL_MAX` | `8` | スロット数（=同時並列数の上限） |
| `GOBP_DB_POOL_TTL` | `1800` | stale 判定の heartbeat 猶予（秒） |

## 注意

- 主 checkout の従来 DB（プール外のコンテナ）がポートを握っている場合、acquire はその slot を
  `foreign busy` として skip し次の空きスロットを取る。全 worktree をプールへ寄せると skip は起きない。
- `docker/`・`scripts/`・`.makefiles/` を含む配線のため、変更時はこのドキュメントも更新すること。
