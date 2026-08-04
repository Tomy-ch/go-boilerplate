# db-slot

[English](README.md) | 日本語

単一の共有インフラ上で per-worktree のスロットをリースし、複数の `git worktree` がホストポート衝突なく DB を使う並列作業を可能にする。各スロットは共有 DB コンテナ（固定 infra compose プロジェクト `gobp-shared`・ホスト 5432）内の専用データベース（`wt<N>_local` / `wt<N>_test`）と、ホスト公開ポートをスロット番号で相対化した専用の app 層 compose プロジェクト（`gobp-wt-N`）を占有する。スロットを取らない checkout も同じ共有インフラへ繋がるため、スロット取得は**並列作業を衝突なく行うための opt-in** に留まる。全体像は `docs/maintenance/db-worktree-pool.md` を参照。

**ホスト**で実行する（tool-runner コンテナ内ではない）: ホストのファイルシステム上のリースを管理し、ホストの `docker compose` を駆動し、共有 DB へは pgx で `localhost:5432` へ接続するため。

## コマンド

```text
db-slot acquire     # 空きスロットをリースし wt<N> DB を作成/設定して .gobp-db-slot を書き出す
db-slot release     # スロットの app コンテナを停止しリースを解放（DB は warm 保持）
db-slot heartbeat   # 保持スロットのリース heartbeat を更新
db-slot status      # スロット占有状況を表示
```

通常は make ラッパ（`make slot-acquire` / `slot-free` / `slot-release` / `slot-status`）を使う。`slot-acquire` はリースした DB のスキーマ再構築も行い、`slot-release` は worktree ごと撤収する。

## 設計

- **リース**（`Registry`）— `~/.cache/gobp-db-pool`（`GOBP_DB_POOL_DIR` で上書き可）配下のスロット毎ロックディレクトリ。`os.Mkdir` の原子性で新規取得、stale 回収は `rename` で原子的に占有権を奪い、`flock` の走査ロックで acquire ループ全体を直列化して 2 worktree が同一スロットを二重リースしないようにする。symlink を向いた pool dir は拒否（先読み攻撃対策）、meta は `0600`。
- **DB 管理**（`DBAdmin` / `PgxAdmin`）— 各 `wt<N>` DB への `CREATE DATABASE` ＋ `pg_trgm` 拡張設定、および stale スロットを reclaim してよいかの判断に使う `pg_stat_activity` の接続確認。timezone は含まない: `database` コンテナの `TZ` がクラスタ既定であり、後から作った DB もそれを継承する。
- **使用中判定** — stale スロットを reclaim する前に、app プロジェクト（`gobp-wt-N`）の稼働中コンテナと、その DB への接続の両方を確認する。接続プールはアイドルで空になるため接続確認だけでは serve 中の worktree を見落とし、コンテナ確認だけではホスト実行の `go test` を捉えられない。
- **Compose**（`Compose` / `ExecCompose`）— 共有 DB を固定 infra プロジェクトで起動（`--wait --no-recreate`）し、release 時にスロットの app プロジェクト（`gobp-wt-N`）を `docker-compose.yaml` + `docker-compose.attach.yaml` で停止する。`--no-recreate` は、他の checkout が使っているコンテナを `acquire` が置き換えないためのもの。compose の config ハッシュは bind mount の絶対パスを含むため、同じコミットでも worktree が違えば一致しない。make 側の `INFRA_NO_RECREATE` と違いここでは条件を持たない。スロットを取ること自体が、共有インフラを他の checkout と分け合うという宣言だからである。
- **スロットファイル** — `acquire` が `.gobp-db-slot`（gitignore・`make` が `-include` で読む `KEY=VALUE`）を書き出し、`.makefiles/docker/compose.mk` の既定値を上書きする。内容は `SLOT`、`DB_NAME_LOCAL` / `DB_NAME_TEST`、スロット相対のホスト公開ポート `API_HOST_PORT` / `MOCK_AUTH_HOST_PORT` / `DLV_HOST_PORT` / `PPROF_HOST_PORT`、`COMPOSE_PROJECT_NAME`（共有 infra プロジェクト）、`SERVE_PROJECT`（`gobp-wt-N`＝app 層プロジェクト）。
- **env ガード** — `APP_ENV` が deploy 系（`dev` / `stg` / `prd`）のときは実行を拒否する。pool は DB を作成/破棄するため dev/test 専用ツールに留める（`config.IsLocalClassEnv`）。

## Test Strategy

上位層の Testing Policy は、判断ロジックをダブルに対してテストできるよう、依存をすべて seam の裏へ押し出す。ここでも判断ロジックはそれに従う — ただし seam の実装自体はアダプタであり、アダプタは実物を駆動して初めて価値を持つ。したがって各コンポーネントは、その subject が実際に存在する層でテストする。

- **`Pool`**（判断ロジック） — 生成 mock（`MockDBAdmin` / `MockCompose`）に対する単体テスト。Postgres にも docker にも到達しない。acquire / release / reclaim の全分岐をここで固定する。「片方だけでは取りこぼす」ことこそが要点である 2 段構えの in-use 判定も含む。
- **`Registry`** — `t.TempDir()` 上の実ファイルシステムプリミティブ。subject が `os.Mkdir` / `os.Rename` の**原子性そのもの**なので、FS を fake にしても何も証明できない。二重リースはまさに fake が覆い隠してしまう類の欠陥である。
- **`ExecCompose`** — `PATH` の先頭に仕込んだスタブ `docker` が、組み立てられた引数列と `COMPOSE_PROJECT_NAME` を記録する。実 compose を走らせずにコマンド構築と環境変数注入を固定する。`PATH` への `t.Setenv` を使うため、これらのケースは `t.Parallel()` と両立しない。
- **`PgxAdmin`** — `DBAdmin` の唯一の実装。共有 Postgres（`localhost:5432`）に対してテストする。その SQL が実際に実行できることを証明する唯一の網である。到達不能ホストのケースはサーバ無しでエラー経路を固定し、作成した DB は cleanup で drop するので実行を繰り返せる。

基準はパッケージではなく subject にある。契約が**判断**であるコンポーネントはダブルに対して、契約が**外部基盤の振る舞い**であるコンポーネントはその基盤に対してテストする。結果としてここのテストは純粋な単体テストのパッケージより遅く、共有 infra の起動を必要とする。

## 環境変数

|変数|既定|説明|
|---|---|---|
|`GOBP_DB_POOL_DIR`|`~/.cache/gobp-db-pool`|リースレジストリの置き場所|
|`GOBP_DB_SHARED_PROJECT`|`gobp-shared`|共有インフラの固定 compose プロジェクト|
|`GOBP_DB_POOL_MAX`|`12`|スロット数（＝同時並列 worktree の上限）|
|`GOBP_DB_POOL_TTL`|`1800`|heartbeat stale 判定の猶予（秒）|
|`GOBP_API_POOL_BASE` / `GOBP_MOCK_AUTH_POOL_BASE`|`8080` / `4000`|API / mock 認証サーバーのホストポートのベース（スロット N = ベース + N）|
|`GOBP_DLV_POOL_BASE` / `GOBP_PPROF_POOL_BASE`|`2345` / `6060`|dlv デバッグ / pprof のホストポートのベース（スロット N = ベース + N）|

## 注意

- **dev/test 専用。** `dev` / `stg` / `prd` では実行を拒否する。
- スロットを取らない checkout も同じ共有インフラへ繋がる（既定の `local` / `test` データベースと既定のホスト公開ポートを使うだけ）。衝突なく並列作業したいときにだけスロットを取る。
