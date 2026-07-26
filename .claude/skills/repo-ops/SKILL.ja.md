> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。直接編集せず、更新は `SKILL.md` 側から流してください。

# Repo Ops Runbook

このリポジトリで繰り返し踏む運用 gotcha の具体的な復旧手順集。ワークフローではなくルックアップ表なので、症状を見つけて対応コマンドを打つ。破壊的な手順（DB データ削除・ツリーの `chown`・他 checkout が依存するコンテナの停止）は先にユーザーへ伝える（`CLAUDE.md` 準拠）。

以下はほぼすべて、次の3つの事実から説明できる:

1. **コード生成は Docker のツールランナーコンテナ内で root 実行される**（`go_tool_runner` / `node_tool_runner` / `python_tool_runner`）。`.:/app` バインドマウント越しなので生成物は root 所有で返り、ツールはホストではなくイメージ側に居る。
2. **インフラは単一の共有 compose プロジェクト**。`database` / `observability` / `garage` は固定プロジェクト `gobp-shared` に**全 checkout で 1 インスタンス**だけ起動し、checkout 毎に分かれるのは `api_server` / `mock_auth_server` のみ。worktree の分離軸はポートではなく**データベース名**（`wt<N>_local` / `wt<N>_test`）。設計の正本は `docs/maintenance/db-worktree-pool.md`、全体像は `docs/maintenance/local-environment.md`。
3. **`make lint` / `fix` / `test` はホスト実行**（mise 経由）。「全部 docker 化」の例外であり、ローカルと CI が食い違う原因になる。

## 症状インデックス

| 症状 | 節 |
| --- | --- |
| `docker compose ps` が空 / "port 5432 already allocated" / サービスが見つからない | §1 |
| `schema.gen.sql` や sqlc 出力がドリフトし `gen-db-artifacts-check` が落ちる | §2 |
| `make gen-query` が DB に届かない / 複数 checkout が互いを壊す | §3 |
| `git add` / `restore` が生成ディレクトリで permission denied | §4 |
| `make slot-acquire` 後にテスト・マイグレーションが別 DB を見る | §5 |
| ブランチ切り替え直後に統合テストが落ちる | §5 |
| pre-push の `secret-scan` が自分の追加していない秘密を検出する | §6 |
| `sample-removal-check` が CI で落ちる | §7 |
| 触っていないのに `env/.env` が dirty | §8 |
| ローカル golangci-lint が CI と食い違う / `golangci-lint: not found` | §9 |
| `commitlint: not found` / `orval: not found` / ツールが古い | §10 |
| 自分の変更と無関係な理由でフックが落ちる | §11 |
| `pin-images-check` / `pin-actions-check` が未固定・未登録で落ちる | §12 |
| pre-commit の "Migration version gap / duplicate" | §13 |
| ローカルの S3 呼び出しが 503 を返す | §14 |
| golden JWKS のドリフト / mock-auth の OpenAPI が古い | §15 |
| 特定環境向けイメージのビルド | §16 |
| CI で `sync-versions` のドリフト | §17 |
| どのドキュメントが答えを決めるのか分からない / `grep` が対訳・生成物に埋もれる | §0 |

## 0. 正本の見つけ方

このツリーの Markdown の大半は、読んではいけない日本語ミラーか、コードに遅れて追随する生成物のどちらかである。そのためリポジトリ全体を素朴に検索すると、実際に答えを決めている唯一のファイルが埋もれる。追跡されている `*.md` 927 件のうち **409 件が `*.ja.md` 対訳**、**72 件が README を写した生成物 `docs/portal/guides/**`**。さらに `docs/godoc/**` が約 1,250 件、`docs/db-schema/**` が約 390 件ある。

### 正本の所在

