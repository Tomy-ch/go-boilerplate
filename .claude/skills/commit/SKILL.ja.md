> このファイルは `SKILL.md`（canonical / 英語）の日本語参考訳です。スキルとしては読み込まれません（参考用）。

# Commit

`/commit` で起動される。引数文字列: `$ARGUMENTS`

このコマンドは作業ツリーの未コミット変更を分析し、適切な粒度とプロジェクトのプレフィックス規約に沿った 1 つ以上の git コミットを生成する。コミットメッセージはすべて `CLAUDE.md` に従い日本語で書く。

このコマンドは各コミットで意図的に lefthook をバイパスする（`git commit --no-verify`）。これは、複数コミットへの分割中に pre-commit チェック（`make lint` / `make test` / `make sql-lint` / migration チェック）が N 回発火するのを避けるためである。代わりに、全コミットが成功した後、Step 6 で lefthook 定義の各 `pre-commit` コマンドと `make fix` を 1 回の検証パスとして直接実行する。`lefthook run pre-commit` 自体を呼ばないのは、staged が何もないとき（このコマンドが全てをステージ・コミットした後がまさにその状態）lefthook が登録コマンドをスキップするためである。

## Step 0. 自動フォーマット

まず最初に `make fix` を 1 回実行し、フォーマット修正（gofmt / goimports / 自動修正可能な lint ルール）を吸収する。これにより後続の diff 確認における最も一般的なノイズ源が除去され、Step 6 の検証が純粋なフォーマット差分で失敗する可能性が下がる。

```sh
make fix
```

`make fix` 自体が失敗した場合は中止し、失敗をユーザーに報告する。続行しない。`make fix` が生成した変更は作業ツリーに畳み込まれ、Step 2 で確認する候補変更セットの一部となる。

## Step 1. 事前チェック

以下を並列で実行する:

```sh
git rev-parse --abbrev-ref HEAD                      # 現在のブランチ
git rev-parse HEAD                                   # 現在の HEAD コミット（ORIGINAL_HEAD として保存）
git status --porcelain                               # staged + unstaged
git diff --shortstat                                 # unstaged サマリ
git diff --staged --shortstat                        # staged サマリ
git rev-parse --verify MERGE_HEAD 2>/dev/null        # 進行中の merge を検出
git rev-parse --verify CHERRY_PICK_HEAD 2>/dev/null  # 進行中の cherry-pick を検出
git rev-parse --verify REBASE_HEAD 2>/dev/null       # 進行中の rebase を検出
```

現在の HEAD コミットハッシュを `ORIGINAL_HEAD` として保存する。これは Step 5 中に何か失敗したときのロールバック先である。

以下のいずれかに該当する場合は中止する（コミットしない）:

- 現在のブランチが `^(production|develop|staging|release/.+)$` に一致する。`CLAUDE.md` の git ルールにより、保護ブランチには決してコミットしない。ユーザーに知らせ、先に feature ブランチ（例: `feature/<issue-or-topic>`）を作るよう促す。
- staged / unstaged の porcelain 出力が両方とも空。コミットするものが無い旨を伝えて停止する。
- `MERGE_HEAD` / `CHERRY_PICK_HEAD` / `REBASE_HEAD` のいずれかが設定されている。リポジトリが操作の途中なので、先にそれを解決するよう促す。

### マージ済み PR チェック（現在ブランチの PR が既にマージ済みなら新ブランチを推奨）

保護ブランチの中止判定を通過した後、現在のブランチに関連する pull request が既に **マージ済み** かどうかを確認する。PR がマージ済みのブランチに新規コミットを積むのはほぼ確実に意図しない操作である — そのコミットは base に流れない死んだブランチに積み上がり、後続の `submit-pr` はマージ済み PR を再オープン / 更新しようとしてしまう。

以下を実行する（gh CLI。`gh` が無い / 未認証 / remote が無い場合はグレースフルに劣化 — その場合はこのチェックをスキップして続行する）:

```sh
gh pr view --json number,state,mergedAt,baseRefName,headRefName,url 2>/dev/null
```

結果の解釈:

