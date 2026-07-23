# db-pool

[English](README.md) | 日本語

単一の共有 Postgres 上で per-worktree のスロットをリースし、複数の `git worktree` がホストポート衝突なく DB を使う並列作業を可能にする。各スロットは 1 つの共有 DB コンテナ（固定 compose プロジェクト `gobp-shared`・ホスト 5432）内の専用データベース（`wt<N>_local` / `wt<N>_test`）を占有する。全体像は `docs/maintenance/db-worktree-pool.md` を参照。

**ホスト**で実行する（tool-runner コンテナ内ではない）: ホストのファイルシステム上のリースを管理し、ホストの `docker compose` を駆動し、共有 DB へは pgx で `localhost:5432` へ接続するため。

## コマンド

```text
db-pool acquire     # 空きスロットをリースし wt<N> DB を作成/設定して .gobp-db-slot を書き出す
db-pool release     # スロットの serve コンテナを停止しリースを解放（DB は warm 保持）
db-pool heartbeat   # 保持スロットのリース heartbeat を更新
db-pool status      # スロット占有状況を表示
```

通常は make ラッパ（`make db-acquire` / `db-release` / `db-pool-status`）を使う。`db-acquire` はリースした DB のスキーマ再構築も行う。

## 設計

- **リース**（`Registry`）— `~/.cache/gobp-db-pool`（`GOBP_DB_POOL_DIR` で上書き可）配下のスロット毎ロックディレクトリ。`os.Mkdir` の原子性で新規取得、stale 回収は `rename` で原子的に占有権を奪い、`flock` の走査ロックで acquire ループ全体を直列化して 2 worktree が同一スロットを二重リースしないようにする。symlink を向いた pool dir は拒否（先読み攻撃対策）、meta は `0600`。
- **DB 管理**（`DBAdmin` / `PgxAdmin`）— 各 `wt<N>` DB への `CREATE DATABASE` ＋ `pg_trgm` 拡張 ＋ `Asia/Tokyo` timezone 設定、および `pg_stat_activity` の接続確認で、稼働中の DB を持つスロットの reclaim を防ぐ。
- **Compose**（`Compose` / `ExecCompose`）— 共有 DB を起動（`--wait`）し、release 時に worktree の serve プロジェクト（`gobp-wt-N`）を停止する。
- **env ガード** — `APP_ENV` が deploy 系（`dev` / `stg` / `prd`）のときは実行を拒否する。pool は DB を作成/破棄するため dev/test 専用ツールに留める（`config.IsLocalClassEnv`）。

## 環境変数

|変数|既定|説明|
|---|---|---|
|`GOBP_DB_POOL_DIR`|`~/.cache/gobp-db-pool`|リースレジストリの置き場所|
|`GOBP_DB_SHARED_PROJECT`|`gobp-shared`|共有 DB の固定 compose プロジェクト|
|`GOBP_DB_POOL_MAX`|`8`|スロット数（＝同時並列 worktree の上限）|
|`GOBP_DB_POOL_TTL`|`1800`|heartbeat stale 判定の猶予（秒）|
|`GOBP_API_POOL_BASE` / `GOBP_MOCK_AUTH_POOL_BASE`|`8080` / `4000`|ホストポートのベース（スロット N = ベース + N）|

## 注意

- **dev/test 専用。** `dev` / `stg` / `prd` では実行を拒否する。
- プール運用に入ると DB の住処が各 checkout の既定プロジェクトから `gobp-shared` へ移る。両者はホスト 5432 上で排他。