| 知りたいこと | 読む場所 | 正本に見えるが違うもの |
| --- | --- | --- |
| make ターゲットの実体 / どのターゲットが X をやるか | `.makefiles/**/*.mk`、索引は `.makefiles/README.md`、`make help` | `docs/portal/guides/make.md` — 生成物で `.mk` に遅れる |
| レイヤ境界・生成物 / DTO / tx / コメントの規則 | `docs/rules.md` | — |
| システム構造とレイヤの責務 | `docs/architecture.md` | — |
| 変更の進め方（API / DB / ビジネスロジック） | `docs/development-flow.md` | — |
| テスト規約 | `docs/testing-conventions.md` + 当該レイヤ README の *Test Strategy* | — |
| なぜその設計なのか | `docs/adr/`（107 レコード。ログは `docs/adr/README.md`） | — |
| サブシステムの仕組み（rest / worker / job / outbox / idempotency / o11y / auth） | `docs/design/README.md` と同ディレクトリ | — |
| ローカルの構成・compose の層分け・ホットリロード | `docs/maintenance/local-environment.md` | — |
| worktree スロットリングと共有 DB | `docs/maintenance/db-worktree-pool.md` | — |
| パッケージ単位の詳細と設計意図 | 最も近い `internal/**/README.md` / `pkg/**/README.md`（171 個） | スキル本体 — スキルが README に従うのであって逆ではない |
| 環境変数 | `internal/config/envspec.go` + `model.go`、`env/README.md` の表 | `env/.env*` の値だけ |
| CI ゲートやフックが実際に検査する内容 | `.github/workflows/*.yaml`、`.lefthook.yaml` | — |
| ツール / ランタイムのバージョン | `mise.toml`（他はすべて派生 — §17） | `go.mod`・Dockerfile・README — いずれも派生物 |
| 生成物・保護パス・スコープ | `AGENTS.md` | — |

### ノイズを除いた検索

```bash
rg "<pattern>" \
  -g '!**/*.ja.md' -g '!docs/ja/**' -g '!docs/portal/**' \
  -g '!docs/godoc/**' -g '!docs/db-schema/**' -g '!docs/openapi/**' -g '!docs/coverage/**'
```

実測の効果: `gen-query` は 45 → 19 ヒット、`NormalizeError` は 74 → 33。`rg` は `.gitignore` を尊重するため `vendor/` と `node_modules/` は既に除外されるが、生成された `docs/` 配下は*追跡されている*ので明示的な glob が要る。`*.ja.md` にヒットすること自体は**所在の手がかり**として有用（その話題が文書化されている証拠）だが、`AGENTS.md` の「`*.ja.md` を読まない」規則どおり、隣の英語正本を読むこと。

### 情報源が食い違うとき

`AGENTS.md` の指示優先順位に従う: `AGENTS.md` → `docs/rules.md` → `docs/architecture.md` → ユーザー指示。設計意図と実装方針については **README > Code > SKILL**（`back-prop` が強制する規則）。README と矛盾するスキルのほうが古い。コードと README が食い違うならそれはドリフトであり、表面化させる価値がある（`back-prop` が検出する）。

## 1. compose のプロジェクト解決 — 素の `docker compose` は別プロジェクトを指す

compose はプロジェクト名をディレクトリから導出するため、このリポジトリ（や worktree）で素の `docker compose <cmd>` を打つと、共有インフラではなく**空のプロジェクト**を相手にする。`.makefiles/docker/compose.mk` が `COMPOSE_PROJECT_NAME=gobp-shared` を export しているので、これが効くのは `make` 経由のときだけ。

```bash
docker compose ps                  # ← 空。プロジェクト = <ディレクトリ名>
docker compose -p gobp-shared ps   # ← 実際の共有インフラ
```

取り違えたときの結果: `docker compose up -d database` は**2台目**の Postgres を起動してホストポート 5432 を奪い合う。`docker compose exec database psql …` はサービスが起動していないと言う。`docker compose run --rm go_tool_runner make db-migrate-ci-up` は一時コンテナが別ネットワークに繋がるため `database` ホストを解決できない。

対処: インフラは `make` 経由で操作する。素の compose が避けられないならプロジェクトを明示する。

```bash
make infra-up                                    # database / observability / garage（+ garage_init）を起動
make serve                                       # infra-up + この checkout の app 層
make serve-stop                                  # この checkout の app だけ停止
COMPOSE_PROJECT_NAME=gobp-shared docker compose exec -T database psql -U postgres -l
```

`make infra-down` は**全** checkout / worktree のインフラを止める。実行前に確認すること。

## 2. DB 生成物のドリフト → `gen-db-artifacts-check` が落ちる