- **PR が見つからない、または `gh` が使えない** → 通常どおり続行（何もしない）。
- **`state` が `OPEN`** → 通常の「既存 PR ブランチ」ケース。続行する。Step 7 が PR ブランチに対する push 前確認ルールを既に強制している。
- **`state` が `MERGED`**（または `mergedAt` が非 null）→ コミット前に停止し、`AskUserQuestion` で（最新の）base から新ブランチを切ることを推奨する:
  - 質問: 「現在のブランチ `<headRefName>` は PR #`<number>` が既にマージ済みです。このままコミットすると、base に流れない死んだブランチに積み増しになります。新しいブランチを切って作業しますか？」
  - 選択肢:
    - 「新しいブランチを切る（推奨）」 — 保留中の変更に由来するブランチ名（例: `feature/<topic>`）を提案・確認し、base を更新して切り替える:

      ```sh
      git fetch origin <baseRefName>
      git switch -c <new-branch> origin/<baseRefName>
      ```

      `<baseRefName>` は **アクティブなリリースライン** である — 本リポジトリは feature 作業を最新の `release/v1.X.0` を base にしており、これは GitHub のデフォルトブランチではない（デフォルトは現行リリースより遅れている）。`defaultBranchRef` よりマージ済み PR の `baseRefName` を信頼する。`git switch -c … origin/release/*` は新ブランチの upstream を **保護された** base に設定するため、最終的な push は明示 refspec（`git push -u origin <new-branch>`）を使い、bare `git push`（保護 base を対象にしてしまう）は決して使わないこと。未コミットの作業ツリー変更は新ブランチへ持ち越されるので、そのまま通常フロー（Step 2 以降）を続ける。**例外:** `--dry-run` ではブランチを切り替えず、警告と推奨コマンドを提示するだけにして、dry-run の提案に進む。
    - 「このブランチのまま続ける」 — ユーザーがマージ済みブランチへのコミットを受け入れる場合は、現在のブランチのまま続行する。
- **`state` が `CLOSED`**（マージされずクローズ）→ ブロックはしないが、一度ユーザーに知らせて（ブランチの PR はクローズ済み）続行する。

`.lefthook.yaml`（あれば）を読み、`pre-commit:` のコマンドエントリ一覧を抽出する。この一覧は 2 か所で使う: (a) 分割中にスキップされる内容をユーザーに知らせるため Step 4 で表示、(b) コミット後の検証ゲートとして Step 6 で直接実行。`.lefthook.yaml` が無い場合はその旨を記録して続行する（Step 6 は `make fix` のみの実行にフォールバックする）。

`$ARGUMENTS` をパースする:

| フラグ | 効果 |
| --- | --- |
| `--dry-run` | 分割の提案のみ生成し、ステージ・コミットはしない。 |
| `--scope=staged` | 現在 staged の変更のみを対象にする。 |
| `--scope=all` | staged / unstaged 両方を対象にする（既定）。 |

## Step 2. 変更の確認

各変更の性質を理解するため、詳細な diff を収集する:

```sh
git diff --staged                     # staged の全 diff
git diff                              # unstaged の全 diff
git diff --staged --name-only
git diff --name-only
```

以下は **rider ファイル** として扱う — それ単独でコミットを構成せず、それを生成したソース変更に相乗りする:

- 生成ファイル: `**/*.gen.go`、`**/*.sql.go`、`*_mock.go`、`**/openapi.gen.yaml`、`docs/portal/links/implements/` 配下の生成物
- vendored コンテンツ: `vendor/**`

例: `openapi/**/*.yaml` の変更は、その `*.gen.go` 出力を同じコミットに持ち込む。`database/dml/**/*.sql` の変更は、その `internal/infrastructure/rdb/sqlc/gen/*.gen.go` 出力を同じコミットに持ち込む。

## Step 3. プレフィックス一覧

コミットごとに、以下のいずれか **1 つ** のプレフィックスを使う（大文字始まり・英語・コロン付き）:

| プレフィックス | 用途 | 例 |
| --- | --- | --- |
| `Feat:` | 新機能・新エンドポイント・新マイグレーション | 新ハンドラ、`openapi/` の新 API、`database/migrations/` 配下の新 SQL |
| `Fix:` | バグ修正（意図から逸脱した挙動の是正） | エラーハンドリング修正、ロジック修正 |
| `Refactor:` | 外部挙動を変えない内部整理 | 関数分割、リネーム、責務移動、レイヤー再編 |
| `Perf:` | パフォーマンス改善 | クエリ最適化、N+1 解消、アロケーション削減 |
| `Docs:` | ドキュメント変更 | `README*`、`docs/`、`*.ja.md`、コードコメント、リリースノート |
| `Test:` | テストの追加・修正 | `*_test.go`、テストフィクスチャ、テストヘルパー |
| `Build:` | ビルドシステム・依存・ツール | `Dockerfile`、`go.mod` / `go.sum`、`Makefile`、`.makefiles/**`、`tools.yaml` |
| `CI:` | CI/CD 設定 | `.github/workflows/**`、`lefthook.yml`、GitHub Actions 関連 |
| `Chore:` | 雑多な作業 | `.gitignore`、エディタ設定、`.claude/**`、その他の小タスク |
| `Style:` | ロジックに影響しないフォーマットのみの変更 | `make fix` / `gofmt` / `goimports` の出力 |
| `Revert:` | 既存コミットの取り消し | `git revert` の出力、または同等の手動 revert |

