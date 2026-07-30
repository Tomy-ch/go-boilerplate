# scripts

[English](README.md) | 日本語

`scripts/` には、コード生成・ドキュメント・バージョニング・プロジェクト初期設定のための**ユーティリティスクリプト**が格納されています。

## ディレクトリ構成

```text
scripts/
├── gen-docs-json.mjs           # ポータルナビゲーション用 docs.json の生成
├── gen-portal-docs.mjs         # manifest.yaml に基づくドキュメントのポータルへのコピー
├── build-portal.mjs            # ポータルフロントエンド（src/main.jsx）を esbuild でバンドル
├── semver.mjs                  # セマンティックバージョニングヘルパー（patch/minor/major）
├── stamp-openapi-version.mjs   # release/vX.Y.Z のブランチ名から openapi.yaml の info.version を同期
├── sync-versions/              # mise.toml の go / node / python を go.mod と Dockerfile FROM へ反映（Go 実装）
├── make_help.mjs                # Make ターゲットのヘルプ出力生成
├── mermaid-lint.mjs            # Markdown 内の ```mermaid フェンスを mermaid パーサで構文検証
├── skill-lint.mjs              # .claude/** のスキル / エージェント定義を実態および .codex/** の対応と突き合わせて検証
├── pr-comment-secret-lint.mjs  # PR コメントを投稿するワークフロージョブへの secret 混入を検出
├── pr-comment-fence-lint.mjs   # PR コメント本文を囲む固定長 Markdown フェンスを検出
├── actions-cutoff-lint.mjs     # ジョブの timeout 設定と、打ち切りに耐える PR コメントを強制
├── genctxkey/                  # コンテキストキーコードジェネレータ（Go）
├── actions-shellcheck/         # composite action の `run:` スクリプトを shellcheck で検査（Go）
├── pin-actions/                # GitHub Actions の `uses:` 参照を commit SHA へ固定（Go）
├── pin-images/                 # Dockerfile の `FROM` base image を digest へ固定（Go）
└── setup/                     # プロジェクト初期設定スクリプト
    ├── replace-module.mjs
    ├── replace-app-metadata.mjs
    ├── replace-license-copyright.mjs
    ├── replace-repository-reference.mjs
    ├── replace-codeowners.mjs
    ├── remove-sample-api.mjs  # サンプルAPI(user/product/order)を削除 <!-- sample-api:line -->
    └── lib/                   # setup スクリプト共通ユーティリティ