`database/gen/schema.gen.sql` と sqlc 出力（`internal/infrastructure/rdb/sqlc/gen/*.gen.sql.go`・`database/gen/*.gen.sql`）は `make gen-query` が生成する。`database/migrations/**`・`database/dml/**`・`docker/database/**` に触れば内容が変わるため、ソースだけコミットすると CI チェック（再生成 → `git diff`）が落ちる。

```bash
make infra-up
make gen-query                     # dump-schema → merge-dml → sqlc generate → fmt
git add database/gen internal/infrastructure/rdb/sqlc/gen
```

原則: **スキーマ / DML に影響する変更と、その再生成物は同じ変更に含める。** PR が SQL ソースに触っていないのに CI がドリフトを報告する場合は generator 側のドリフト（sqlc バージョン差・base の生成物が古い）＝ローカルで再生成するか、base を最新化してマージする。

## 3. `make gen-query` — インフラ起動が前提、かつ `gen_schema` は共有

`dump-schema` はもう自分の `local` データベースをダンプしない。専用の **`gen_schema`** データベース（`.makefiles/database/gen.mk` の `SCHEMA_GEN_DB`）を用意してテーブルを削除し、*このブランチの* migration で migrate-up してからダンプする。よって作業用 DB の残骸テーブルが生成コードへ漏れることは無くなったが、次の2点は残る:

- 実体は `docker compose exec database` 経由で共有 Postgres を叩くので、インフラ起動は依然必須（§1）。落ちていれば接続エラーで死ぬ。
- **`gen_schema` は唯一の共有インスタンス上の単一データベース名**。複数 checkout が同時に `make gen-query` を走らせると同じデータベースを作り直し、互いの出力を壊す。スキーマ生成は 1 checkout ずつ行うこと。（この制約は現在 `gen.mk` のコメントにしか無く、`docs/` には載っていない。）

## 4. `git` が生成ディレクトリを拒否する — root 所有になっている

ツールランナーはバインドマウント上で root 実行されるため、*新規作成*したファイルはホスト側で root 所有になる: `make gen-portal-docs` の `docs/portal/{guides,docs.json}`、`make gen-go-code` が作る mock ディレクトリ（例 `pkg/fs/mock`）、`docs/godoc/`、その他新規に増えた生成物。以降ホストの `git add` / `git restore` が `permission denied` で失敗する。

対処: コンテナ経由で所有権を返す（ホストの `sudo` より優先）:

```bash
docker compose run --rm --user root node_tool_runner chown -R $(id -u):$(id -g) /app/docs/portal
docker compose run --rm --user root go_tool_runner   chown -R $(id -u):$(id -g) /app/pkg/fs/mock /app/pkg/exec/mock
```

罠: root 所有を掃除するための `git restore docs/portal` は、`docs/portal` 配下に**手で入れた編集も巻き戻す**。chown 後に別コミットで再適用すること。

## 5. スロット取得後・ブランチ切り替え後に別 DB を見てしまう

`make slot-acquire` は `.gobp-db-slot` を書き出し、make がそこから `DB_NAME_LOCAL` / `DB_NAME_TEST`（`wt<N>_local` / `wt<N>_test`）を伝播する。ここに2つの罠がある:

- **`db-init` / `db-local-reinit` / `db-test-reinit` は `DB=local` / `DB=test` を直書きしている。** スロット取得後にこれらを叩くと自分のではなく*共有*データベースを作り直す＝それを使っている相手に対して破壊的。自分の DB は明示して指定する:

  ```bash
  set -a; . ./.gobp-db-slot; set +a
  make db-reinit DB="$DB_NAME_LOCAL"      # drop tables → migrate-up → seed
  make db-reinit DB="$DB_NAME_TEST"
  ```

- **素の `go test ./...` には `DB_NAME_TEST` が渡らない** — export しているのは make であってシェルではない — ため、黙って共有 `test` データベースへ繋ぐ。そこは別ブランチの migration が当たっているかもしれない。`make test` を使う（あるいは自分で export する）。

