# DB スロットプール（worktree 並列開発）

複数の git worktree（および主 checkout）が **単一の共有 Postgres（ホスト 5432）** を衝突なく並列利用する
ための仕組み。各 worktree は**スロット**を 1 つリースし、そのスロット専用の**データベース**
（`wt<N>_local` / `wt<N>_test`）を共有 DB 内に持つ。worktree の分離を「別コンテナ・別ポート」ではなく
「同一コンテナ内の別データベース」で行うことで、DB 軸のホストポート割当が不要になり、「別 worktree が
5432 を握っていて `make test` が動かせない」を解消する。

## 仕組み

- **共有 DB** = 固定 compose プロジェクト `gobp-shared` に 1 個だけ起動する `database` コンテナ
  （ホスト 5432）。worktree はディレクトリ毎に compose の既定プロジェクト名が変わるため、共有 DB は
  明示の固定名（`GOBP_DB_SHARED_PROJECT`）に固定する。
- **スロット N** = 共有 DB 内のデータベース名ペア `wt<N>_local` / `wt<N>_test`（既定 MAX 8 = wt1〜wt8）。
- **実装** = ホスト実行の Go CLI `cmd/db-slot`（コアは `internal/cli/dbslot`）。リース判定・DB 作成・
  compose 起動をテスト可能な形で担う。make ターゲットは `go run ./cmd/ db-slot <sub>` を呼ぶ。
- **リース** = ホスト上のロックディレクトリ `${GOBP_DB_POOL_DIR:-~/.cache/gobp-db-pool}/slot-N.lock`。
  新規取得は `os.Mkdir` の原子性、stale 回収は `rename` で原子的に占有権を奪い、acquire の走査全体を
  `flock` で直列化するため、2 worktree が同一スロットを二重リースしない。`meta`（owner / heartbeat /
  branch）は `0600`、symlink を向いた pool dir は先読み攻撃対策として拒否する。
- **実行環境ガード** = `APP_ENV` が deploy 系（dev/stg/prd）のとき db-slot は実行を拒否する（DB を
  作成/破棄するため dev/test 専用）。`internal/config` のテスト設定も同様に deploy 系では DB_NAME 上書きを
  無視する。
- **占有情報** = acquire が worktree ルートに `.gobp-db-slot`（gitignore）を書き出す。`make` が
  `-include` して以下を全ターゲットへ伝播する:
  - `COMPOSE_PROJECT_NAME` = `gobp-shared`（DB ツーリング migrate/seed/psql/gen が共有 DB を指す）
  - `DB_NAME_LOCAL` / `DB_NAME_TEST` = `wt<N>_local` / `wt<N>_test`（host 実行の `go test` は
    共有 DB の localhost:5432 経由でこの名前の自 worktree DB へ繋ぐ。`internal/config` のテスト設定が参照）
  - `API_HOST_PORT` = `8080+N` / `MOCK_AUTH_HOST_PORT` = `4000+N`（`make serve` の並列化用ポート）
  - `SERVE_PROJECT` = `gobp-wt-N`（プール serve で app コンテナを分離するプロジェクト）
- **拡張・timezone のブートストラップ**: acquire は `wt<N>_local` / `wt<N>_test` を CREATE DATABASE
  （存在ガード）した後、各 DB に `pg_trgm` 拡張と `Asia/Tokyo` timezone を設定する（init スクリプトが
  `local` / `test` に施すのと同じもの。動的に作る worktree DB には明示設定が必要）。
- **スキーマ安全性**: acquire は取得後に `wt<N>_local` / `wt<N>_test` を drop→migrate→seed で
  自ブランチのスキーマへ作り直す。別ブランチが使ったスロットを引き継いでも安全。
- **serve の分離**: `make serve` はプール取得時、app コンテナ（api_server / mock_auth）を
  per-worktree プロジェクト `gobp-wt-N` に分離し、`docker-compose.pool.yaml` override で共有 DB を
  `host.docker.internal:5432` 経由で参照して `wt<N>_local` へ繋ぐ（`docker-compose.pool.yaml` 冒頭参照）。
- **解放**: `db-release` は、このスロットの serve プロジェクト（`gobp-wt-N`）の app コンテナを停止して
  から lease と `.gobp-db-slot` を削除する。データベースは warm 保持で次に貸す。
- **stale 回収の安全化**: heartbeat が TTL（既定 1800 秒、`GOBP_DB_POOL_TTL`）超過したリースは acquire 時に
  別 worktree が再取得できる（crash した worktree がスロットを握り続けない）。ただし回収前に
  `pg_stat_activity` でそのスロットの DB に稼働中接続があるか確認し、あれば（heartbeat 更新漏れの保険）
  破壊せず skip する。serve 中は `make serve` が heartbeat を打つ。

## 使い方

```sh
make db-acquire      # 空きスロットをリースし共有 DB 起動 + 自 worktree DB を作成/再構築
make test            # ホストから localhost:5432 経由で wt<N>_test へ接続
make serve           # app を gobp-wt-N に分離起動 → curl localhost:$API_HOST_PORT（DB は共有の wt<N>_local）
make db-pool-status  # スロット占有状況（DB 名 / API ポート）を表示
make db-release      # スロットを解放（データベースは warm 保持）
```

`make db-acquire` を実行しなければ、従来どおり既定プロジェクト / DB 名 `local`・`test` で単独動作する
（プールは opt-in）。ただしプール運用に入ると DB の住処は各 checkout の既定プロジェクトから共有
`gobp-shared` へ移るため、プールと従来の per-checkout DB は同じホスト 5432 上で排他となる。

## 環境変数

| 変数 | 既定 | 意味 |
| --- | --- | --- |
| `GOBP_DB_POOL_DIR` | `~/.cache/gobp-db-pool` | リースレジストリの置き場所（symlink は拒否） |
| `GOBP_DB_SHARED_PROJECT` | `gobp-shared` | 共有 DB コンテナの固定 compose プロジェクト名 |
| `GOBP_API_POOL_BASE` | `8080` | `API_HOST_PORT` のベース（スロット N = ベース+N） |
| `GOBP_MOCK_AUTH_POOL_BASE` | `4000` | `MOCK_AUTH_HOST_PORT` のベース |
| `GOBP_DB_POOL_MAX` | `8` | スロット数（=同時並列数の上限） |
| `GOBP_DB_POOL_TTL` | `1800` | stale 判定の heartbeat 猶予（秒） |

## 注意

- **共有インスタンスの blast radius**: 全 worktree が 1 個の Postgres を共有する。データベースは分離
  されるため DDL は DB を跨げないが、reinit の対象 DB 名を取り違えると他 worktree を壊しうる。
- **プール serve の制約**: observability(3000/4317/4318/3200) と dlv/pprof(2345/6060) は固定ポートで、
  プール serve では observability を起動対象から外す（OTLP は best-effort）。dlv/pprof は publish しない。
- API 帯 8080–8087 と被らないよう `sql_editor` / `docs_viewer` は 7000 番台へ退避済み。
- `docker/`・`internal/cli/dbslot`・`.makefiles/` を含む配線のため、変更時はこのドキュメントも更新すること。