この一覧外のプレフィックスを作らない。曖昧なときは最も近いものを選ぶ（多くは `Feat` / `Fix` / `Refactor` のいずれか）。

### パスベースのヒント

| パスパターン | 候補プレフィックス |
| --- | --- |
| `internal/**/*.go`（非テスト） | `Feat` / `Fix` / `Refactor` / `Perf`（diff から判断） |
| `**/*_test.go` | `Test` |
| `openapi/**/*.yaml` | `Feat`（API 変更） |
| `database/migrations/**/*.sql` | `Feat`（スキーマ変更） |
| `database/dml/**/*.sql` | `Feat` / `Refactor`（新クエリ vs 整理） |
| `docs/**/*.md`、`README*.md`、`*.ja.md` | `Docs` |
| `Dockerfile`、`docker/**`、`go.mod`、`go.sum`、`Makefile`、`.makefiles/**`、`tools.yaml` | `Build` |
| `.github/workflows/**`、`lefthook.yml`、`.lefthook.yaml` | `CI` |
| `.gitignore`、`.claude/**`、エディタ設定 | `Chore` |

## Step 4. コミット分割の提案

適切な粒度で提案コミットの一覧を作る。各項目:

```txt
[N] <Prefix>: <短い日本語タイトル>
    files:
      - path/to/file1
      - path/to/file2
    rationale: <これらを 1 コミットにまとめる理由>
```

### 粒度のガイドライン

- **1 つの意味的変更 = 1 コミット。** feature + refactor + fix を 1 コミットに混ぜない。
- **テストはカバーする実装と同居してよい**（新ハンドラとそのテストは同じコミット）。既存コードへのテスト追加のみなら、単独の `Test:` コミットにする。
- **生成物はソース変更と同居する。** `openapi/*.yaml` が変わったら、再生成された `*.gen.go` は同じコミットに属す。`make gen-api` / `make gen-query` の出力も同じルール。
- **フォーマットのみの変更は単独の `Style:` コミット。** Step 0 の `make fix` が生成した出力は、同じ変更の一部であることが明確なら適切な既存グループに畳み込んでよい。無関係なら独立した `Style:` コミットとして提示する。
- **`Docs:` は既定で単独。** 例外: ドキュメントが新機能の一部である場合（例: 新パッケージに添える README）は同居してよい。
- **1 コミット 1 プレフィックス。** 2 つ書きたくなったら、分割が間違っている。

### lefthook 通知

分割の提案とあわせて、コミットフェーズ中は **スキップ** されるが Step 6 で検証ゲートとして **直接実行** される lefthook コマンドを表示する。`.lefthook.yaml` から動的に読む（この一覧は設定であり、ハードコードしない）。現在の設定が lint/test/sql-lint/migration チェックを定義している場合の出力例:

```txt
This command will run `git commit --no-verify` on every commit.
The following lefthook pre-commit commands will be SKIPPED during commits but
EXECUTED automatically in Step 6 (verification) after all commits succeed:
  - lint                    (make lint)
  - test                    (make test)
  - sql-lint                (make sql-lint)
  - migration-check-version (make check-migration-up-version check-migration-down-version)
  - migration-check-gap     (make check-migration-up-gap check-migration-down-gap)
Plus `make fix` as a final formatting pass.
```

### 確認

`AskUserQuestion` で提案を確認する:

- 質問: 「提案したコミット分割でよいですか？」
- 選択肢: 「この提案で進める」 / 「修正したい箇所を指摘する」

`--dry-run` が設定されている場合は提案を表示して停止する。ステージ・コミットはしない。

## Step 5. 各コミットの実行

承認された各グループについて、以下を順に実行する:

```sh
# このグループに属すファイルのみをステージ（-A / . は決して使わない）
git add path/to/file1 path/to/file2

# HEREDOC が必須（タイトル / 空行 / 本文 / フッターのレイアウトを保つ）。
# --no-verify は意図的: lefthook は設計上バイパスされる（Step 4 の通知参照）。
git commit --no-verify -m "$(cat <<'EOF'
<Prefix>: <短い日本語タイトル>

<任意の本文: 何を・なぜ変えたか>

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>
EOF
)"
```

### コミットメッセージ規則