統合テスト（`internal/integration`・`internal/infrastructure/rdb/**`）は自分でマイグレーションしない。migrate + seed 済みのデータベースを前提にする。migration の異なるブランチへ切り替えたら先に作り直すこと。スロットの DB は `make slot-acquire` が、共有 `test` は `make db-test-reinit` が担当する。`db-init` より `db-reinit` を優先すること: `db-init` は先に `migrate-down` を走らせるため、`.down.sql` が失われたダーティスキーマからは復帰できない。

## 6. pre-push の `secret-scan` が自分の追加していない秘密を検出する

`.gitleaksignore` のエントリは `<path>:<rule>:<line>` 形式のフィンガープリントで、**行番号を含む**（例: `docker-compose.yaml:generic-api-key:134`・`env/.env:generic-api-key:64`）。該当ファイルを編集して行がずれると旧フィンガープリントが一致しなくなり、意図的に許容していたサンプル資格情報が新規検出として報告される。

```bash
make secret-scan          # 再現する。出力に新しいフィンガープリントが載る（値は redact 済み）
```

そのうえで `.gitleaksignore` の該当行を新しい行番号へ更新する（説明コメントは残す）。これを行ってよいのは、そこに意図的なものとして既に記載されているエントリ（mock-auth の署名鍵・Garage の開発用資格情報）だけ。本当に新規の検出は実際の秘密なので、無視せずツリーから除去する。

## 7. `sample-removal-check` が CI で落ちる

`scripts/setup/lib/sample-manifest.mjs` は、テンプレート利用者がサンプル API を剥がすときに `make setup-remove-sample-api` が削除する全パスを宣言している。サンプルドメイン（user / product / purchase / …）配下でファイルを追加・移動・改名したのに登録しないと、削除後に参照が宙に浮く。CI はこれを、実際に削除を実行してから build / lint / test することで検出する。ローカルでは何も落ちない＝自分で走らせない限り CI 専用で見つかる。

サンプルドメインにファイルを足したら（handler / usecase / domain / repository / DML / migration / seed / spec / 統合テスト / サンプル専用の生成物）、該当ドメインのエントリにパスを追記する。共有ファイルに混ざった行はパスではなく `sample-api` マーカーコメントで囲って扱う。削除せずに影響だけ見るには:

```bash
DRY_RUN=1 make setup-remove-sample-api
```

## 8. 触っていないのに `env/.env` が dirty

`env/.env` は git 管理下にあり、`make materialize-env` がビルド時埋め込みのために `env/.env.$(APP_ENV)`（既定 `ci`）で上書きする。CI とイメージビルドがこれを行うため、ローカルで途中中断するとコピーが残る。

```bash
make restore-env          # git restore env/.env
```

材料化された `env/.env` はコミットしないこと。なおこのファイルを編集すると §6 の gitleaks フィンガープリントもずれる。

## 9. ホスト実行の lint / format / test と、2つの golangci 設定

`make lint` / `make fix` は golangci-lint を**ホスト**の mise 経由で解決し（`mise which golangci-lint`）、`make test` もホストで `go test` を走らせる。バイナリが無ければ `mise install`。ここでコンテナに手を伸ばさない。

設定ファイルは2本あり、素の `golangci-lint run` は誤ったほうを拾う:

```bash
golangci-lint run                                  # → .golangci.yaml（軽量。一部 linter が無効）
make lint                                          # → .golangci-full.yaml。CI と同じ
golangci-lint run --config .golangci-full.yaml     # 追加フラグを付けたいときはこちら
```

CI の失敗を再現するときは必ず full 設定を使う。

## 10. `commitlint: not found` / `orval: not found` / ツールが古い

ツールランナーのイメージは **`mise.toml` と `docker/tools/package*.json` のビルド成果物**。ツールはホストではなくランナー内で解決される（コード生成と同じ再現性ルール。`docs/rules.md` 参照）。どちらかを変更した後、あるいはイメージがそれらより古いクローンでは、ランナーにツールが無い / 版が古い状態になる。

```bash
make tool-runners-build           # go / node / python ランナーを再ビルド（キャッシュ利用）
make tool-runners-build-clean     # --no-cache --pull。キャッシュ層が原因のとき
```

