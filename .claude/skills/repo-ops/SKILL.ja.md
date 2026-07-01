> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Repo Ops Runbook

このリポジトリで繰り返し踏みがちな運用 gotcha（生成物・Docker ツールランナー・DB まわり）の具体的な復旧／手順集。根っこはほぼ1つ＝**`make gen-*` は Docker のツールランナーコンテナ（`node_tool_runner` / `go_tool_runner` / `python_tool_runner`）内で root 実行**され、DB 連動の生成は**ライブ DB**をダンプする、という点。これを押さえれば以下は全て説明がつく。

ワークフローではなくルックアップ表。症状を見つけて対応コマンドを打つ。破壊的（DB データ削除・ツリーの `chown`）な手順はユーザーに先に伝える（`CLAUDE.md` 準拠）。

## 1. `schema.gen.sql` ドリフト → `generate-db-check` / `gen-*-artifacts-check` が落ちる

`database/gen/schema.gen.sql` は `make dump-schema`（**起動中**の DB を `pg_dump`）が生成する。DB 初期化 SQL（`docker/database/*.sql` の拡張）や `database/migrations/**` を変えるとダンプ内容が変わり、コミット済み `schema.gen.sql` がドリフトして CI チェック（`dump-schema` → `git diff`）が落ちる。

対処: 再生成してコミット。

```bash
docker compose up -d database          # DB を起動
make dump-schema                       # ライブ DB から database/gen/schema.gen.sql を再生成
git add database/gen/schema.gen.sql
```

原則: **`docker/database/`・`database/migrations/`・DB 拡張に手を入れたら、同じ変更内で `schema.gen.sql` を再生成・コミットする。**

## 2. `git` が生成ディレクトリを拒否する — root 所有になっている

`make gen-portal-docs`（など docker-root 生成）は `.:/app` バインドマウント上で root 実行されるため `docs/portal/{guides,docs.json}` を **root 所有**で書く。以降ホストの `git add` / `git restore` が `permission denied`。`make gen-go-code` が新規作成する mock ディレクトリ（例 `pkg/fs/mock`・`pkg/exec/mock`）も同様。

対処: コンテナ経由で所有権を返す（必要時以外 `sudo` しない）:

```bash
# portal 出力
docker compose run --rm --user root node_tool_runner chown -R $(id -u):$(id -g) /app/docs/portal
# root 所有の mock ディレクトリ
docker compose run --rm --user root go_tool_runner chown -R $(id -u):$(id -g) /app/pkg/fs/mock /app/pkg/exec/mock
```

罠: root 所有掃除のための `git restore docs/portal` は、`docs/portal` 配下に**手で入れた編集も巻き戻す**。chown 後に別コミットで再適用／再同期すること。

## 3. `make gen-*` が DB 起動・リセットを要求する

`make gen-query` は `dump-schema`（§1）を連鎖するため **DB コンテナ起動が必須**（停止なら `connection refused`）。さらに現存テーブルを反映する＝テーブルのソースを消しても DROP していなければダンプに残り、生成モデルに死んだ型が残る。

スキーマに影響する変更後の再生成手順:

```bash
docker compose up -d database
make db-local-migrate-down db-test-migrate-down   # クリーンまで落とす（破壊的・要ユーザー確認）
make db-init-local db-init-test                   # 現行マイグレーション集合で再構築（＋seed）
make gen-query                                     # 意図したスキーマをダンプ
```

DB 層の統合テスト（`internal/infrastructure/rdb/repository`・`query_service`）は **test DB の migrate＋seed が前提**＝リセット後は local/test 両方を migrate-up＋seed。ホストからバイナリを docker DB に向けるには `DB_HOST=localhost go run ./cmd/ ...`。

## 4. per-env Docker イメージ

runtime イメージはビルド時 `--build-arg APP_ENV=<env>` で選んだ `env/.env.<env>` を1つだけ焼き込む（deploy ワークフローが注入）。実行時 ENV で切り替える単一イメージ方式ではない。

```bash
docker build --build-arg APP_ENV=stg --target runtime -t <img> -f docker/server/Dockerfile .
# .env.stg のみが焼き込まれたか検証
```

migration 専用イメージは無い。`env/` と `database/migrations` はバイナリに埋め込まれるため、マイグレーションは同じ `runtime` イメージの command override（`./server migrate-up`）で実行する。

## 5. commit-msg フック / commitlint エラー

lefthook の **commit-msg** フックは `make commitlint COMMIT_MSG_FILE={1}` を実行し、`commitlint` を `node_tool_runner` コンテナ内で走らせる（ツールはホストではなくコンテナ化ランナーで解決する。他の lint / 生成ターゲットと同じ再現性ルール。`docs/rules.md` 参照）。`node_tool_runner` イメージが未ビルド／古いと `commitlint: not found` で失敗するので、`make tool-runners-build`（または `docker compose build node_tool_runner`）で一度再ビルドする:

```text
make commitlint COMMIT_MSG_FILE={1}
```

`commitlint.config.js` は意図的に `type-case` を無効化（prefix は Cap-first `Feat`/`Fix`/… だが CI は全大文字＝単一 case を強制できない）し、`type-enum` をプロジェクト prefix に固定。`Merge`/`Revert` は commitlint 既定で自動無視。

## 6. 生成モック — 手書きせず Docker で再生成

モックは生成物・手書き禁止。各ソースに `//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE` を宣言し（`$GOFILE` ベースの destination はリポジトリ全体で統一されており、どの interface ファイルにも同じ行をコピペできる）、`make gen-go-code` が `go_tool_runner` 内で**ピン版** mockgen（`v0.6.0`）で実行。再生成された mock ディレクトリは root 所有で返ることがある（§2）。

インターフェース変更（シグネチャ・新メソッド）後は `*_mock.go` を編集せず再生成:

```bash
make gen-go-code        # go:generate（mockgen）を docker・ピン版で実行
```

## 7. `.makefiles` の SETUP_DRY_RUN 規約

setup/teardown 系 make ターゲットは `$(if $(DRY_RUN),--dry-run,)` と `[ -n "$(DRY_RUN)" ]` で dry-run を判定し、**空でない値は全て真**＝`DRY_RUN=0 make <target>` でも dry-run。実行は変数自体を付けない（`make <target>`）、プレビューは `DRY_RUN=1 make <target>`。

## 制約

- ✅ 知識提供（read-only）: 正確なコマンドを提示し、実行はユーザー依頼時のみ。
- ✅ 破壊的手順（§3 の DB 削除・§2 の `chown -R`）は事前に伝える（`CLAUDE.md` 準拠）。
- ✅ ホスト `sudo` より docker の `--user root … chown` 復旧を優先。
- ❌ 生成物（`*.gen.go`・`*.sql.go`・`*_mock.go`・`openapi.gen.yaml`・`schema.gen.sql`・`docs/portal/guides/**`）を手編集しない＝再生成する。
- ❌ スキーマ影響変更を `schema.gen.sql` 再生成なしにコミットしない（§1）。
