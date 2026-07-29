# Shared Infra and the DB Slot Pool (parallel worktree development)

English: [db-worktree-pool.md](../../maintenance/db-worktree-pool.md)

複数の git worktree（および主 checkout）が **単一の共有インフラ** を衝突なく並列利用するための仕組み。
compose のサービスを 2 層に分ける:

- **infra 層** — 固定ポートでしか動けないサービス（`database` 5432 / `observability` 3000・4317・4318・3200 /
  `garage` 3900・3903）。固定 compose プロジェクト `gobp-shared` に **全 checkout で 1 インスタンスだけ**置く。
- **app 層** — checkout 毎に要る `api_server` / `mock_auth_server`。checkout 毎の compose プロジェクトへ分離し、
  ホスト公開ポートをずらして並列起動する。

DB の worktree 分離は「別コンテナ・別ポート」ではなく「同一インスタンス内の別データベース」
（`wt<N>_local` / `wt<N>_test`）で行う。これにより DB 軸のホストポート割当が不要になり、「別 worktree が
5432 を握っていて `make test` が動かせない」も「主 checkout の serve が worktree の DB と衝突する」も起きない。
o11y は共有が利点になる（全 checkout のトレース / メトリクス / ログが 1 つの Grafana に集まる）。

## 仕組み

- **infra 層** = 固定 compose プロジェクト `gobp-shared`（`GOBP_DB_SHARED_PROJECT`）。worktree はディレクトリ毎に
  compose の既定プロジェクト名が変わるため、共有側は明示の固定名に固定する。`make infra-up` で起動し、
  `make serve` / `make job` / `make worker` も冒頭で冪等に呼ぶ。
- **app 層** = `APP_PROJECT`。スロット未取得なら `gobp-app-<ディレクトリ名>`、取得時は `gobp-wt-N`。
  `docker-compose.attach.yaml` を重ね、共有インフラを `host.docker.internal` のホスト公開ポート経由で参照する
  （`DB_HOST` / `OBS_OTLP_ENDPOINT` / `OBJECT_STORAGE_ENDPOINT` を実行時 env で上書きする。`loader.go` は
  実行時 env を `env/.env` より優先する）。
- **スロット N** = 共有 DB 内のデータベース名ペア `wt<N>_local` / `wt<N>_test`（既定 MAX 12 = wt1〜wt12）。
  スロットを取らない checkout は既定の `local` / `test` をそのまま使うため、スロット取得は**並列作業のための
  opt-in** に留まる。
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
  `-include` して `.makefiles/docker/compose.mk` の既定値を上書きし、全ターゲットへ伝播する:
  - `DB_NAME_LOCAL` / `DB_NAME_TEST` = `wt<N>_local` / `wt<N>_test`（既定 `local` / `test`。host 実行の
    `go test` は共有 DB の localhost:5432 経由でこの名前の自 worktree DB へ繋ぐ。`internal/config` のテスト設定が参照）
  - `API_HOST_PORT` = `8080+N` / `MOCK_AUTH_HOST_PORT` = `4000+N` / `DLV_HOST_PORT` = `2345+N` /
    `PPROF_HOST_PORT` = `6060+N`（app 層のホスト公開ポートは全てスロット番号で相対化する）
  - `SERVE_PROJECT` = `gobp-wt-N`（app 層の compose プロジェクト = `APP_PROJECT`）
  - `COMPOSE_PROJECT_NAME` = `gobp-shared`（DB ツーリング migrate/seed/psql/gen が共有インフラのネットワークで
    動くよう既定プロジェクトを infra 層へ寄せる。未取得時も compose.mk が同じ既定を置く）