- **タイトル**: `<Prefix>: <日本語タイトル>`。50 文字以内を目安。
- **本文**: 任意。ある場合はタイトルの後に空行を 1 つ入れ、72 文字前後で折り返す。「何を」より「なぜ」を優先。
- **言語**: 日本語（`CLAUDE.md` の出力ルールに従う）。
- **`Co-Authored-By` フッター**: 必須。`Claude Opus 4.8 (1M context) <noreply@anthropic.com>` を使う。
- **`Refs:` フッター（レビュー適用コミットのみ）**: コミットが `full-apply` / `local-review` / `code-review` の指摘を適用する場合（変更がレビュー台帳に遡れる場合）、フッターに `Refs: <reviews-dir>/mod_*.md (<severity>)` 行を追加し、コミットを指摘にリンクする。通常のコミットでは省略する。
- **HEREDOC**: 必須（タイトル + 空行 + 本文 + フッターのレイアウトを保つ）。
- **`--no-verify`**: このコマンドが生成する全コミットで必須。これはプロジェクト全体のルールに対する、コマンド限定の明示的な例外である。理由は Step 4 に記載（lefthook は分割中に N 回ではなく、push 前に手動で 1 回実行する）。
- **`-a` / `git add -A` / `git add .` を決して使わない。** 常にファイルを名前でステージする（`.env` や認証情報の巻き込みを避ける）。
- **`--no-gpg-sign` と `--amend` は引き続き禁止**（`--amend` は `permissions.deny` によりハードブロックもされている）。直前のコミットを修正するには、`git reset`（mixed — 決して `--hard` は使わない）で全てを unstage し、意図したファイルを再度 `git add` して再コミットする。bare な `git reset --soft` は index に古いエントリを残しうるので、mixed reset + 明示的な再 add を優先する。

### エラーハンドリング

いずれかのグループで `git add` / `git commit` が失敗した場合（ファイルパスの打ち間違い、事前チェックをすり抜けた操作途中状態、GPG 署名失敗など）:

1. 直ちに以降のコミットを止める。次のグループに進まない。
2. ユーザーに報告する:
   - 失敗したグループ（`[k]` インデックスと提案タイトル）
   - 失敗コマンドの stderr
   - このセッションで既に作成したコミット: `git log --oneline <ORIGINAL_HEAD>..HEAD`
3. `AskUserQuestion` で復旧方法を尋ねる:
   - 質問: 「ここまでに作成したコミットをどうしますか？」
   - 選択肢:
     - 「ロールバックする (`git reset --mixed <ORIGINAL_HEAD>`)」 — HEAD を保存した `ORIGINAL_HEAD` まで巻き戻し、全変更を作業ツリーに残し、index をクリアする
     - 「そのまま残して停止する」 — 部分的なコミットを残し、ユーザーに制御を戻す
4. ユーザーがロールバックを選んだら `git reset --mixed <ORIGINAL_HEAD>` を実行し、`git status` と `git log --oneline -n 3` で確認する。`--hard` は決して使わない。

## Step 6. 検証

全コミットが成功した後、(a) `.lefthook.yaml` の `pre-commit:` `commands:` で定義された各コマンドと (b) 最終フォーマットパスとしての `make fix` から成る検証ゲートを実行する。`lefthook run pre-commit` 自体は実行しない — staged が何もないとき（コミット後の状態）lefthook は登録コマンドをスキップし、「no matching staged files」と報告して何も検査せず終了するためである。代わりに各コマンドを直接実行する。

### 手順

1. `.lefthook.yaml` を再読し、`pre-commit.commands.*.run` の値を列挙する。`.lefthook.yaml` が無ければこのステップをスキップする。
2. 各コマンドを **順次** 実行する（並列より出力が明確で、失敗時にどのステップが失敗したか分かる）。各々について終了ステータスと出力の短い末尾を記録する。
3. lefthook 定義の全コマンドが終わったら `make fix` を実行する。`make fix` が追跡ファイルを変更したら、その diff をユーザーに提示する — コミット状態が完全にフォーマットされていなかったことを示すので、それらの修正をステージ・コミットするかはユーザーが判断する。
4. 結果を表形式でユーザーに要約する:

   ```txt
   検証コマンドの実行結果:
     - make lint                                                       → OK / FAIL
     - make test                                                       → OK / FAIL
     - make sql-lint                                                   → OK / FAIL
     - make check-migration-up-version check-migration-down-version    → OK / FAIL
     - make check-migration-up-gap check-migration-down-gap            → OK / FAIL
     - make fix                                                        → no changes / changes detected
   ```