```

## スクリプトカテゴリ

### ドキュメント生成

|スクリプト|説明|実行元|
|---|---|---|
|`gen-portal-docs.mjs`|`manifest.yaml` に基づきソースドキュメントをポータルの `guides/` にコピー|`make gen-docs`|
|`gen-docs-json.mjs`|ポータルアプリ用のナビゲーション `docs.json` を生成|`make gen-docs`|
|`build-portal.mjs`|ポータルフロントエンド（`docs/portal/src/main.jsx`）を esbuild で `docs/portal/dist/` 配下（`bundle.js` / `bundle.css` + 遅延チャンク）へバンドルし、`mermaid.min.js` も同じく dist/ へ配置。従来の CDN + ブラウザ内 Babel 構成を置き換え。|`make gen-portal-build`|

### Lint

|スクリプト|説明|実行元|
|---|---|---|
|`mermaid-lint.mjs`|リポジトリ内 Markdown の ` ```mermaid ` フェンスを全抽出し（除外範囲は `markdownlint-cli2` と同一）、実 `mermaid.parse` で構文検証する（DOM は `linkedom` で供給）。壊れた図が 1 つでもあれば非 0 で終了。`markdownlint` は Markdown の体裁しか見ず図の文法を見ない、その穴を塞ぐ。|`make md-lint` / `make md-mermaid-lint`|
|`skill-lint.mjs`|`.claude/**` のスキル / エージェント定義を意味的に検査する: frontmatter（`name` がディレクトリ / ファイル名と一致、`name` + `description` の存在）、対訳ペア（`SKILL.ja.md` の存在・frontmatter 不在・冒頭の翻訳注記・見出しレベル列が `SKILL.md` と一致）、参照の実在性（本文の `` `make <target>` `` が `Makefile` / `.makefiles/**` に実在、インラインコード中のリポジトリルート相対パスが実在）。あわせて各 skill / agent が `.codex/**` にも存在することを検査する。依存なしの ESM。スキル定義はエージェントの指示書でありながら、記述と実態の一致を誰も検査しておらず、片側の AI 環境にだけ入った skill にも誰も気づかない — その穴を塞ぐ。検査範囲と ignore ディレクティブは [Skill Lint](#skill-lint) を参照。|`make md-lint` / `make md-skill-lint`|
|`actions-shellcheck/`|`.github/actions/**` の `action.yaml` / `action.yml` を解析し、composite action の `runs.steps[].run` を抽出して各スクリプトを標準入力経由で `shellcheck` に掛け、指摘を `action.yaml` 上の行番号へ写し戻す。`actionlint` は `.github/workflows` しか走査せず、action マニフェストを直接渡してもワークフローとして解釈して失敗するため、composite action 内のシェルはどのゲートにも掛かっていなかった。その死角を埋める。方言はステップの `shell:` から決め、shebang として渡すことで `-s` を使わずに対象シェルを確定させる。`pwsh` / `python` / `cmd` や式で指定された `shell:` は検査せず skip として数える。`${{ }}` 式は行数を保つプレースホルダへ置換する（ワークフローの `run:` に対して `actionlint` が採る方式と同じ）。検査ステップ数が 0 なら非 0 で終了するため、抽出が壊れた状態が「緑」として通ることはない。|`make actions-lint` / `make actions-shellcheck`|
|`pr-comment-secret-lint.mjs`|`.github/workflows/` の各ワークフローをジョブ単位に切り出し、`./.github/actions/upsert-pr-comment` を使うジョブが `GITHUB_TOKEN` 以外の secret を参照していれば失敗する（ワークフロー全体の `env:` も対象）。依存なしの ESM。`actionlint` では表現できない規約を機械化したもので、規約の理由は [`.github/workflows/README.ja.md`](../.github/workflows/README.ja.md) を参照。検出範囲は `${{ }}` 式に現れる secrets の直接参照（`secrets.NAME` / `secrets['NAME']` / `toJSON(secrets)` のようなコンテキスト全体）。別ジョブで読んで `needs.<job>.outputs` 経由で渡す間接参照は静的には追えず、検査を通る。|`make actions-lint` / `make actions-comment-secret-lint`|
|`pr-comment-fence-lint.mjs`|ワークフローの `run:` ブロックが PR コメント本文を固定長の Markdown フェンスで囲んでいる場合と、複製されている `fence_for` の実装が互いに一致しなくなった場合に失敗する。依存なしの ESM。`actionlint` では表現できない規約を機械化したもので、フェンスを囲む本文から算出すべき理由は [`.github/workflows/README.ja.md`](../.github/workflows/README.ja.md) を参照。検出範囲は `echo` 中のリテラルなフェンスと、実装同士の文字列一致。ある本文が攻撃者制御かどうかはここでは判定できず、規約に委ねる。|`make actions-lint` / `make actions-comment-fence-lint`|
|`actions-cutoff-lint.mjs`|ジョブに `timeout-minutes` が無い場合と、`./.github/actions/upsert-pr-comment` を呼ぶステップの `if:` がキャンセルされたジョブから到達できない場合に失敗する。依存なしの ESM。`actionlint` では表現できない規約を機械化したもので、打ち切りが何を残すべきかは [`.github/workflows/README.ja.md`](../.github/workflows/README.ja.md) を参照。検出範囲は条件中の `always()` / `cancelled()`。`failure()` は意図的に数えない（キャンセル時は false）。reusable workflow を呼ぶジョブは同キーが invalid なため除外する。到達性を自ら打ち消す条件（`!always()`）は書けてしまい静的には捕まらないので、そこは規約が支える。|`make actions-lint` / `make actions-cutoff-lint`|

#### Skill Lint

`skill-lint.mjs` は、Makefile のターゲット一覧・ファイルシステム・見出し抽出から機械的に導出できることだけを主張する（文面の良し悪しは判断しない）。参照検査が読むのは**コードフェンス外のインラインコードスパン**に限る（フェンス内は例示・出力サンプルであり実在性を保証しない）。

パス参照は、パスであることが一意に決まるときだけ検査する: 先頭セグメントがリポジトリルート直下の実在エントリであり、かつ末尾が `/` か basename にドットを含むもの。これにより Go の import パス（`database/sql`）、パッケージ修飾シンボル（`pkg/ptr.Copy`）、省略記法（`internal/controller/handler/...`）、文脈相対のファイル名（`SKILL.md`）は意図的に対象外となる — いずれも解決先が一意に決まらない。`<placeholder>` / `*` / `**` / `{a,b}` はパターンとして解決し、パスは参照元ファイルからの相対でも解決を試みる（スキルが同梱する `scripts/` を指せるようにするため）。

仮定の例示・任意配置・対向 AI ツール側リポジトリのファイルなど、意図的に不在な参照には、その行のどこかに ignore ディレクティブを置く:

```markdown
- `internal/controller/handler/debug/README.md` → `docs/portal/guides/controller-handler-debug.md` (if it were added) <!-- skill-lint-ignore -->
```

2 つの AI 環境をまたぐ検査は**存在のみ**に限る: `.claude/skills/<name>/` には `.codex/skills/<name>/` があり（逆も同様）、`.claude/agents/<name>.md` には `.codex/agents/<name>.toml` があり（逆も同様）、Codex の各スキルは README が定める `SKILL.md` + `agents/openai.yaml` を持つこと。Codex 側の `SKILL.ja.md` は任意なので、存在するときだけ対訳ペアとして検査する。

本文の追随は意図的に検査しない。`sync-ai` は逐語コピーではなく意味ポートであり、`CLAUDE.md` ↔ `AGENTS.md` の言い換え・Claude 固有機構の適応・凝縮スタイルへの書き下ろしといった意図的な差分が恒久的に残る（共通スキルのうち少なからぬ数は、見出し集合がまったく重ならない）。存在の対応であれば例外は宣言可能な件数に収まり、それでいて肝心の事故 — 片側の環境にだけマージされた skill — は捕まえられる。

意図的に片側の環境だけへ置く skill は、`skill-lint.mjs` の `PLATFORM_ONLY_SKILLS` へ**理由付きで**登録する。理由が空の登録は落ち、両環境に揃った（あるいはどちらにも無くなった）skill の登録も落ちるので、例外リストが例外より長生きすることはない。**この仕組みの正典はここで、他のドキュメントは再掲せずここへリンクする。**

エージェント役割にはこの逃げ道が無く、対応は無条件に要求される。意図的に片側だけへ置いたエージェントはこれまで無く、例外の表を用意してもエントリが 0 件になる — 例外機構は実際の事例が出てから足す。

### バージョニング

|スクリプト|説明|実行元|
|---|---|---|
|`semver.mjs`|セマンティックバージョンのバンプ（patch / minor / major）|リリースワークフロー|
|`stamp-openapi-version.mjs`|`release/vX.Y.Z` のブランチ名から `X.Y.Z` を導出し `openapi.yaml` の `info.version` に書き込む（先頭の `version:` 行のみ・冪等・非 release ref は no-op）。契約版のみで SHA / build metadata は付けない（commit 単位の追跡は runtime の `/version` の責務）。依存ゼロの ESM で、素の runner `node` で動く。|`auto-generate-docs.yaml`|
|`sync-versions/`|Go 実装の sync ツール。`mise.toml` の `[tools]` table を行ベース parser で解析し（外部依存ゼロ）、`go` / `node` / `python` を `go.mod` の `go` directive と `docker/*/Dockerfile` の `FROM golang:` / `FROM node:` / `FROM python:` 行へ反映する。version 存在・ファイル存在・期待マッチ数の事前 validate を全 rule で通してからファイル単位 atomic に書き出すため、partial state にならない。|`make sync-versions`|

その他のツールのバージョンは [`mise.toml`](../mise.toml) を SSOT として管理しています。各環境（host / docker / CI）は必要なものだけ `mise install <tool>` で個別に取得するため、sync スクリプトは不要です。

### Makefile サポート

|スクリプト|説明|実行元|
|---|---|---|
|`make_help.mjs`|`.makefiles/*.mk` を解析してターゲット説明を表示|`make help`|

### コード生成

|スクリプト|説明|実行元|
|---|---|---|
|`genctxkey/`|Echo コンテキストキーヘルパーの生成（Go コードジェネレータ）。`internal/controller/ctxhelper/generate.go` の `//go:generate` ディレクティブから `go generate ./...` 経由で実行される。|`make gen-go-code`|

詳細は [genctxkey/README.ja.md](genctxkey/README.ja.md) を参照。

### CI / サプライチェーン

|スクリプト|説明|実行元|
|---|---|---|
|`pin-actions/`|`.github/workflows/**` と `.github/actions/**` の外部 GitHub Actions `uses:` を不変の commit SHA へ固定する。`resolve` は参照を走査し各 tag/branch を `git ls-remote` で SHA へ解決して lockfile `.github/actions-pin.toml`（SSOT）へ書き出す。`PIN_ACTIONS_MIN_AGE_DAYS`（既定 14）日未満の新しすぎるコミットは採用せず既存ピンを維持する supply-chain quarantine 付き。`apply` は lockfile を元に各 `uses:` を `@<sha> # <tag>` へ書き換える。`check` は書き換えずに同じ判定を行い、未固定/古い/未登録があれば非 0 で終了する（CI / hook 用）。既に固定済みの行はコメント末尾の `# <tag>` を版として再解決するため冪等。|`make pin-actions-resolve` / `pin-actions-apply` / `pin-actions-check`|
|`pin-images/`|`docker/*/Dockerfile` の全 `FROM` base image を不変の digest へ固定する。`resolve` は各 `image:tag` を集め `docker buildx imagetools inspect` で現在 digest へ解決して lockfile `docker/images-pin.toml`（SSOT）へ書き出す。image-config の `created` が `PIN_IMAGES_MIN_AGE_DAYS`（既定 14）日未満の digest は採用しない supply-chain cooldown 付き。mutable tag は履歴を問えないため step-back 先はツール自身の前回 lock で、初回（無い場合）は tag のまま残す。`apply` は lockfile を元に各 `FROM` を `image:tag@sha256:...` へ正規化し、quarantine 中の image は digest を剥がして tag のみへ戻す。`check` は書き換えずに同じ判定を行い、drift があれば非 0 で終了する（CI / hook 用）。tag は版の SSOT として `FROM` 行に残す。|`make pin-images-resolve` / `pin-images-apply` / `pin-images-check`|

### 初期設定（`setup/`）

ボイラープレートから新規プロジェクトを作成する際の設定スクリプトです。

|スクリプト|説明|
|---|---|
|`replace-module.mjs`|Go モジュール名を全 `.go`、`go.mod` 等で置換|
|`replace-app-metadata.mjs`|env ファイルと OpenAPI 仕様のアプリ名・説明を置換|
|`replace-license-copyright.mjs`|LICENSE の著作権者名と年を置換|
|`replace-repository-reference.mjs`|README と OpenAPI の GitHub リポジトリ参照を置換|
|`replace-codeowners.mjs`|`.github/CODEOWNERS` の全ルールの所有者を置換。コメント行は記載例を保つため対象外で、所有者欄を判定できないルール行は書き換えずに報告する。|
|`remove-sample-api.mjs`|サンプルAPI(`user`/`product`/`order`)を削除。`lib/sample-api.mjs` に宣言したパスを削除し、共有 DI モジュールと `openapi.yaml` の `sample-api` マーカーブロックを除去する。再生成・整形・Lint まで行うには `make setup-remove-sample-api` 経由で実行する。 <!-- sample-api:line -->|

すべての setup スクリプトはプレビュー用の `--dry-run` をサポートしています。
<!-- sample-api:begin -->

`remove-sample-api.mjs` の削除対象とマーカーは [`lib/sample-api.mjs`](setup/lib/sample-api.mjs) に宣言されています。サンプルは3ドメイン構成（`user` はフルスタック、`product`/`order` は拡張予定の DB スタブ）で、拡張時は該当ドメインブロックにパスを追記し、混在行を `sample-api:begin … sample-api:end`（または `sample-api:line`）で囲むだけで対象に含まれます。
<!-- sample-api:end -->

## 注意点

- ドキュメント生成スクリプトは Node.js と `js-yaml` が必要（`docker/tools/` 経由でインストール）
- setup スクリプトは一度だけ使用 — ボイラープレートから新規プロジェクト作成時に実行
- AI エージェントは明示的な指示がない限りこのディレクトリを変更しないこと