node ランナーは `/app/scripts/node_modules` を匿名ボリュームとして持つ（バインドマウントによる shadow を防ぐため）。補助スクリプトと `gen-mock-auth-oapi` はここからバイナリを解決するので、イメージが古いとこれらが壊れる。

commit-msg フックは `node_tool_runner` 経由で `make commitlint COMMIT_MSG_FILE={1}` を実行するため、コミット失敗の典型原因はこれ。`commitlint.config.js` は意図的に `type-case` を無効化し（prefix は Cap-first の `Feat`/`Fix`/… だが CI のメッセージは全大文字＝単一 case を強制できない）、`type-enum` をプロジェクトの prefix に固定している。`Merge` / `Revert` は既定で無視される。

## 11. フック対応表 — 何がいつ走るか、自分の変更と無関係に落ちたときどうするか

`.lefthook.yaml` をトリガー別に整理すると:

| フック | glob → コマンド（抜粋） |
| --- | --- |
| pre-commit | `*.go` → `make lint`・`make test-cached`／`*.sql` → `make sql-lint`／`*.md` → `make md-lint`／`.github/workflows/**` → `make actions-lint`・`make pin-actions-check`／`openapi/**` → `make lint-oapi`／`docker/mock-auth-server/openapi/**` → `make lint-mock-auth-oapi`／`docker/**/Dockerfile`・`docker-compose*.yaml` → `make docker-lint`・`make pin-images-check`／`database/migrations/*.sql` → migration の重複・ギャップ検査 |
| commit-msg | `make commitlint COMMIT_MSG_FILE={1}` |
| pre-push | `make secret-scan`／`*.go` → `make test`／`*.go`・`openapi/**` → 再生成して `*.gen.go`・mock・`openapi.gen.yaml` を `git diff --exit-code`／`go.mod`・`go.sum` → `go mod tidy` + diff |

pre-push の `gen-go-check` は Docker で再生成して差分があれば落ちる。対処は再実行ではなく、再生成物をコミットすること（§2・§4）。自分の変更と無関係な理由で赤いとき（base ブランチに元からある失敗・環境要因）は `--no-verify` で push し、原因は別途つぶす。変更のほうを歪めない。

## 12. `pin-images-check` / `pin-actions-check` — fail-closed な lockfile

どちらも fail-closed。`docker/*/Dockerfile` の全 `FROM`、`docker-compose*.yaml` の全 `image:`、`.github/workflows/**` の全 `uses:` は、固定済みで**かつ** lockfile（`docker/images-pin.toml`・`.github/actions-pin.toml`）に登録済みでなければならない。compose サービスや action を新規追加すると、未登録（lockfile に無い）／未固定（tag のまま）でコミットがブロックされる。

固定済みの digest / SHA を手で書き換えないこと。専用スキル（images は `images-pin`、actions は `actions-pin`）を使う。サプライチェーンの cooldown（公開直後の digest は採用せず拒否する）と `resolve` → `apply` → `check` の手順はそちらが持っている。

## 13. pre-commit の "Migration version gap / duplicate"

`make check-migration-{up,down}-{version,gap}` は `database/migrations/**` の連番が up / down 双方で一意かつ連続であることを要求する。典型的な発火要因は、自分の番号を既に使っている base ブランチをマージしたとき。*自分の*新規ファイル側を採番し直す（`.up.sql` と `.down.sql` 両方）。コミット済みの既存 migration は `AGENTS.md` のとおり編集しない。新規作成は `make new-migrate-<name>` で行い、採番をツールに任せる。

## 14. ローカルの S3 呼び出しが 503 を返す

`garage` のバケット・レイアウト・アクセスキーは one-shot の **`garage_init`** コンテナがプロビジョニングする。プロジェクトを跨ぐため compose の `depends_on` では表現できず、`make infra-up` が起動して終了まで待つ。プロビジョニング完了前に上げた app は S3 エンドポイントから 503 を受ける。

```bash
make infra-up                     # 冪等。garage_init の終了まで待つ
```

バケットは全 checkout で共有される（データベースと違いスキーマを持たないため、ブランチ間で壊れない）。ブランチ毎に隔離したい場合は `OBJECT_STORAGE_BUCKET` を分ける。Go のテストは garage を使わず in-process の gofakes3 を使う。