5. いずれかのコマンドが **失敗** したら、失敗サマリ（終了コード + 出力末尾）を報告して停止する。コミットはロールバックしない — 失敗は情報提供であり、fix-up コミットを足すか amend するかはユーザーが判断する。ユーザーに明示的に伝える:

   ```txt
   検証で失敗があります。push 前に修正してください。
   失敗したコマンド: <name> (<command>)
   ```

6. 全コマンドが **成功** し `make fix` が変更を生まなかったら、Step 7 へ進む。

### 検証のスキップ

ユーザーが `/commit` 自体に `--no-verify` を渡した場合（将来互換のフラグ）、または `.lefthook.yaml` が無い場合は、このステップを完全にスキップし Step 7 のレポートにその旨を記す。既定の挙動は検証を実行することである。

## Step 7. push 方針と最終リマインド

- **自動 push はしない**（`CLAUDE.md` の git ルールに従う）。
- Step 6 が終わったら（全チェック成功か否かに関わらず）ユーザーに報告する。テンプレートは検証結果によって変わる:

  全チェック成功時:

  ```txt
  N 件のコミットを作成し、検証コマンドも全て成功しました。
  プッシュは手動で実行してください: `git push`
  ```

  一部チェック失敗時:

  ```txt
  N 件のコミットを作成しましたが、Step 6 の検証で失敗があります。
  失敗内容を修正してから push してください。
  ```

  検証をスキップした時（`.lefthook.yaml` が無い / 明示スキップ）:

  ```txt
  N 件のコミットを作成しました（検証はスキップしました）。
  push 前に手動で動作確認してください。
  ```

- 既存の PR ブランチで作業している場合は `CLAUDE.md` に従い、push 前に尋ねる:
  「変更はローカルにコミット済みです。これらの変更をプルリクエストにプッシュしますか？」

## 制約（サマリ）

- ❌ `production` / `develop` / `staging` / `release/*` ブランチへの直コミット
- ❌ `git push` / `git push --force` / `git reset --hard` / `git checkout --` / `git clean -f` の自動実行
- ❌ `--no-gpg-sign` / `--amend`
- ❌ `git add -A` / `git add .` / `git commit -a`（常にファイルを名前指定）
- ❌ 1 コミットに複数プレフィックスを混在
- ❌ `--no-verify` なしのコミット（lefthook が N 回走ってしまう）
- ✅ 日本語のコミットメッセージ
- ✅ メッセージは HEREDOC
- ✅ `Co-Authored-By` フッター
- ✅ このコマンドが生成する全コミットで `--no-verify`
- ✅ 現グループのファイルのみステージ
- ✅ 確認前の Step 0 で `make fix` を 1 回
- ✅ Step 1 で安全なロールバック用に `ORIGINAL_HEAD` を保存
- ✅ Step 1 で、現在ブランチの PR がマージ済みか検出（`gh pr view`）し、コミット前に base から新ブランチを切ることを推奨（`gh` が使えない場合はグレースフルに劣化）
- ✅ 失敗時は `AskUserQuestion` で `git reset --mixed <ORIGINAL_HEAD>` を提案
- ✅ Step 6 は lefthook 定義の各コマンド + `make fix` を直接実行（`lefthook run pre-commit` は使わない）
- ❌ `lefthook run pre-commit` を呼ばない — staged が無いとき（コミット後の状態）コマンドをスキップするため

## チェックリスト

完了を報告する前に、以下を確認する:

- [ ] Step 0 で `make fix` が成功した
- [ ] いずれのコミット前にも `ORIGINAL_HEAD` を保存した
- [ ] 非保護ブランチでコミットした
- [ ] 現在ブランチの PR がマージ済みか確認し、そうであれば新ブランチを切ることを推奨した（そしてユーザーの選択に従った）
- [ ] リポジトリが merge / rebase / cherry-pick の途中でなかった
- [ ] ユーザーが提案した分割を承認した（`--dry-run` を除く）
- [ ] lefthook スキップ通知を、動的なコマンド一覧とともにユーザーに表示した
- [ ] 各コミットが単一プレフィックス
- [ ] 各コミットメッセージが日本語で、`Co-Authored-By` フッターを含む
- [ ] 各コミットで `--no-verify` を使い、HEREDOC で渡した
- [ ] `git add` はファイルを明示指定（`-A` / `.` なし）
- [ ] 生成物をソース変更と同居させた
- [ ] Step 6 の検証が lefthook 定義の各コマンド + `make fix` を実行した（または明示的にスキップした）
- [ ] 検証結果（OK / FAIL / no changes）をユーザーに提示した
- [ ] 自動 push を行わなかった