- **ずれたポートに追随する永続データ**: ホスト公開ポートは接続先であるだけでなく、DB に**保存される**値でも
  ある。JWT の issuer がそれで、mock 認証サーバーは `4000+N` で公開されるため発行トークンの `iss` はスロットで
  ずれ、resolver が `(issuer, subject)` で突き合わせる `user_identities` の行も一緒にずれていなければならない
  （リテラル固定だと、スロットを取った worktree では認証を要求する全エンドポイントが 401 になる）。そのため
  seed ファイルは URL ではなく `${AUTH_ISSUER}` を持ち、`make db-seed` がそのスロットの値を渡す
  （`database/seed/README.md` を参照）。`db-reinit` / `db-seed` / `slot-acquire` のいずれを通っても環境に一致する
  identity が入る。この種のデータを足すときも、既定ポートを焼き込まず同じようにスロットへ追随させること。
  DB 名と同じく、この値が host 実行の `go test` に届くのは `make` 経由だけ（`make test` / `test-cached` が
  export する）。素の `go test` は `DB_NAME_TEST` も スロットの issuer も受け取らないため、DB を使うテストは
  これらのターゲットから実行すること。
- **拡張・timezone のブートストラップ**: acquire は `wt<N>_local` / `wt<N>_test` を CREATE DATABASE
  （存在ガード）した後、各 DB に `pg_trgm` 拡張と `Asia/Tokyo` timezone を設定する（init スクリプトが
  `local` / `test` に施すのと同じもの。動的に作る worktree DB には明示設定が必要）。
- **スキーマ安全性**: acquire は取得後に `wt<N>_local` / `wt<N>_test` を drop→migrate→seed で
  自ブランチのスキーマへ作り直す。別ブランチが使ったスロットを引き継いでも安全。
- **スキーマ生成の隔離**: `make gen-query` の `dump-schema` は共有 `local` も自 worktree DB もダンプ
  せず、専用の `gen_schema` DB（`SCHEMA_GEN_DB`）を drop → 当該ブランチの migration で migrate-up して
  からダンプする。並行する別 worktree の migration が生成物（`schema.gen.sql` / `models.gen.go` 等）へ
  混入しない。ローカル専用のガードで、CI は fresh な postgres service を migrate 済みにして
  `dump-schema-ci` を直接呼ぶため本経路は通らない。
- **解放**: `slot-free` は、このスロットの app プロジェクト（`gobp-wt-N`）のコンテナを停止して
  から lease と `.gobp-db-slot` を削除する。データベースは warm 保持で次に貸す。
- **stale 回収の安全化**: heartbeat が TTL（既定 1800 秒、`GOBP_DB_POOL_TTL`）超過したリースは acquire 時に
  別 worktree が再取得できる（crash した worktree がスロットを握り続けない）。heartbeat は `make serve` 時に
  しか打たないため、起動しっぱなしの app を持つスロットも TTL 超過で stale になる。そこで DB を作り直す前に
  そのスロットが実際に使われていないかを 2 段で確かめ、使用中なら破壊せず skip する:
  1. app プロジェクト（`gobp-wt-N`）に稼働中コンテナが無いこと
  2. `pg_stat_activity` にそのスロットの DB への接続が無いこと

  接続プールはアイドルで空になるため、2 だけでは serve 中の worktree を見落とす。逆に 1 だけでは
  ホスト実行の `go test` を捉えられないため、両方を見る。

## 使い方

スロットを取らない checkout（主 checkout など）は、そのまま `make serve` すれば共有インフラへ繋がる。

```sh
make serve           # 共有インフラを起動し app を gobp-app-<dir> で起動 → curl localhost:8080
make serve-stop      # この checkout の app だけ停止（共有インフラは残す）
make infra-up        # 共有インフラだけ起動
make infra-down      # 共有インフラを停止（全 checkout に影響する）
```

worktree で並列に作業するときはスロットを取る。

```sh
make slot-acquire    # 空きスロットをリースし自 worktree DB を作成/再構築
make test            # ホストから localhost:5432 経由で wt<N>_test へ接続
make serve           # app を gobp-wt-N で起動 → curl localhost:$API_HOST_PORT（DB は共有の wt<N>_local）
make slot-status     # スロット占有状況（DB 名 / API ポート）を表示
make slot-free       # スロットだけを解放（データベースは warm 保持、worktree は残す）
```