## 15. mock-auth-server — 独自の生成物を持つ別 npm プロジェクト

`docker/mock-auth-server/` は独立した Node プロジェクト（自前の `package.json` / OpenAPI / テスト）。その出力のうち2つが CI でドリフト検知される:

- **golden JWKS** — `fixtures/jwks/*.json` と `internal/integration/testdata/jwks/*.json` は同一の generator が書き出し、provider と Go 統合テストが単一ソースを共有する。鍵ストアや鍵に触ったら `docker/mock-auth-server` で `npm run gen:jwks` を実行し、両ディレクトリをコミットする。
- **OpenAPI バンドル + zod スキーマ** — `make gen-mock-auth-oapi` で再生成する（`node_tool_runner` 内で実行され、`orval` は `/app/scripts/node_modules` から解決される。見つからない場合は §10）。

`make reset-mock-auth-users` は、テスト実行で書き換わった固定 mock ユーザーを中立な既定へ戻す。

## 16. 環境別 Docker イメージ

runtime イメージは、ビルド時に `--build-arg APP_ENV=<env>` で選んだ `env/.env.<env>` を1つだけ焼き込む（deploy ワークフローが注入する）。実行時 ENV で切り替える単一イメージ方式ではない。

```bash
docker build --build-arg APP_ENV=stg --target runtime -t <img> -f docker/server/Dockerfile .
# .env.stg だけが焼き込まれたか検証する
```

マイグレーション専用イメージは無い。`env/` と `database/migrations` はバイナリに埋め込まれるため、マイグレーションは同じ `runtime` イメージの command override（`./server migrate-up`）で実行する。

## 17. `sync-versions` のドリフト

ツールバージョンの正本は `mise.toml` で、`go.mod`・各 Dockerfile・`docker/**` の README は派生物。派生側を直接編集したり、`mise.toml` を上げて伝播させないと CI チェックが落ちる。

```bash
make sync-versions                # mise.toml から伝播させ、結果をコミットする
```

Go バージョンの更新は `go-upgrade` スキルを使う（手順一式はそちらが持つ）。

## 18. 生成モック — 手書きせず Docker で再生成

各ソースが `//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE` を宣言し（`$GOFILE` ベースの destination はリポジトリ全体で統一されているため、どの interface ファイルにも同じ行をコピペできる）、`make gen-go-code` が `go_tool_runner` 内でピン留めされた mockgen（`mise.toml`。現在 `0.6.0`）で実行する。インターフェース変更後は `*_mock.go` を編集せず再生成する。新規 mock ディレクトリは root 所有で返ることがある（§4）。

```bash
make gen-go-code
```

## 19. `.makefiles` の DRY_RUN 規約

setup / teardown 系ターゲットは `$(if $(DRY_RUN),--dry-run,)` と `[ -n "$(DRY_RUN)" ]` で dry-run を判定するため、**空でない値はすべて真**＝`DRY_RUN=0 make <target>` でも dry-run になる。実際に実行するときは変数自体を付けない。プレビューは `DRY_RUN=1 make <target>`。`setup-repo` はプレビュー不可能なため `DRY_RUN` 自体を拒否する。

## 制約

- ✅ 知識提供（read-only）: 正確なコマンドを提示し、実行はユーザーに依頼されたときだけ行う。
- ✅ 破壊的な操作の前に伝える: DB の作り直し（§5）・`chown -R`（§4）・`make infra-down` など全 checkout に影響する共有インフラ操作（§1）。
- ✅ 素の `docker compose` より `make` を優先する。素の compose が避けられないときは `COMPOSE_PROJECT_NAME=gobp-shared` / `-p gobp-shared` を明示する（§1）。
- ✅ ホストの `sudo` より docker の `--user root … chown` による復旧を優先する。
- ❌ 生成物（`*.gen.go`・`*.sql.go`・`*_mock.go`・`openapi.gen.yaml`・`schema.gen.sql`・`docs/portal/guides/**`・固定済み digest / SHA）を手編集しない＝再生成するか、所管スキルを使う。
- ❌ スキーマ / DML に影響する変更を再生成物なしにコミットしない（§2）。材料化された `env/.env` をコミットしない（§8）。