作業が終わって worktree ごと畳むときは `slot-release` を使う。app の停止とローカルビルド
イメージの削除、スロット解放、worktree 削除をこの順で行う。

```sh
make slot-release    # app 停止+イメージ削除 → スロット解放 → worktree 削除
```

順序は入れ替えられない。`slot-free` が `.gobp-db-slot` を消すと `SERVE_PROJECT` が失われ、
`APP_PROJECT` が `gobp-app-<dir>` へフォールバックするため、先に解放すると app の停止が
別プロジェクトを対象にしてしまう。`git worktree remove` は cwd ごと消すので必ず最後に置く。
未コミット・未追跡ファイルがあれば git が削除を拒否するので、`--force` は付けていない。
主 checkout で誤って叩いた場合は、何もせずエラー終了する。

## 環境変数

| 変数 | 既定 | 意味 |
| --- | --- | --- |
| `GOBP_DB_POOL_DIR` | `~/.cache/gobp-db-pool` | リースレジストリの置き場所（symlink は拒否） |
| `GOBP_DB_SHARED_PROJECT` | `gobp-shared` | 共有インフラの固定 compose プロジェクト名 |
| `GOBP_API_POOL_BASE` | `8080` | `API_HOST_PORT` のベース（スロット N = ベース+N） |
| `GOBP_MOCK_AUTH_POOL_BASE` | `4000` | `MOCK_AUTH_HOST_PORT` のベース |
| `GOBP_DLV_POOL_BASE` | `2345` | `DLV_HOST_PORT` のベース |
| `GOBP_PPROF_POOL_BASE` | `6060` | `PPROF_HOST_PORT` のベース |
| `GOBP_DB_POOL_MAX` | `12` | スロット数（=同時並列数の上限） |
| `GOBP_DB_POOL_TTL` | `1800` | stale 判定の heartbeat 猶予（秒） |

## 注意

- **共有インスタンスの blast radius**: 全 checkout が 1 個の Postgres / o11y / オブジェクトストレージを共有する。
  データベースは分離されるため DDL は DB を跨げないが、reinit の対象 DB 名を取り違えると他 worktree を壊しうる。
  具体的には `db-init` / `db-local-reinit` / `db-test-reinit` は `DB=local` / `DB=test` を直書きするため、
  スロット保持中に叩くと自 worktree DB ではなく共有 DB を作り直す。自分の DB を作り直すときは
  `make db-reinit DB=$DB_NAME_LOCAL` のように対象を明示する。`make infra-down` も同様に全 checkout を止める。
- **スキーマ生成は同時実行できない**: `gen_schema` は共有インスタンス上の単一データベース名のため、複数の
  checkout が同時に `make gen-query`（`dump-schema`）を走らせると同じ DB を作り直して互いの出力を壊す。
  スキーマ生成は 1 checkout ずつ行う。
- **infra 層の再作成**: `gobp-shared` を動かす checkout が変わったとき、その checkout の compose 定義が
  前回と異なると（ブランチ違いで image の digest pin が変わった等）compose がコンテナを作り直す。
  名前付きボリュームは残るためデータは失われないが、稼働中 app の接続は切れる。
- **オブジェクトストレージは共有**: `garage` のバケットは全 checkout で共通（DB と違いスキーマを持たないため
  ブランチ間で壊れない）。ブランチ毎に隔離したい場合は `OBJECT_STORAGE_BUCKET` を分ける。
- API 帯 8080–8092 と被らないよう `sql_editor` / `docs_viewer` は 7000 番台へ退避済み。
- `docker/`・`internal/cli/dbslot`・`.makefiles/` を含む配線のため、変更時はこのドキュメントも更新すること。
